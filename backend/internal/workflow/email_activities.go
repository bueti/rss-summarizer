package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/domain/article"
	"github.com/bbu/rss-summarizer/backend/internal/domain/email_source"
	"github.com/bbu/rss-summarizer/backend/internal/domain/newsletter_filter"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/bbu/rss-summarizer/backend/internal/service/email"
	"github.com/bbu/rss-summarizer/backend/internal/service/gmail"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

type EmailActivities struct {
	emailSourceRepo email_source.Repository
	filterRepo      newsletter_filter.Repository
	articleRepo     repository.ArticleRepository
	userArticleRepo repository.UserArticleRepository
	gmailService    *gmail.Service
}

// GetActiveEmailSourcesActivity returns all active email sources
func (a *EmailActivities) GetActiveEmailSourcesActivity(ctx context.Context) ([]uuid.UUID, error) {
	sources, err := a.emailSourceRepo.FindAllActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find active email sources: %w", err)
	}

	sourceIDs := make([]uuid.UUID, len(sources))
	for i, source := range sources {
		sourceIDs[i] = source.ID
	}

	return sourceIDs, nil
}

func NewEmailActivities(
	emailSourceRepo email_source.Repository,
	filterRepo newsletter_filter.Repository,
	articleRepo repository.ArticleRepository,
	userArticleRepo repository.UserArticleRepository,
	gmailService *gmail.Service,
) *EmailActivities {
	return &EmailActivities{
		emailSourceRepo: emailSourceRepo,
		filterRepo:      filterRepo,
		articleRepo:     articleRepo,
		userArticleRepo: userArticleRepo,
		gmailService:    gmailService,
	}
}

type FetchEmailsInput struct {
	EmailSourceID uuid.UUID
}

type FetchEmailsOutput struct {
	NewArticleIDs []uuid.UUID
	ErrorMessage  *string
}

// FetchEmailsActivity fetches emails from an email source and creates articles
func (a *EmailActivities) FetchEmailsActivity(ctx context.Context, input FetchEmailsInput) (*FetchEmailsOutput, error) {
	// Get email source
	source, err := a.emailSourceRepo.FindByID(ctx, input.EmailSourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find email source: %w", err)
	}

	if !source.IsActive {
		return &FetchEmailsOutput{NewArticleIDs: []uuid.UUID{}}, nil
	}

	// Get active filters for this email source
	filters, err := a.filterRepo.FindActiveByEmailSourceID(ctx, input.EmailSourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find filters: %w", err)
	}

	if len(filters) == 0 {
		// No filters configured, nothing to do
		return &FetchEmailsOutput{NewArticleIDs: []uuid.UUID{}}, nil
	}

	// Build Gmail query from filters
	query := a.buildGmailQuery(filters)

	// Create token with expiry so OAuth library knows when to refresh
	currentToken := &oauth2.Token{
		AccessToken:  source.AccessToken,
		RefreshToken: source.RefreshToken,
		Expiry:       source.TokenExpiresAt,
	}

	// Fetch emails from Gmail (will auto-refresh if expired)
	emails, newToken, err := a.gmailService.FetchEmailsWithToken(
		ctx,
		currentToken,
		query,
		50, // Max 50 emails per fetch
	)
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch emails: %v", err)
		a.updateEmailSourceError(ctx, source, err)
		return &FetchEmailsOutput{ErrorMessage: &errMsg}, fmt.Errorf("failed to fetch emails: %w", err)
	}

	// Update tokens if refreshed
	if newToken.AccessToken != source.AccessToken {
		updateInput := &email_source.UpdateEmailSourceInput{
			AccessToken:    &newToken.AccessToken,
			TokenExpiresAt: &newToken.Expiry,
		}
		if newToken.RefreshToken != "" && newToken.RefreshToken != source.RefreshToken {
			updateInput.RefreshToken = &newToken.RefreshToken
		}
		if _, err := a.emailSourceRepo.Update(ctx, source.ID, updateInput); err != nil {
			log.Error().Err(err).Str("email_source_id", source.ID.String()).Msg("Failed to update email source tokens")
		}
	}

	var newArticleIDs []uuid.UUID
	var itemErrors int

	// Process each email. See FetchFeedActivity for the rationale on not
	// pre-seeding user_articles; the same LEFT JOIN handles defaults here.
	for _, msg := range emails {
		if !a.matchesFilters(msg, filters) {
			continue
		}

		exists, err := a.articleRepo.ExistsByEmailMessageID(ctx, msg.ID)
		if err != nil {
			log.Error().Err(err).Str("message_id", msg.ID).Msg("Failed to check article existence")
			itemErrors++
			continue
		}

		if exists {
			newToken, err = a.gmailService.MarkAsReadWithToken(ctx, newToken, msg.ID)
			if err != nil {
				log.Error().Err(err).Str("message_id", msg.ID).Msg("Failed to mark existing email as read")
			}
			continue
		}

		content, err := email.ParseEmailContent(msg.BodyHTML, msg.BodyPlain)
		if err != nil {
			log.Error().Err(err).Str("message_id", msg.ID).Msg("Failed to parse email content")
			itemErrors++
			continue
		}

		if content == "" {
			log.Warn().Str("message_id", msg.ID).Msg("No content extracted from email")
			continue
		}

		art := &article.Article{
			ID:               uuid.New(),
			FeedID:           nil,
			EmailSourceID:    &source.ID,
			Title:            msg.Subject,
			URL:              "",
			PublishedAt:      &msg.Date,
			OriginalContent:  content,
			SourceType:       "email",
			EmailMessageID:   &msg.ID,
			ProcessingStatus: article.ProcessingPending,
		}

		if err := a.articleRepo.Create(ctx, art); err != nil {
			log.Error().Err(err).Str("message_id", msg.ID).Msg("Failed to create article from email")
			itemErrors++
			continue
		}

		newArticleIDs = append(newArticleIDs, art.ID)

		newToken, err = a.gmailService.MarkAsReadWithToken(ctx, newToken, msg.ID)
		if err != nil {
			log.Error().Err(err).Str("message_id", msg.ID).Msg("Failed to mark email as read")
		}
	}

	// Update last fetched time, tokens, and clear any previous errors
	now := time.Now()
	emptyError := ""
	updateInput := &email_source.UpdateEmailSourceInput{
		LastFetchedAt: &now,
		LastError:     &emptyError, // Clear previous errors on success
	}

	// Update tokens if they were refreshed during mark-as-read operations
	if newToken.AccessToken != source.AccessToken {
		updateInput.AccessToken = &newToken.AccessToken
		updateInput.TokenExpiresAt = &newToken.Expiry
		if newToken.RefreshToken != "" && newToken.RefreshToken != source.RefreshToken {
			updateInput.RefreshToken = &newToken.RefreshToken
		}
	}

	if _, err := a.emailSourceRepo.Update(ctx, source.ID, updateInput); err != nil {
		log.Error().Err(err).Str("email_source_id", source.ID.String()).Msg("Failed to update email source")
	}

	if itemErrors > 0 && len(newArticleIDs) == 0 {
		return nil, fmt.Errorf("fetch emails from %s: %d messages failed and none succeeded", source.ID, itemErrors)
	}

	return &FetchEmailsOutput{NewArticleIDs: newArticleIDs}, nil
}

// buildGmailQuery builds a Gmail search query from filters
func (a *EmailActivities) buildGmailQuery(filters []*newsletter_filter.NewsletterFilter) string {
	var parts []string

	// Add sender patterns from all filters
	for _, filter := range filters {
		if filter.SenderPattern != "" {
			// Convert pattern to Gmail query format
			if after, ok := strings.CutPrefix(filter.SenderPattern, "*@"); ok {
				// Domain wildcard: *@substack.com -> from:@substack.com
				domain := after
				parts = append(parts, fmt.Sprintf("from:@%s", domain))
			} else {
				// Exact or other patterns
				parts = append(parts, fmt.Sprintf("from:%s", filter.SenderPattern))
			}
		}
	}

	// Combine sender patterns with OR
	query := "(" + strings.Join(parts, " OR ") + ")"

	// Only fetch unread emails - these will be marked as read after processing
	query += " is:unread"

	return query
}

// matchesFilters checks if an email matches any of the filters
func (a *EmailActivities) matchesFilters(msg *gmail.EmailMessage, filters []*newsletter_filter.NewsletterFilter) bool {
	senderEmail := email.ExtractSenderEmail(msg.From)

	for _, filter := range filters {
		// Check sender pattern
		if !email.MatchesSenderPattern(senderEmail, filter.SenderPattern) {
			continue
		}

		// Check subject pattern if specified
		if !email.MatchesSubjectPattern(msg.Subject, filter.SubjectPattern) {
			continue
		}

		// All patterns match
		return true
	}

	return false
}

// updateEmailSourceError increments error count and updates status
func (a *EmailActivities) updateEmailSourceError(ctx context.Context, source *email_source.EmailSource, err error) {
	errMsg := err.Error()
	isActive := source.IsActive

	// Disable source if error is due to revoked access
	if strings.Contains(strings.ToLower(errMsg), "invalid_grant") ||
		strings.Contains(strings.ToLower(errMsg), "revoked") ||
		strings.Contains(strings.ToLower(errMsg), "unauthorized") {
		isActive = false
		errMsg = "Access revoked. Please reconnect your Gmail account."
	}

	updateInput := &email_source.UpdateEmailSourceInput{
		LastError: &errMsg,
		IsActive:  &isActive,
	}

	if _, updateErr := a.emailSourceRepo.Update(ctx, source.ID, updateInput); updateErr != nil {
		log.Error().Err(updateErr).Str("email_source_id", source.ID.String()).Msg("Failed to update email source error status")
	}
}
