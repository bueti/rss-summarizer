package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	domerrors "github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/bbu/rss-summarizer/backend/internal/domain/feed"
	"github.com/bbu/rss-summarizer/backend/internal/domain/subscription"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/bbu/rss-summarizer/backend/internal/service/rss"
	"github.com/bbu/rss-summarizer/backend/internal/workflow"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

type FeedHandlers struct {
	feedRepo         repository.FeedRepository
	subscriptionRepo repository.SubscriptionRepository
	rssService       rss.Service
	temporalClient   client.Client
}

func NewFeedHandlers(
	feedRepo repository.FeedRepository,
	subscriptionRepo repository.SubscriptionRepository,
	rssService rss.Service,
	temporalClient client.Client,
) *FeedHandlers {
	return &FeedHandlers{
		feedRepo:         feedRepo,
		subscriptionRepo: subscriptionRepo,
		rssService:       rssService,
		temporalClient:   temporalClient,
	}
}

// Helper to convert feed domain model to response
func feedToResponse(f *feed.Feed) FeedResponse {
	return FeedResponse{
		Body: struct {
			ID                   uuid.UUID  `json:"id"`
			URL                  string     `json:"url"`
			Title                string     `json:"title"`
			Description          string     `json:"description"`
			PollFrequencyMinutes int        `json:"poll_frequency_minutes"`
			LastPolledAt         *time.Time `json:"last_polled_at,omitempty"`
			IsActive             bool       `json:"is_active"`
			Status               string     `json:"status"`
			LastError            *string    `json:"last_error,omitempty"`
			ErrorCount           int        `json:"error_count"`
			CreatedAt            time.Time  `json:"created_at"`
			UpdatedAt            time.Time  `json:"updated_at"`
		}{
			ID:                   f.ID,
			URL:                  f.URL,
			Title:                f.Title,
			Description:          f.Description,
			PollFrequencyMinutes: f.PollFrequencyMinutes,
			LastPolledAt:         f.LastPolledAt,
			IsActive:             f.IsActive,
			Status:               f.Status,
			LastError:            f.LastError,
			ErrorCount:           f.ErrorCount,
			CreatedAt:            f.CreatedAt,
			UpdatedAt:            f.UpdatedAt,
		},
	}
}

// Request/Response types
type CreateFeedRequest struct {
	Body struct {
		URL                  string `json:"url" pattern:"^https?://.+" doc:"RSS feed URL"`
		PollFrequencyMinutes int    `json:"poll_frequency_minutes" minimum:"15" maximum:"1440" default:"60" doc:"Polling frequency in minutes"`
	}
}

type FeedResponse struct {
	Body struct {
		ID                   uuid.UUID  `json:"id"`
		URL                  string     `json:"url"`
		Title                string     `json:"title"`
		Description          string     `json:"description"`
		PollFrequencyMinutes int        `json:"poll_frequency_minutes"`
		LastPolledAt         *time.Time `json:"last_polled_at,omitempty"`
		IsActive             bool       `json:"is_active"`
		Status               string     `json:"status"`
		LastError            *string    `json:"last_error,omitempty"`
		ErrorCount           int        `json:"error_count"`
		CreatedAt            time.Time  `json:"created_at"`
		UpdatedAt            time.Time  `json:"updated_at"`
	}
}

type ListFeedsRequest struct {
	Limit  int `query:"limit" default:"50" minimum:"1" maximum:"100" doc:"Maximum number of feeds to return"`
	Offset int `query:"offset" default:"0" minimum:"0" doc:"Number of feeds to skip"`
}

type ListFeedsResponse struct {
	Body struct {
		Feeds []struct {
			ID                   uuid.UUID  `json:"id"`
			URL                  string     `json:"url"`
			Title                string     `json:"title"`
			Description          string     `json:"description"`
			PollFrequencyMinutes int        `json:"poll_frequency_minutes"`
			LastPolledAt         *time.Time `json:"last_polled_at,omitempty"`
			IsActive             bool       `json:"is_active"`
			Status               string     `json:"status"`
			LastError            *string    `json:"last_error,omitempty"`
			ErrorCount           int        `json:"error_count"`
			CreatedAt            time.Time  `json:"created_at"`
			UpdatedAt            time.Time  `json:"updated_at"`
		} `json:"feeds"`
		TotalCount int `json:"total_count"`
		Limit      int `json:"limit"`
		Offset     int `json:"offset"`
	}
}

type UpdateFeedRequest struct {
	ID   string `path:"id" format:"uuid" doc:"Feed ID"`
	Body struct {
		Title                *string `json:"title,omitempty" doc:"Feed title"`
		PollFrequencyMinutes *int    `json:"poll_frequency_minutes,omitempty" minimum:"15" maximum:"1440" doc:"Poll frequency in minutes"`
		IsActive             *bool   `json:"is_active,omitempty" doc:"Whether the feed is active"`
	}
}

type RefreshFeedRequest struct {
	ID string `path:"id" format:"uuid" doc:"Feed ID"`
}

type FeedHealthResponse struct {
	Body struct {
		Status       string    `json:"status"`
		LastError    *string   `json:"last_error,omitempty"`
		ErrorCount   int       `json:"error_count"`
		LastPolledAt *time.Time `json:"last_polled_at,omitempty"`
		IsActive     bool       `json:"is_active"`
	}
}

func (h *FeedHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-feed",
		Method:      http.MethodPost,
		Path:        "/v1/feeds",
		Summary:     "Create a new RSS feed",
		Description: "Add a new RSS feed for the authenticated user",
		Tags:        []string{"Feeds"},
	}, h.CreateFeed)

	huma.Register(api, huma.Operation{
		OperationID: "list-feeds",
		Method:      http.MethodGet,
		Path:        "/v1/feeds",
		Summary:     "List all feeds",
		Description: "Get all RSS feeds for the authenticated user with pagination",
		Tags:        []string{"Feeds"},
	}, h.ListFeeds)

	huma.Register(api, huma.Operation{
		OperationID: "get-feed",
		Method:      http.MethodGet,
		Path:        "/v1/feeds/{id}",
		Summary:     "Get feed by ID",
		Description: "Get a specific RSS feed by ID",
		Tags:        []string{"Feeds"},
	}, h.GetFeed)

	huma.Register(api, huma.Operation{
		OperationID: "update-feed",
		Method:      http.MethodPut,
		Path:        "/v1/feeds/{id}",
		Summary:     "Update a feed",
		Description: "Update feed settings",
		Tags:        []string{"Feeds"},
	}, h.UpdateFeed)

	huma.Register(api, huma.Operation{
		OperationID: "delete-feed",
		Method:      http.MethodDelete,
		Path:        "/v1/feeds/{id}",
		Summary:     "Delete a feed",
		Description: "Delete an RSS feed and all its articles",
		Tags:        []string{"Feeds"},
	}, h.DeleteFeed)

	huma.Register(api, huma.Operation{
		OperationID: "refresh-feed",
		Method:      http.MethodPost,
		Path:        "/v1/feeds/{id}/refresh",
		Summary:     "Manually refresh a feed",
		Description: "Trigger an immediate poll of the feed",
		Tags:        []string{"Feeds"},
	}, h.RefreshFeed)

	huma.Register(api, huma.Operation{
		OperationID: "get-feed-health",
		Method:      http.MethodGet,
		Path:        "/v1/feeds/{id}/health",
		Summary:     "Get feed health",
		Description: "Get the health status and error information for a feed",
		Tags:        []string{"Feeds"},
	}, h.GetFeedHealth)
}

func (h *FeedHandlers) CreateFeed(ctx context.Context, input *CreateFeedRequest) (*FeedResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	// Validate feed URL by fetching it
	metadata, err := h.rssService.FetchFeed(ctx, input.Body.URL)
	if err != nil {
		return nil, huma.Error400BadRequest(fmt.Sprintf("Invalid feed URL: %v", err))
	}

	pollFreq := input.Body.PollFrequencyMinutes
	if pollFreq == 0 {
		pollFreq = 60 // default 1 hour
	}

	// Find or create global feed
	f, err := h.feedRepo.FindByURL(ctx, input.Body.URL)
	if err != nil {
		// If feed doesn't exist, create it
		var notFoundErr *domerrors.NotFoundError
		if errors.As(err, &notFoundErr) {
			f, err = h.feedRepo.Create(ctx, &feed.CreateFeedInput{
				URL:                  input.Body.URL,
				PollFrequencyMinutes: pollFreq,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create feed: %w", err)
			}

			// Update with metadata
			f.Title = metadata.Title
			f.Description = metadata.Description
			f.Status = feed.StatusHealthy
			if err := h.feedRepo.Update(ctx, f); err != nil {
				return nil, fmt.Errorf("failed to update feed metadata: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to find feed: %w", err)
		}
	}

	// Create subscription for this user
	_, err = h.subscriptionRepo.Create(ctx, &subscription.CreateSubscriptionInput{
		UserID: userID,
		FeedID: f.ID,
	})
	if err != nil {
		// If already subscribed, return duplicate error
		var dupErr *domerrors.DuplicateError
		if errors.As(err, &dupErr) {
			return nil, huma.Error400BadRequest("Already subscribed to this feed")
		}
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	resp := feedToResponse(f)
	return &resp, nil
}

func (h *FeedHandlers) ListFeeds(ctx context.Context, input *ListFeedsRequest) (*ListFeedsResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	rows, total, err := h.subscriptionRepo.ListSubscribedFeeds(ctx, userID, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscribed feeds: %w", err)
	}

	response := &ListFeedsResponse{}
	response.Body.TotalCount = total
	response.Body.Limit = input.Limit
	response.Body.Offset = input.Offset
	response.Body.Feeds = make([]struct {
		ID                   uuid.UUID  `json:"id"`
		URL                  string     `json:"url"`
		Title                string     `json:"title"`
		Description          string     `json:"description"`
		PollFrequencyMinutes int        `json:"poll_frequency_minutes"`
		LastPolledAt         *time.Time `json:"last_polled_at,omitempty"`
		IsActive             bool       `json:"is_active"`
		Status               string     `json:"status"`
		LastError            *string    `json:"last_error,omitempty"`
		ErrorCount           int        `json:"error_count"`
		CreatedAt            time.Time  `json:"created_at"`
		UpdatedAt            time.Time  `json:"updated_at"`
	}, 0, len(rows))

	for _, f := range rows {
		response.Body.Feeds = append(response.Body.Feeds, struct {
			ID                   uuid.UUID  `json:"id"`
			URL                  string     `json:"url"`
			Title                string     `json:"title"`
			Description          string     `json:"description"`
			PollFrequencyMinutes int        `json:"poll_frequency_minutes"`
			LastPolledAt         *time.Time `json:"last_polled_at,omitempty"`
			IsActive             bool       `json:"is_active"`
			Status               string     `json:"status"`
			LastError            *string    `json:"last_error,omitempty"`
			ErrorCount           int        `json:"error_count"`
			CreatedAt            time.Time  `json:"created_at"`
			UpdatedAt            time.Time  `json:"updated_at"`
		}{
			ID:                   f.ID,
			URL:                  f.URL,
			Title:                f.Title,
			Description:          f.Description,
			PollFrequencyMinutes: f.EffectivePollFrequencyMinutes,
			LastPolledAt:         f.LastPolledAt,
			IsActive:             f.IsActive,
			Status:               f.Status,
			LastError:            f.LastError,
			ErrorCount:           f.ErrorCount,
			CreatedAt:            f.CreatedAt,
			UpdatedAt:            f.UpdatedAt,
		})
	}

	return response, nil
}

type GetFeedRequest struct {
	ID string `path:"id" format:"uuid" doc:"Feed ID"`
}

func (h *FeedHandlers) GetFeed(ctx context.Context, input *GetFeedRequest) (*FeedResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	feedID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid feed ID")
	}

	// Get user's subscription to this feed
	sub, err := h.subscriptionRepo.FindByUserAndFeed(ctx, userID, feedID)
	if err != nil {
		var notFoundErr *domerrors.NotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, huma.Error404NotFound("Feed not found or not subscribed")
		}
		return nil, fmt.Errorf("failed to check subscription: %w", err)
	}

	f, err := h.feedRepo.FindByID(ctx, feedID)
	if err != nil {
		var notFoundErr *domerrors.NotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, huma.Error404NotFound("Feed not found")
		}
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	// Use subscription override if set, otherwise use feed default
	if sub.PollFrequencyOverride != nil {
		f.PollFrequencyMinutes = *sub.PollFrequencyOverride
	}

	resp := feedToResponse(f)
	return &resp, nil
}

func (h *FeedHandlers) UpdateFeed(ctx context.Context, input *UpdateFeedRequest) (*FeedResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	feedID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid feed ID")
	}

	// Get user's subscription
	sub, err := h.subscriptionRepo.FindByUserAndFeed(ctx, userID, feedID)
	if err != nil {
		var notFoundErr *domerrors.NotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, huma.Error404NotFound("Feed not found or not subscribed")
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Update subscription settings (not global feed)
	// Note: Title is not user-editable (comes from RSS metadata)
	if input.Body.PollFrequencyMinutes != nil {
		sub.PollFrequencyOverride = input.Body.PollFrequencyMinutes
	}
	if input.Body.IsActive != nil {
		sub.IsActive = *input.Body.IsActive
	}

	if err := h.subscriptionRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Return the global feed with effective poll frequency for this user
	f, err := h.feedRepo.FindByID(ctx, feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	// Use subscription override if set, otherwise use feed default
	if sub.PollFrequencyOverride != nil {
		f.PollFrequencyMinutes = *sub.PollFrequencyOverride
	}

	resp := feedToResponse(f)
	return &resp, nil
}

func (h *FeedHandlers) RefreshFeed(ctx context.Context, input *RefreshFeedRequest) (*struct{}, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	feedID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid feed ID")
	}

	// Verify user has subscription to this feed
	if _, err := h.subscriptionRepo.FindByUserAndFeed(ctx, userID, feedID); err != nil {
		var notFoundErr *domerrors.NotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, huma.Error404NotFound("Feed not found or not subscribed")
		}
		return nil, fmt.Errorf("failed to check subscription: %w", err)
	}

	// Start the ProcessFeedWorkflow directly instead of rewinding last_polled_at
	// and relying on the 5-minute poller. Matches the retry-article flow.
	workflowOptions := client.StartWorkflowOptions{
		ID:                    "process-feed-" + feedID.String(),
		TaskQueue:             workflow.FeedPollingTaskQueue,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	}
	if _, err := h.temporalClient.ExecuteWorkflow(ctx, workflowOptions, workflow.ProcessFeedWorkflow, feedID); err != nil {
		return nil, fmt.Errorf("failed to start feed refresh workflow: %w", err)
	}

	return &struct{}{}, nil
}

func (h *FeedHandlers) GetFeedHealth(ctx context.Context, input *GetFeedRequest) (*FeedHealthResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	feedID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid feed ID")
	}

	// Verify user has subscription to this feed
	_, err = h.subscriptionRepo.FindByUserAndFeed(ctx, userID, feedID)
	if err != nil {
		var notFoundErr *domerrors.NotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, huma.Error404NotFound("Feed not found or not subscribed")
		}
		return nil, fmt.Errorf("failed to check subscription: %w", err)
	}

	f, err := h.feedRepo.FindByID(ctx, feedID)
	if err != nil {
		var notFoundErr *domerrors.NotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, huma.Error404NotFound("Feed not found")
		}
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	return &FeedHealthResponse{
		Body: struct {
			Status       string    `json:"status"`
			LastError    *string   `json:"last_error,omitempty"`
			ErrorCount   int       `json:"error_count"`
			LastPolledAt *time.Time `json:"last_polled_at,omitempty"`
			IsActive     bool       `json:"is_active"`
		}{
			Status:       f.Status,
			LastError:    f.LastError,
			ErrorCount:   f.ErrorCount,
			LastPolledAt: f.LastPolledAt,
			IsActive:     f.IsActive,
		},
	}, nil
}

type DeleteFeedRequest struct {
	ID string `path:"id" format:"uuid" doc:"Feed ID"`
}

func (h *FeedHandlers) DeleteFeed(ctx context.Context, input *DeleteFeedRequest) (*struct{}, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	feedID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid feed ID")
	}

	// Delete user's subscription (not the global feed)
	if err := h.subscriptionRepo.DeleteByUserAndFeed(ctx, userID, feedID); err != nil {
		var notFoundErr *domerrors.NotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, huma.Error404NotFound("Feed not found or not subscribed")
		}
		return nil, fmt.Errorf("failed to delete subscription: %w", err)
	}

	// Optionally: Delete the global feed if no subscribers remain
	// For now, we keep feeds around even with 0 subscribers
	// This allows for faster re-subscription and preserves article history

	return &struct{}{}, nil
}
