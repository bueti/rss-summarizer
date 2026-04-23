package repository

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"

	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/bbu/rss-summarizer/backend/internal/domain/user_article"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type UserArticleRepository interface {
	// Upsert creates or updates user article state
	Upsert(ctx context.Context, userID, articleID uuid.UUID, isRead bool) error

	// FindByUserAndArticle gets user state for a specific article
	FindByUserAndArticle(ctx context.Context, userID, articleID uuid.UUID) (*user_article.UserArticle, error)

	// FindByUserID gets all user article states for a user
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*user_article.UserArticle, error)

	// MarkAsRead updates read status (includes auto-archive logic)
	MarkAsRead(ctx context.Context, userID, articleID uuid.UUID, isRead bool) error

	// MarkReadBulk bulk updates read status (includes auto-archive logic)
	MarkReadBulk(ctx context.Context, userID uuid.UUID, articleIDs []uuid.UUID, isRead bool) error

	// SetSaved updates saved status
	SetSaved(ctx context.Context, userID, articleID uuid.UUID, isSaved bool) error

	// SetSavedBulk bulk updates saved status
	SetSavedBulk(ctx context.Context, userID uuid.UUID, articleIDs []uuid.UUID, isSaved bool) error

	// SetArchived updates archived status
	SetArchived(ctx context.Context, userID, articleID uuid.UUID, isArchived bool) error

	// SetArchivedBulk bulk updates archived status
	SetArchivedBulk(ctx context.Context, userID uuid.UUID, articleIDs []uuid.UUID, isArchived bool) error

	// DeleteByArticle removes all user states for an article (when article deleted)
	DeleteByArticle(ctx context.Context, articleID uuid.UUID) error
}

type userArticleRepository struct {
	db *database.DB
}

func NewUserArticleRepository(db *database.DB) UserArticleRepository {
	return &userArticleRepository{db: db}
}

func (r *userArticleRepository) Upsert(ctx context.Context, userID, articleID uuid.UUID, isRead bool) error {
	query := `
		INSERT INTO user_articles (id, user_id, article_id, is_read, created_at, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, article_id)
		DO UPDATE SET is_read = EXCLUDED.is_read, updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, userID, articleID, isRead)
	if err != nil {
		return fmt.Errorf("failed to upsert user_article: %w", err)
	}

	return nil
}

func (r *userArticleRepository) FindByUserAndArticle(ctx context.Context, userID, articleID uuid.UUID) (*user_article.UserArticle, error) {
	var ua user_article.UserArticle
	query := `SELECT * FROM user_articles WHERE user_id = $1 AND article_id = $2`

	if err := r.db.GetContext(ctx, &ua, query, userID, articleID); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, &errors.NotFoundError{Resource: "user_article", ID: fmt.Sprintf("user=%s,article=%s", userID, articleID)}
		}
		return nil, fmt.Errorf("failed to find user_article: %w", err)
	}

	return &ua, nil
}

func (r *userArticleRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*user_article.UserArticle, error) {
	var uas []*user_article.UserArticle
	query := `SELECT * FROM user_articles WHERE user_id = $1 ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &uas, query, userID); err != nil {
		return nil, fmt.Errorf("failed to find user_articles: %w", err)
	}

	return uas, nil
}

func (r *userArticleRepository) MarkAsRead(ctx context.Context, userID, articleID uuid.UUID, isRead bool) error {
	// Auto-archive logic:
	// - If marking as read AND article is not saved → archive it
	// - If unmarking as read → un-archive it
	query := `
		INSERT INTO user_articles (id, user_id, article_id, is_read, is_saved, is_archived, created_at, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, false, $3, NOW(), NOW())
		ON CONFLICT (user_id, article_id)
		DO UPDATE SET
			is_read = EXCLUDED.is_read,
			is_archived = CASE
				WHEN EXCLUDED.is_read = true AND user_articles.is_saved = false THEN true
				WHEN EXCLUDED.is_read = false THEN false
				ELSE user_articles.is_archived
			END,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, userID, articleID, isRead)
	if err != nil {
		return fmt.Errorf("failed to mark article as read: %w", err)
	}

	return nil
}

func (r *userArticleRepository) MarkReadBulk(ctx context.Context, userID uuid.UUID, articleIDs []uuid.UUID, isRead bool) error {
	if len(articleIDs) == 0 {
		return nil
	}

	// Build batch upsert query with auto-archive logic
	query := `
		INSERT INTO user_articles (id, user_id, article_id, is_read, is_saved, is_archived, created_at, updated_at)
		SELECT uuid_generate_v4(), $1, unnest($2::uuid[]), $3, false, $3, NOW(), NOW()
		ON CONFLICT (user_id, article_id)
		DO UPDATE SET
			is_read = EXCLUDED.is_read,
			is_archived = CASE
				WHEN EXCLUDED.is_read = true AND user_articles.is_saved = false THEN true
				WHEN EXCLUDED.is_read = false THEN false
				ELSE user_articles.is_archived
			END,
			updated_at = NOW()
	`

	// Convert []uuid.UUID to pq.Array for PostgreSQL
	_, err := r.db.ExecContext(ctx, query, userID, pq.Array(articleIDsToStringArray(articleIDs)), isRead)
	if err != nil {
		return fmt.Errorf("failed to bulk mark articles as read: %w", err)
	}

	return nil
}

func (r *userArticleRepository) SetSaved(ctx context.Context, userID, articleID uuid.UUID, isSaved bool) error {
	// When unsaving a read article, it should auto-archive
	query := `
		INSERT INTO user_articles (id, user_id, article_id, is_saved, created_at, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, article_id)
		DO UPDATE SET
			is_saved = EXCLUDED.is_saved,
			is_archived = CASE
				WHEN EXCLUDED.is_saved = false AND user_articles.is_read = true THEN true
				ELSE user_articles.is_archived
			END,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, userID, articleID, isSaved)
	if err != nil {
		return fmt.Errorf("failed to set article saved status: %w", err)
	}

	return nil
}

func (r *userArticleRepository) SetSavedBulk(ctx context.Context, userID uuid.UUID, articleIDs []uuid.UUID, isSaved bool) error {
	if len(articleIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO user_articles (id, user_id, article_id, is_saved, created_at, updated_at)
		SELECT uuid_generate_v4(), $1, unnest($2::uuid[]), $3, NOW(), NOW()
		ON CONFLICT (user_id, article_id)
		DO UPDATE SET
			is_saved = EXCLUDED.is_saved,
			is_archived = CASE
				WHEN EXCLUDED.is_saved = false AND user_articles.is_read = true THEN true
				ELSE user_articles.is_archived
			END,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, userID, pq.Array(articleIDsToStringArray(articleIDs)), isSaved)
	if err != nil {
		return fmt.Errorf("failed to bulk set articles saved status: %w", err)
	}

	return nil
}

func (r *userArticleRepository) SetArchived(ctx context.Context, userID, articleID uuid.UUID, isArchived bool) error {
	query := `
		INSERT INTO user_articles (id, user_id, article_id, is_archived, created_at, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, article_id)
		DO UPDATE SET is_archived = EXCLUDED.is_archived, updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, userID, articleID, isArchived)
	if err != nil {
		return fmt.Errorf("failed to set article archived status: %w", err)
	}

	return nil
}

func (r *userArticleRepository) SetArchivedBulk(ctx context.Context, userID uuid.UUID, articleIDs []uuid.UUID, isArchived bool) error {
	if len(articleIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO user_articles (id, user_id, article_id, is_archived, created_at, updated_at)
		SELECT uuid_generate_v4(), $1, unnest($2::uuid[]), $3, NOW(), NOW()
		ON CONFLICT (user_id, article_id)
		DO UPDATE SET is_archived = EXCLUDED.is_archived, updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, userID, pq.Array(articleIDsToStringArray(articleIDs)), isArchived)
	if err != nil {
		return fmt.Errorf("failed to bulk set articles archived status: %w", err)
	}

	return nil
}

func (r *userArticleRepository) DeleteByArticle(ctx context.Context, articleID uuid.UUID) error {
	query := `DELETE FROM user_articles WHERE article_id = $1`

	_, err := r.db.ExecContext(ctx, query, articleID)
	if err != nil {
		return fmt.Errorf("failed to delete user_articles: %w", err)
	}

	return nil
}

// Helper function to convert []uuid.UUID to []string for PostgreSQL array
func articleIDsToStringArray(ids []uuid.UUID) []string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	return strs
}
