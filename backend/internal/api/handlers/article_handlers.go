package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	"github.com/bbu/rss-summarizer/backend/internal/domain/article"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/bbu/rss-summarizer/backend/internal/workflow"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

type ArticleHandlers struct {
	articleRepo     repository.ArticleRepository
	userArticleRepo repository.UserArticleRepository
	temporalClient  client.Client
}

func NewArticleHandlers(
	articleRepo repository.ArticleRepository,
	userArticleRepo repository.UserArticleRepository,
	temporalClient client.Client,
) *ArticleHandlers {
	return &ArticleHandlers{
		articleRepo:     articleRepo,
		userArticleRepo: userArticleRepo,
		temporalClient:  temporalClient,
	}
}

type ArticleResponse struct {
	ID               uuid.UUID  `json:"id"`
	FeedID           *uuid.UUID `json:"feed_id,omitempty" doc:"RSS feed ID (null for email-sourced articles)"`
	EmailSourceID    *uuid.UUID `json:"email_source_id,omitempty" doc:"Email source ID (null for RSS-sourced articles)"`
	Title            string     `json:"title"`
	URL              string     `json:"url"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	Summary          string     `json:"summary,omitempty"`
	KeyPoints        []string   `json:"key_points,omitempty"`
	ImportanceScore  *int       `json:"importance_score,omitempty"`
	Topics           []string   `json:"topics,omitempty"`
	IsRead           bool       `json:"is_read"`
	IsSaved          bool       `json:"is_saved"`
	IsArchived       bool       `json:"is_archived"`
	ProcessingStatus string     `json:"processing_status"`
	ProcessingError  *string    `json:"processing_error,omitempty"`
	SourceType       string     `json:"source_type" doc:"Source type: 'rss' or 'email'"`
	EmailMessageID   *string    `json:"email_message_id,omitempty" doc:"Gmail message ID for email-sourced articles"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ListArticlesRequest struct {
	FeedID           string `query:"feed_id" format:"uuid" doc:"Filter by feed ID"`
	EmailSourceID    string `query:"email_source_id" format:"uuid" doc:"Filter by email source ID"`
	MinImportance    int    `query:"min_importance" minimum:"1" maximum:"5" doc:"Minimum importance score"`
	Topic            string `query:"topic" doc:"Filter by topic"`
	IsRead           string `query:"is_read" doc:"Filter by read status (true/false)"`
	IsSaved          string `query:"is_saved" doc:"Filter by saved status (true/false)"`
	IsArchived       string `query:"is_archived" doc:"Filter by archived status (true/false)"`
	ProcessingStatus string `query:"processing_status" doc:"Filter by processing status (pending/processing/completed/failed)"`
	SortBy           string `query:"sort_by" enum:"date,importance" doc:"Sort order: date (newest first) or importance (highest first)"`
	Limit            int    `query:"limit" default:"50" maximum:"100" doc:"Results per page"`
	Offset           int    `query:"offset" default:"0" doc:"Pagination offset"`
}

type ListArticlesResponse struct {
	Body struct {
		Articles   []ArticleResponse `json:"articles"`
		TotalCount int               `json:"total_count"`
		Limit      int               `json:"limit"`
		Offset     int               `json:"offset"`
	}
}

type BulkMarkReadResponse struct {
	Body struct {
		Updated int `json:"updated"`
	}
}

type BulkMarkReadRequest struct {
	Body struct {
		ArticleIDs []string `json:"article_ids" minItems:"1" maxItems:"100" doc:"Array of article IDs"`
		IsRead     bool     `json:"is_read" doc:"Read status to set"`
	}
}

type RetryArticleRequest struct {
	ID string `path:"id" format:"uuid" doc:"Article ID"`
}

type GetArticleRequest struct {
	ID string `path:"id" format:"uuid" doc:"Article ID"`
}

type GetArticleResponse struct {
	Body ArticleResponse
}

type MarkAsReadRequest struct {
	ID   string `path:"id" format:"uuid" doc:"Article ID"`
	Body struct {
		IsRead bool `json:"is_read" doc:"Read status"`
	}
}

type SetSavedRequest struct {
	ID   string `path:"id" format:"uuid" doc:"Article ID"`
	Body struct {
		IsSaved bool `json:"is_saved" doc:"Saved status"`
	}
}

type SetArchivedRequest struct {
	ID   string `path:"id" format:"uuid" doc:"Article ID"`
	Body struct {
		IsArchived bool `json:"is_archived" doc:"Archived status"`
	}
}

type BulkSetSavedRequest struct {
	Body struct {
		ArticleIDs []string `json:"article_ids" minItems:"1" maxItems:"100" doc:"Array of article IDs"`
		IsSaved    bool     `json:"is_saved" doc:"Saved status to set"`
	}
}

type BulkSetArchivedRequest struct {
	Body struct {
		ArticleIDs []string `json:"article_ids" minItems:"1" maxItems:"100" doc:"Array of article IDs"`
		IsArchived bool     `json:"is_archived" doc:"Archived status to set"`
	}
}

func (h *ArticleHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-articles",
		Method:      http.MethodGet,
		Path:        "/v1/articles",
		Summary:     "List articles",
		Description: "Get articles with optional filters and pagination",
		Tags:        []string{"Articles"},
	}, h.ListArticles)

	huma.Register(api, huma.Operation{
		OperationID: "get-article",
		Method:      http.MethodGet,
		Path:        "/v1/articles/{id}",
		Summary:     "Get article by ID",
		Description: "Get a specific article by ID",
		Tags:        []string{"Articles"},
	}, h.GetArticle)

	huma.Register(api, huma.Operation{
		OperationID: "mark-article-read",
		Method:      http.MethodPatch,
		Path:        "/v1/articles/{id}/read",
		Summary:     "Mark article as read/unread",
		Description: "Update the read status of an article",
		Tags:        []string{"Articles"},
	}, h.MarkAsRead)

	huma.Register(api, huma.Operation{
		OperationID: "bulk-mark-read",
		Method:      http.MethodPatch,
		Path:        "/v1/articles/mark-read",
		Summary:     "Bulk mark articles as read",
		Description: "Mark multiple articles as read or unread",
		Tags:        []string{"Articles"},
	}, h.BulkMarkRead)

	huma.Register(api, huma.Operation{
		OperationID: "process-article",
		Method:      http.MethodPost,
		Path:        "/v1/articles/{id}/retry",
		Summary:     "Process or reprocess article",
		Description: "Trigger processing for pending articles, retry failed articles, or reprocess completed articles (e.g., to regenerate summaries with updated prompts)",
		Tags:        []string{"Articles"},
	}, h.RetryArticle)

	huma.Register(api, huma.Operation{
		OperationID: "set-article-saved",
		Method:      http.MethodPatch,
		Path:        "/v1/articles/{id}/save",
		Summary:     "Save/unsave article",
		Description: "Update the saved status of an article",
		Tags:        []string{"Articles"},
	}, h.SetSaved)

	huma.Register(api, huma.Operation{
		OperationID: "set-article-archived",
		Method:      http.MethodPatch,
		Path:        "/v1/articles/{id}/archive",
		Summary:     "Archive/unarchive article",
		Description: "Update the archived status of an article",
		Tags:        []string{"Articles"},
	}, h.SetArchived)

	huma.Register(api, huma.Operation{
		OperationID: "bulk-set-saved",
		Method:      http.MethodPatch,
		Path:        "/v1/articles/bulk-save",
		Summary:     "Bulk save/unsave articles",
		Description: "Save or unsave multiple articles",
		Tags:        []string{"Articles"},
	}, h.BulkSetSaved)

	huma.Register(api, huma.Operation{
		OperationID: "bulk-set-archived",
		Method:      http.MethodPatch,
		Path:        "/v1/articles/bulk-archive",
		Summary:     "Bulk archive/unarchive articles",
		Description: "Archive or unarchive multiple articles",
		Tags:        []string{"Articles"},
	}, h.BulkSetArchived)
}

func (h *ArticleHandlers) ListArticles(ctx context.Context, input *ListArticlesRequest) (*ListArticlesResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	filters := article.ArticleFilters{
		Limit:  50,
		Offset: 0,
	}

	// Handle optional filters
	if input.MinImportance > 0 {
		filters.MinImportance = &input.MinImportance
	}

	if input.Topic != "" {
		filters.Topic = &input.Topic
	}

	if input.IsRead != "" {
		isRead := input.IsRead == "true"
		filters.IsRead = &isRead
	}

	if input.IsSaved != "" {
		isSaved := input.IsSaved == "true"
		filters.IsSaved = &isSaved
	}

	if input.IsArchived != "" {
		isArchived := input.IsArchived == "true"
		filters.IsArchived = &isArchived
	}

	if input.ProcessingStatus != "" {
		filters.ProcessingStatus = &input.ProcessingStatus
	}

	if input.Limit > 0 {
		filters.Limit = input.Limit
	}

	if input.Offset > 0 {
		filters.Offset = input.Offset
	}

	if input.FeedID != "" {
		feedID, err := uuid.Parse(input.FeedID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid feed ID")
		}
		filters.FeedID = &feedID
	}

	if input.EmailSourceID != "" {
		emailSourceID, err := uuid.Parse(input.EmailSourceID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid email source ID")
		}
		filters.EmailSourceID = &emailSourceID
	}

	if input.SortBy != "" {
		filters.SortBy = &input.SortBy
	}

	// Get total count with user state
	totalCount, err := h.articleRepo.CountByUserIDWithState(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to count articles: %w", err)
	}

	// Get articles with user state (includes is_read from user_articles join)
	articles, err := h.articleRepo.FindByUserIDWithState(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list articles: %w", err)
	}

	response := &ListArticlesResponse{}
	response.Body.TotalCount = totalCount
	response.Body.Limit = filters.Limit
	response.Body.Offset = filters.Offset
	response.Body.Articles = make([]ArticleResponse, 0, len(articles))

	for _, a := range articles {
		response.Body.Articles = append(response.Body.Articles, ArticleResponse{
			ID:               a.ID,
			FeedID:           a.FeedID,
			EmailSourceID:    a.EmailSourceID,
			Title:            a.Title,
			URL:              a.URL,
			PublishedAt:      a.PublishedAt,
			Summary:          a.Summary,
			KeyPoints:        a.KeyPoints,
			ImportanceScore:  a.ImportanceScore,
			Topics:           a.Topics,
			IsRead:           a.IsRead,
			IsSaved:          a.IsSaved,
			IsArchived:       a.IsArchived,
			ProcessingStatus: a.ProcessingStatus,
			ProcessingError:  a.ProcessingError,
			SourceType:       a.SourceType,
			EmailMessageID:   a.EmailMessageID,
			CreatedAt:        a.CreatedAt,
			UpdatedAt:        a.UpdatedAt,
		})
	}

	return response, nil
}

func (h *ArticleHandlers) GetArticle(ctx context.Context, input *GetArticleRequest) (*GetArticleResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	articleID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid article ID")
	}

	// Get global article
	a, err := h.articleRepo.FindByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	// Get user's state for this article (read, saved, archived)
	isRead := false
	isSaved := false
	isArchived := false
	userArticle, err := h.userArticleRepo.FindByUserAndArticle(ctx, userID, articleID)
	if err == nil {
		isRead = userArticle.IsRead
		isSaved = userArticle.IsSaved
		isArchived = userArticle.IsArchived
	}
	// If not found, it means user hasn't interacted with it yet (all defaults to false)

	return &GetArticleResponse{
		Body: ArticleResponse{
			ID:               a.ID,
			FeedID:           a.FeedID,
			EmailSourceID:    a.EmailSourceID,
			Title:            a.Title,
			URL:              a.URL,
			PublishedAt:      a.PublishedAt,
			Summary:          a.Summary,
			KeyPoints:        a.KeyPoints,
			ImportanceScore:  a.ImportanceScore,
			Topics:           a.Topics,
			IsRead:           isRead,
			IsSaved:          isSaved,
			IsArchived:       isArchived,
			ProcessingStatus: a.ProcessingStatus,
			ProcessingError:  a.ProcessingError,
			SourceType:       a.SourceType,
			EmailMessageID:   a.EmailMessageID,
			CreatedAt:        a.CreatedAt,
			UpdatedAt:        a.UpdatedAt,
		},
	}, nil
}

func (h *ArticleHandlers) MarkAsRead(ctx context.Context, input *MarkAsReadRequest) (*struct{}, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	articleID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid article ID")
	}

	// Update user's read state for this article
	if err := h.userArticleRepo.MarkAsRead(ctx, userID, articleID, input.Body.IsRead); err != nil {
		return nil, fmt.Errorf("failed to update read status: %w", err)
	}

	return &struct{}{}, nil
}

func (h *ArticleHandlers) BulkMarkRead(ctx context.Context, input *BulkMarkReadRequest) (*BulkMarkReadResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	if len(input.Body.ArticleIDs) == 0 {
		return nil, huma.Error400BadRequest("Article IDs array is empty")
	}

	// Parse all article IDs
	ids := make([]uuid.UUID, 0, len(input.Body.ArticleIDs))
	for _, idStr := range input.Body.ArticleIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, huma.Error400BadRequest(fmt.Sprintf("Invalid article ID: %s", idStr))
		}
		ids = append(ids, id)
	}

	// Bulk update user's read state for these articles
	if err := h.userArticleRepo.MarkReadBulk(ctx, userID, ids, input.Body.IsRead); err != nil {
		return nil, fmt.Errorf("failed to bulk mark articles: %w", err)
	}

	return &BulkMarkReadResponse{
		Body: struct {
			Updated int `json:"updated"`
		}{
			Updated: len(ids),
		},
	}, nil
}

func (h *ArticleHandlers) RetryArticle(ctx context.Context, input *RetryArticleRequest) (*struct{}, error) {
	articleID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid article ID")
	}

	// Get the article to validate it exists
	a, err := h.articleRepo.FindByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	// Allow reprocessing for any status (pending, processing, failed, or completed)
	// This enables regenerating summaries with updated prompts

	// Skip processing if currently being processed to avoid duplicate workflows
	if a.ProcessingStatus == article.ProcessingProcessing {
		return nil, huma.Error400BadRequest("Article is currently being processed. Please wait for it to complete.")
	}

	// Reset processing status to pending and clear any previous errors
	if err := h.articleRepo.UpdateProcessingStatus(ctx, articleID, article.ProcessingPending, nil); err != nil {
		return nil, fmt.Errorf("failed to reset processing status: %w", err)
	}

	// Trigger Temporal workflow to process this article
	workflowOptions := client.StartWorkflowOptions{
		ID:        "summarize-article-reprocess-" + articleID.String(),
		TaskQueue: workflow.FeedPollingTaskQueue,
	}

	_, err = h.temporalClient.ExecuteWorkflow(ctx, workflowOptions, workflow.SummarizeArticleWorkflow, articleID)
	if err != nil {
		// If workflow start fails, set status to failed
		errMsg := fmt.Sprintf("failed to start workflow: %v", err)
		h.articleRepo.UpdateProcessingStatus(ctx, articleID, article.ProcessingFailed, &errMsg)
		return nil, fmt.Errorf("failed to trigger article processing workflow: %w", err)
	}

	return &struct{}{}, nil
}

func (h *ArticleHandlers) SetSaved(ctx context.Context, input *SetSavedRequest) (*struct{}, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	articleID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid article ID")
	}

	// Update user's saved state for this article
	if err := h.userArticleRepo.SetSaved(ctx, userID, articleID, input.Body.IsSaved); err != nil {
		return nil, fmt.Errorf("failed to update saved status: %w", err)
	}

	return &struct{}{}, nil
}

func (h *ArticleHandlers) SetArchived(ctx context.Context, input *SetArchivedRequest) (*struct{}, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	articleID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid article ID")
	}

	// Update user's archived state for this article
	if err := h.userArticleRepo.SetArchived(ctx, userID, articleID, input.Body.IsArchived); err != nil {
		return nil, fmt.Errorf("failed to update archived status: %w", err)
	}

	return &struct{}{}, nil
}

func (h *ArticleHandlers) BulkSetSaved(ctx context.Context, input *BulkSetSavedRequest) (*BulkMarkReadResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	if len(input.Body.ArticleIDs) == 0 {
		return nil, huma.Error400BadRequest("Article IDs array is empty")
	}

	// Parse all article IDs
	ids := make([]uuid.UUID, 0, len(input.Body.ArticleIDs))
	for _, idStr := range input.Body.ArticleIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, huma.Error400BadRequest(fmt.Sprintf("Invalid article ID: %s", idStr))
		}
		ids = append(ids, id)
	}

	// Bulk update user's saved state for these articles
	if err := h.userArticleRepo.SetSavedBulk(ctx, userID, ids, input.Body.IsSaved); err != nil {
		return nil, fmt.Errorf("failed to bulk set saved status: %w", err)
	}

	return &BulkMarkReadResponse{
		Body: struct {
			Updated int `json:"updated"`
		}{
			Updated: len(ids),
		},
	}, nil
}

func (h *ArticleHandlers) BulkSetArchived(ctx context.Context, input *BulkSetArchivedRequest) (*BulkMarkReadResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("User not authenticated")
	}

	if len(input.Body.ArticleIDs) == 0 {
		return nil, huma.Error400BadRequest("Article IDs array is empty")
	}

	// Parse all article IDs
	ids := make([]uuid.UUID, 0, len(input.Body.ArticleIDs))
	for _, idStr := range input.Body.ArticleIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, huma.Error400BadRequest(fmt.Sprintf("Invalid article ID: %s", idStr))
		}
		ids = append(ids, id)
	}

	// Bulk update user's archived state for these articles
	if err := h.userArticleRepo.SetArchivedBulk(ctx, userID, ids, input.Body.IsArchived); err != nil {
		return nil, fmt.Errorf("failed to bulk set archived status: %w", err)
	}

	return &BulkMarkReadResponse{
		Body: struct {
			Updated int `json:"updated"`
		}{
			Updated: len(ids),
		},
	}, nil
}
