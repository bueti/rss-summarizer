package repository

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/article"
	"github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ArticleRepository interface {
	Create(ctx context.Context, a *article.Article) error
	FindByID(ctx context.Context, id uuid.UUID) (*article.Article, error)
	FindByFeedID(ctx context.Context, feedID uuid.UUID, filters article.ArticleFilters) ([]*article.Article, error)
	FindByUserIDWithState(ctx context.Context, userID uuid.UUID, filters article.ArticleFilters) ([]*article.ArticleWithUserState, error)
	Update(ctx context.Context, a *article.Article) error
	UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status string, processingError *string) error
	ExistsByFeedAndURL(ctx context.Context, feedID uuid.UUID, url string) (bool, error)
	ExistsByEmailMessageID(ctx context.Context, messageID string) (bool, error)
	CountByUserIDWithState(ctx context.Context, userID uuid.UUID, filters article.ArticleFilters) (int, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type articleRepository struct {
	db *database.DB
}

func NewArticleRepository(db *database.DB) ArticleRepository {
	return &articleRepository{db: db}
}

// titleCaseASCII upper-cases the first rune of each whitespace-delimited word.
// strings.Title is deprecated and mishandles non-ASCII casing; we only need
// ASCII behavior here since acronyms and technical terms are handled separately.
func titleCaseASCII(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	capitalizeNext := true
	for _, r := range s {
		if r == ' ' || r == '-' || r == '/' {
			capitalizeNext = true
			b.WriteRune(r)
			continue
		}
		if capitalizeNext && r >= 'a' && r <= 'z' {
			b.WriteRune(r - ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
		capitalizeNext = false
	}
	return b.String()
}

// normalizeTopics deduplicates and normalizes topic names to title case
func normalizeTopics(topics []string) []string {
	if len(topics) == 0 {
		return topics
	}

	// Common acronyms and technical terms that should stay uppercase
	acronyms := map[string]string{
		"ai":      "AI",
		"api":     "API",
		"apis":    "APIs",
		"aws":     "AWS",
		"gcp":     "GCP",
		"devops":  "DevOps",
		"cicd":    "CI/CD",
		"ml":      "ML",
		"llm":     "LLM",
		"llms":    "LLMs",
		"ui":      "UI",
		"ux":      "UX",
		"css":     "CSS",
		"html":    "HTML",
		"json":    "JSON",
		"xml":     "XML",
		"sql":     "SQL",
		"nosql":   "NoSQL",
		"rest":    "REST",
		"graphql": "GraphQL",
		"grpc":    "gRPC",
		"http":    "HTTP",
		"https":   "HTTPS",
		"ssh":     "SSH",
		"tcp":     "TCP",
		"udp":     "UDP",
		"dns":     "DNS",
		"cdn":     "CDN",
		"saas":    "SaaS",
		"paas":    "PaaS",
		"iaas":    "IaaS",
		"oauth":   "OAuth",
		"jwt":     "JWT",
		"tls":     "TLS",
		"ssl":     "SSL",
		"vpn":     "VPN",
	}

	// Use a map to track unique topics (case-insensitive)
	seen := make(map[string]string) // lowercase -> normalized
	var result []string

	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}

		lowerKey := strings.ToLower(topic)

		// Only add if we haven't seen this topic before (case-insensitive)
		if _, exists := seen[lowerKey]; !exists {
			// Check if it's a known acronym
			if acronym, isAcronym := acronyms[lowerKey]; isAcronym {
				seen[lowerKey] = acronym
				result = append(result, acronym)
			} else {
				normalized := titleCaseASCII(strings.ToLower(topic))
				seen[lowerKey] = normalized
				result = append(result, normalized)
			}
		}
	}

	return result
}

func (r *articleRepository) Create(ctx context.Context, a *article.Article) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}

	// Default to pending status if not set
	if a.ProcessingStatus == "" {
		a.ProcessingStatus = article.ProcessingPending
	}

	// Default to "rss" source type if not set
	if a.SourceType == "" {
		a.SourceType = "rss"
	}

	// Normalize topics to deduplicate and standardize casing
	a.Topics = normalizeTopics(a.Topics)

	query := `
		INSERT INTO articles (
		    id, feed_id, email_source_id, title, url, published_at,
		    original_content, full_text, summary, key_points,
		    importance_score, topics, processing_status, processing_error,
		    source_type, email_message_id, created_at, updated_at
		) VALUES (
		    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW()
		)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		a.ID, a.FeedID, a.EmailSourceID, a.Title, a.URL, a.PublishedAt,
		a.OriginalContent, a.FullText, a.Summary, a.KeyPoints,
		a.ImportanceScore, a.Topics, a.ProcessingStatus, a.ProcessingError,
		a.SourceType, a.EmailMessageID,
	).Scan(&a.CreatedAt, &a.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return &errors.DuplicateError{Resource: "article", Field: "url", Value: a.URL}
		}
		return fmt.Errorf("failed to create article: %w", err)
	}

	return nil
}

func (r *articleRepository) FindByID(ctx context.Context, id uuid.UUID) (*article.Article, error) {
	var a article.Article
	query := `SELECT * FROM articles WHERE id = $1`

	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, &errors.NotFoundError{Resource: "article", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to find article: %w", err)
	}

	return &a, nil
}

func (r *articleRepository) FindByFeedID(ctx context.Context, feedID uuid.UUID, filters article.ArticleFilters) ([]*article.Article, error) {
	var articles []*article.Article

	// Build query with filters
	query := `SELECT * FROM articles WHERE feed_id = $1`
	args := []any{feedID}
	argPos := 2

	if filters.MinImportance != nil {
		query += fmt.Sprintf(" AND importance_score >= $%d", argPos)
		args = append(args, *filters.MinImportance)
		argPos++
	}

	if filters.Topic != nil {
		query += fmt.Sprintf(" AND $%d = ANY(topics)", argPos)
		args = append(args, *filters.Topic)
		argPos++
	}

	if filters.ProcessingStatus != nil {
		query += fmt.Sprintf(" AND processing_status = $%d", argPos)
		args = append(args, *filters.ProcessingStatus)
		argPos++
	}

	// Order by date first, then importance
	query += " ORDER BY published_at DESC NULLS LAST, importance_score DESC NULLS LAST"

	// Pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filters.Limit)
		argPos++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filters.Offset)
	}

	if err := r.db.SelectContext(ctx, &articles, query, args...); err != nil {
		return nil, fmt.Errorf("failed to find articles: %w", err)
	}

	return articles, nil
}

func (r *articleRepository) FindByUserIDWithState(ctx context.Context, userID uuid.UUID, filters article.ArticleFilters) ([]*article.ArticleWithUserState, error) {
	var results []*article.ArticleWithUserState

	// Join articles with user_articles to get per-user state
	// Include both RSS articles (via feed subscriptions) and email articles (via email sources)
	query := `
		SELECT
			a.*,
			COALESCE(ua.is_read, false) as is_read,
			COALESCE(ua.is_saved, false) as is_saved,
			COALESCE(ua.is_archived, false) as is_archived
		FROM articles a
		LEFT JOIN user_feed_subscriptions ufs ON ufs.feed_id = a.feed_id AND ufs.user_id = $1
		LEFT JOIN email_sources es ON es.id = a.email_source_id AND es.user_id = $1
		LEFT JOIN user_articles ua ON ua.article_id = a.id AND ua.user_id = $1
		WHERE (
			(a.source_type = 'rss' AND ufs.is_active = true)
			OR (a.source_type = 'email' AND es.id IS NOT NULL AND es.is_active = true)
		)
	`
	args := []any{userID}
	argPos := 2

	if filters.FeedID != nil {
		query += fmt.Sprintf(" AND a.feed_id = $%d", argPos)
		args = append(args, *filters.FeedID)
		argPos++
	}

	if filters.EmailSourceID != nil {
		query += fmt.Sprintf(" AND a.email_source_id = $%d", argPos)
		args = append(args, *filters.EmailSourceID)
		argPos++
	}

	if filters.MinImportance != nil {
		query += fmt.Sprintf(" AND a.importance_score >= $%d", argPos)
		args = append(args, *filters.MinImportance)
		argPos++
	}

	if filters.Topic != nil {
		query += fmt.Sprintf(" AND $%d = ANY(a.topics)", argPos)
		args = append(args, *filters.Topic)
		argPos++
	}

	if filters.IsRead != nil {
		query += fmt.Sprintf(" AND COALESCE(ua.is_read, false) = $%d", argPos)
		args = append(args, *filters.IsRead)
		argPos++
	}

	if filters.IsSaved != nil {
		query += fmt.Sprintf(" AND COALESCE(ua.is_saved, false) = $%d", argPos)
		args = append(args, *filters.IsSaved)
		argPos++
	}

	if filters.IsArchived != nil {
		query += fmt.Sprintf(" AND COALESCE(ua.is_archived, false) = $%d", argPos)
		args = append(args, *filters.IsArchived)
		argPos++
	}

	if filters.ProcessingStatus != nil {
		query += fmt.Sprintf(" AND a.processing_status = $%d", argPos)
		args = append(args, *filters.ProcessingStatus)
		argPos++
	}

	// Dynamic sorting
	if filters.SortBy != nil && *filters.SortBy == "importance" {
		query += " ORDER BY a.importance_score DESC NULLS LAST, a.published_at DESC NULLS LAST"
	} else {
		// Default: sort by date first
		query += " ORDER BY a.published_at DESC NULLS LAST, a.importance_score DESC NULLS LAST"
	}

	// Pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filters.Limit)
		argPos++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filters.Offset)
	}

	if err := r.db.SelectContext(ctx, &results, query, args...); err != nil {
		return nil, fmt.Errorf("failed to find articles with user state: %w", err)
	}

	return results, nil
}

func (r *articleRepository) Update(ctx context.Context, a *article.Article) error {
	// Normalize topics to deduplicate and standardize casing
	a.Topics = normalizeTopics(a.Topics)

	query := `
		UPDATE articles
		SET title = $2, summary = $3, key_points = $4,
		    importance_score = $5, topics = $6, full_text = $7,
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		a.ID, a.Title, a.Summary, a.KeyPoints, a.ImportanceScore, a.Topics, a.FullText,
	)
	if err != nil {
		return fmt.Errorf("failed to update article: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return &errors.NotFoundError{Resource: "article", ID: a.ID.String()}
	}

	return nil
}

func (r *articleRepository) ExistsByFeedAndURL(ctx context.Context, feedID uuid.UUID, url string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM articles WHERE feed_id = $1 AND url = $2)`

	if err := r.db.GetContext(ctx, &exists, query, feedID, url); err != nil {
		return false, fmt.Errorf("failed to check article existence: %w", err)
	}

	return exists, nil
}

func (r *articleRepository) ExistsByEmailMessageID(ctx context.Context, messageID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM articles WHERE email_message_id = $1)`

	if err := r.db.GetContext(ctx, &exists, query, messageID); err != nil {
		return false, fmt.Errorf("failed to check article existence by email message ID: %w", err)
	}

	return exists, nil
}

func (r *articleRepository) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status string, processingError *string) error {
	query := `
		UPDATE articles
		SET processing_status = $2, processing_error = $3, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status, processingError)
	if err != nil {
		return fmt.Errorf("failed to update processing status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return &errors.NotFoundError{Resource: "article", ID: id.String()}
	}

	return nil
}

func (r *articleRepository) CountByUserIDWithState(ctx context.Context, userID uuid.UUID, filters article.ArticleFilters) (int, error) {
	var count int

	// Count articles for user (both RSS and email sources)
	query := `
		SELECT COUNT(*)
		FROM articles a
		LEFT JOIN user_feed_subscriptions ufs ON ufs.feed_id = a.feed_id AND ufs.user_id = $1
		LEFT JOIN email_sources es ON es.id = a.email_source_id AND es.user_id = $1
		LEFT JOIN user_articles ua ON ua.article_id = a.id AND ua.user_id = $1
		WHERE (
			(a.source_type = 'rss' AND ufs.is_active = true)
			OR (a.source_type = 'email' AND es.id IS NOT NULL AND es.is_active = true)
		)
	`
	args := []any{userID}
	argPos := 2

	if filters.FeedID != nil {
		query += fmt.Sprintf(" AND a.feed_id = $%d", argPos)
		args = append(args, *filters.FeedID)
		argPos++
	}

	if filters.EmailSourceID != nil {
		query += fmt.Sprintf(" AND a.email_source_id = $%d", argPos)
		args = append(args, *filters.EmailSourceID)
		argPos++
	}

	if filters.MinImportance != nil {
		query += fmt.Sprintf(" AND a.importance_score >= $%d", argPos)
		args = append(args, *filters.MinImportance)
		argPos++
	}

	if filters.Topic != nil {
		query += fmt.Sprintf(" AND $%d = ANY(a.topics)", argPos)
		args = append(args, *filters.Topic)
		argPos++
	}

	if filters.IsRead != nil {
		query += fmt.Sprintf(" AND COALESCE(ua.is_read, false) = $%d", argPos)
		args = append(args, *filters.IsRead)
		argPos++
	}

	if filters.IsSaved != nil {
		query += fmt.Sprintf(" AND COALESCE(ua.is_saved, false) = $%d", argPos)
		args = append(args, *filters.IsSaved)
		argPos++
	}

	if filters.IsArchived != nil {
		query += fmt.Sprintf(" AND COALESCE(ua.is_archived, false) = $%d", argPos)
		args = append(args, *filters.IsArchived)
		argPos++
	}

	if filters.ProcessingStatus != nil {
		query += fmt.Sprintf(" AND a.processing_status = $%d", argPos)
		args = append(args, *filters.ProcessingStatus)
	}

	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count articles: %w", err)
	}

	return count, nil
}

func (r *articleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM articles WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete article: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return &errors.NotFoundError{Resource: "article", ID: id.String()}
	}

	return nil
}
