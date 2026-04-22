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

	// Create articles for new items (up to max limit)
	for _, item := range metadata.Items {
		// Stop if we've reached the max articles per poll
		if articlesProcessed >= maxArticles {
			break
		}

		// Skip if article already exists (global check)
		exists, err := a.articleRepo.ExistsByFeedAndURL(ctx, f.ID, item.URL)
		if err != nil {
			// Log error but continue processing
			fmt.Printf("Failed to check article existence for %s: %v\n", item.URL, err)
			continue
		}
		if exists {
			continue
		}

		// Create new global article (no user_id)
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
			// Log error but continue processing other articles
			fmt.Printf("Failed to create article %s: %v\n", item.URL, err)
			continue
		}

		// CRITICAL: Create user_articles for ALL subscribers to this feed
		subscribers, err := a.subscriptionRepo.FindByFeedID(ctx, f.ID)
		if err != nil {
			fmt.Printf("Failed to get feed subscribers: %v\n", err)
		} else {
			for _, sub := range subscribers {
				if err := a.userArticleRepo.Upsert(ctx, sub.UserID, art.ID, false); err != nil {
					fmt.Printf("Failed to create user_article for user %s: %v\n", sub.UserID, err)
				}
			}
		}

		newArticleIDs = append(newArticleIDs, art.ID)
		articlesProcessed++
	}

	// Update feed status on success
	a.updateFeedSuccess(ctx, f)

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
		fmt.Printf("Failed to update feed error status: %v\n", updateErr)
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
		fmt.Printf("Failed to update feed success status: %v\n", err)
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
		fmt.Printf("Failed to update article processing status: %v\n", err)
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
	summary, err := a.llmService.SummarizeArticle(ctx, art.Title, content)
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
			fmt.Printf("Failed to ensure topics exist: %v\n", err)
		}
	}

	// Set status to completed
	if err := a.articleRepo.UpdateProcessingStatus(ctx, art.ID, article.ProcessingCompleted, nil); err != nil {
		fmt.Printf("Failed to update article to completed status: %v\n", err)
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
