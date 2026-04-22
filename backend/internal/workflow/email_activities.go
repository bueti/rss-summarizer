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
			fmt.Printf("Failed to update email source tokens: %v\n", err)
		}
	}

	var newArticleIDs []uuid.UUID

	// Process each email
	for _, msg := range emails {
		// Check if email matches any filter
		if !a.matchesFilters(msg, filters) {
			continue
		}

		// Check if article already exists (check by email_message_id)
		exists, err := a.articleRepo.ExistsByEmailMessageID(ctx, msg.ID)
		if err != nil {
			fmt.Printf("Failed to check article existence for message %s: %v\n", msg.ID, err)
			continue
		}

		// If article already exists, just mark email as read and skip
		if exists {
			newToken, err = a.gmailService.MarkAsReadWithToken(ctx, newToken, msg.ID)
			if err != nil {
				fmt.Printf("Failed to mark existing email as read: %v\n", err)
			}
			continue
		}

		// Parse email content
		content, err := email.ParseEmailContent(msg.BodyHTML, msg.BodyPlain)
		if err != nil {
			fmt.Printf("Failed to parse email content for message %s: %v\n", msg.ID, err)
			continue
		}

		if content == "" {
			fmt.Printf("No content extracted from message %s\n", msg.ID)
			continue
		}

		// Create article from email
		art := &article.Article{
			ID:               uuid.New(),
			FeedID:           nil,        // No feed for email-sourced articles
			EmailSourceID:    &source.ID, // Link to email source for filtering
			Title:            msg.Subject,
			URL:              "", // Emails don't have URLs
			PublishedAt:      &msg.Date,
			OriginalContent:  content,
			SourceType:       "email",
			EmailMessageID:   &msg.ID,
			ProcessingStatus: article.ProcessingPending,
		}

		if err := a.articleRepo.Create(ctx, art); err != nil {
			fmt.Printf("Failed to create article from email %s: %v\n", msg.ID, err)
			continue
		}

		// Create user_article entry for the email source owner
		if err := a.userArticleRepo.Upsert(ctx, source.UserID, art.ID, false); err != nil {
			fmt.Printf("Failed to create user_article for email %s: %v\n", msg.ID, err)
		}

		newArticleIDs = append(newArticleIDs, art.ID)

		// Mark email as read after successful processing
		newToken, err = a.gmailService.MarkAsReadWithToken(ctx, newToken, msg.ID)
		if err != nil {
			fmt.Printf("Failed to mark email as read: %v\n", err)
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
		fmt.Printf("Failed to update email source: %v\n", err)
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
		fmt.Printf("Failed to update email source error status: %v\n", updateErr)
	}
}
