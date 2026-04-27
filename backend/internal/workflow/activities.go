package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/domain/article"
	"github.com/bbu/rss-summarizer/backend/internal/domain/feed"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/bbu/rss-summarizer/backend/internal/service/llm"
	"github.com/bbu/rss-summarizer/backend/internal/service/rss"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type Activities struct {
	feedRepo        repository.FeedRepository
	articleRepo     repository.ArticleRepository
	prefsRepo       repository.PreferencesRepository
	topicRepo       repository.TopicRepository
	subscriptionRepo repository.SubscriptionRepository
	userArticleRepo  repository.UserArticleRepository
	rssService      rss.Service
	llmService      llm.Service
	provider        string
}

func NewActivities(
	feedRepo repository.FeedRepository,
	articleRepo repository.ArticleRepository,
	prefsRepo repository.PreferencesRepository,
	topicRepo repository.TopicRepository,
	subscriptionRepo repository.SubscriptionRepository,
	userArticleRepo repository.UserArticleRepository,
	rssService rss.Service,
	llmService llm.Service,
	provider string,
) *Activities {
	return &Activities{
		feedRepo:        feedRepo,
		articleRepo:     articleRepo,
		prefsRepo:       prefsRepo,
		topicRepo:       topicRepo,
		subscriptionRepo: subscriptionRepo,
		userArticleRepo:  userArticleRepo,
		rssService:      rssService,
		llmService:      llmService,
		provider:        provider,
	}
}

type FetchFeedInput struct {
	FeedID uuid.UUID
}

type FetchFeedOutput struct {
	NewArticleIDs []uuid.UUID
}

// FetchFeedActivity fetches RSS feed and creates new global articles for all subscribers
func (a *Activities) FetchFeedActivity(ctx context.Context, input FetchFeedInput) (*FetchFeedOutput, error) {
	// Get feed (now global, no user_id)
	f, err := a.feedRepo.FindByID(ctx, input.FeedID)
	if err != nil {
		return nil, fmt.Errorf("failed to find feed: %w", err)
	}

	// Use system default for max articles (no per-user preferences)
	maxArticles := 20

	// Fetch RSS feed
	metadata, err := a.rssService.FetchFeed(ctx, f.URL)
	if err != nil {
		// Update feed status on error
		a.updateFeedError(ctx, f, err)
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}

	// Update feed metadata if changed
	if metadata.Title != "" && f.Title != metadata.Title {
		f.Title = metadata.Title
	}
	if metadata.Description != "" && f.Description != metadata.Description {
		f.Description = metadata.Description
	}

	var newArticleIDs []uuid.UUID
	articlesProcessed := 0
	var itemErrors int

	// Create articles for new items (up to max limit).
	// Per-user state rows (user_articles) are intentionally not pre-seeded here:
	// FindByUserIDWithState LEFT-JOINs user_articles and supplies defaults for
	// missing rows, so pre-seeding would (a) incur N writes per new article,
	// and (b) risk racing with user interactions on activity retry — a rerun
	// calling Upsert(is_read=false) would overwrite is_read=true.
	for _, item := range metadata.Items {
		if articlesProcessed >= maxArticles {
			break
		}

		exists, err := a.articleRepo.ExistsByFeedAndURL(ctx, f.ID, item.URL)
		if err != nil {
			log.Error().Err(err).Str("feed_id", f.ID.String()).Str("url", item.URL).Msg("Failed to check article existence")
			itemErrors++
			continue
		}
		if exists {
			continue
		}

		feedID := f.ID
		art := &article.Article{
			ID:               uuid.New(),
			FeedID:           &feedID,
			Title:            item.Title,
			URL:              item.URL,
			PublishedAt:      item.PublishedAt,
			OriginalContent:  item.Content,
			SourceType:       "rss",
			ProcessingStatus: article.ProcessingPending,
		}

		if err := a.articleRepo.Create(ctx, art); err != nil {
			log.Error().Err(err).Str("feed_id", f.ID.String()).Str("url", item.URL).Msg("Failed to create article")
			itemErrors++
			continue
		}

		newArticleIDs = append(newArticleIDs, art.ID)
		articlesProcessed++
	}

	a.updateFeedSuccess(ctx, f)

	// If every item we tried to process failed, fail the activity so Temporal
	// retries — a partial success (some articles created) is still considered
	// progress and returns nil.
	if itemErrors > 0 && len(newArticleIDs) == 0 {
		return nil, fmt.Errorf("fetch feed %s: %d items failed and none succeeded", f.ID, itemErrors)
	}

	return &FetchFeedOutput{NewArticleIDs: newArticleIDs}, nil
}

// updateFeedError increments error count and updates feed status
func (a *Activities) updateFeedError(ctx context.Context, f *feed.Feed, err error) {
	f.ErrorCount++
	errMsg := err.Error()
	f.LastError = &errMsg

	// Set status based on error count
	if f.ErrorCount >= 3 {
		f.Status = feed.StatusError
	} else if f.ErrorCount >= 1 {
		f.Status = feed.StatusWarning
	}

	now := time.Now()
	f.LastPolledAt = &now

	if updateErr := a.feedRepo.Update(ctx, f); updateErr != nil {
		log.Error().Err(updateErr).Str("feed_id", f.ID.String()).Msg("Failed to update feed error status")
	}
}

// updateFeedSuccess resets error count and sets status to healthy
func (a *Activities) updateFeedSuccess(ctx context.Context, f *feed.Feed) {
	f.ErrorCount = 0
	f.LastError = nil
	f.Status = feed.StatusHealthy

	now := time.Now()
	f.LastPolledAt = &now

	if err := a.feedRepo.Update(ctx, f); err != nil {
		log.Error().Err(err).Str("feed_id", f.ID.String()).Msg("Failed to update feed success status")
	}
}

type SummarizeArticleInput struct {
	ArticleID uuid.UUID
}

// SummarizeArticleActivity summarizes a global article using system LLM config
func (a *Activities) SummarizeArticleActivity(ctx context.Context, input SummarizeArticleInput) error {
	// Get article (now global, no user_id)
	art, err := a.articleRepo.FindByID(ctx, input.ArticleID)
	if err != nil {
		return fmt.Errorf("failed to find article: %w", err)
	}

	// Skip if already summarized
	if art.Summary != "" && art.ProcessingStatus == article.ProcessingCompleted {
		return nil
	}

	// Set status to processing
	if err := a.articleRepo.UpdateProcessingStatus(ctx, art.ID, article.ProcessingProcessing, nil); err != nil {
		log.Error().Err(err).Str("article_id", art.ID.String()).Msg("Failed to update article processing status")
	}

	// Use original content or full text
	content := art.OriginalContent
	if content == "" {
		content = art.FullText
	}

	if content == "" {
		errMsg := "no content to summarize"
		a.articleRepo.UpdateProcessingStatus(ctx, art.ID, article.ProcessingFailed, &errMsg)
		return fmt.Errorf("no content to summarize")
	}

	// CHANGE: Always use system LLM config (no per-user preferences)
	// Use SummarizeArticleWithConfig to support different provider formats (Anthropic vs OpenAI)
	summary, err := a.llmService.SummarizeArticleWithConfig(ctx, art.Title, content, a.provider, "", "", "")
	if err != nil {
		errMsg := fmt.Sprintf("failed to summarize article: %v", err)
		a.articleRepo.UpdateProcessingStatus(ctx, art.ID, article.ProcessingFailed, &errMsg)
		return fmt.Errorf("failed to summarize article: %w", err)
	}

	// Update article with summary
	art.Summary = summary.Summary
	art.KeyPoints = pq.StringArray(summary.KeyPoints)
	art.ImportanceScore = &summary.ImportanceScore
	art.Topics = pq.StringArray(summary.Topics)

	if err := a.articleRepo.Update(ctx, art); err != nil {
		errMsg := fmt.Sprintf("failed to update article: %v", err)
		a.articleRepo.UpdateProcessingStatus(ctx, art.ID, article.ProcessingFailed, &errMsg)
		return fmt.Errorf("failed to update article: %w", err)
	}

	// Ensure all detected topics exist in the global topics table
	if len(summary.Topics) > 0 {
		if err := a.topicRepo.EnsureTopicsExist(ctx, summary.Topics); err != nil {
			// Log but don't fail - topics not existing won't prevent article from being displayed
			log.Error().Err(err).Str("article_id", art.ID.String()).Msg("Failed to ensure topics exist")
		}
	}

	// Set status to completed
	if err := a.articleRepo.UpdateProcessingStatus(ctx, art.ID, article.ProcessingCompleted, nil); err != nil {
		log.Error().Err(err).Str("article_id", art.ID.String()).Msg("Failed to update article to completed status")
	}

	return nil
}

// GetFeedsDueForPollActivity returns feeds that need to be polled
func (a *Activities) GetFeedsDueForPollActivity(ctx context.Context) ([]uuid.UUID, error) {
	feeds, err := a.feedRepo.FindActiveFeedsDueForPoll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find feeds due for poll: %w", err)
	}

	feedIDs := make([]uuid.UUID, len(feeds))
	for i, f := range feeds {
		feedIDs[i] = f.ID
	}

	return feedIDs, nil
}
