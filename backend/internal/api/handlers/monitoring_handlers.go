package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/domain/email_source"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	workflowv1 "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

type MonitoringHandlers struct {
	temporalClient  client.Client
	articleRepo     repository.ArticleRepository
	feedRepo        repository.FeedRepository
	emailSourceRepo email_source.Repository
}

func NewMonitoringHandlers(
	temporalClient client.Client,
	articleRepo repository.ArticleRepository,
	feedRepo repository.FeedRepository,
	emailSourceRepo email_source.Repository,
) *MonitoringHandlers {
	return &MonitoringHandlers{
		temporalClient:  temporalClient,
		articleRepo:     articleRepo,
		feedRepo:        feedRepo,
		emailSourceRepo: emailSourceRepo,
	}
}

type WorkflowInfo struct {
	WorkflowID      string     `json:"workflow_id"`
	WorkflowType    string     `json:"workflow_type"`
	Status          string     `json:"status"` // Running, Completed, Failed, Terminated
	StartTime       *time.Time `json:"start_time,omitempty"`
	CloseTime       *time.Time `json:"close_time,omitempty"`
	ExecutionTimeMs *int64     `json:"execution_time_ms,omitempty"` // Duration in ms
	ArticleID       *string    `json:"article_id,omitempty"`
	ArticleTitle    *string    `json:"article_title,omitempty"`
	SourceName      *string    `json:"source_name,omitempty"` // Feed title or email source
	SourceType      *string    `json:"source_type,omitempty"` // "rss" or "email"
}

type WorkflowOverviewResponse struct {
	Body struct {
		Running          []WorkflowInfo `json:"running"`
		RecentSuccess    []WorkflowInfo `json:"recent_success"`
		RecentFailed     []WorkflowInfo `json:"recent_failed"`
		TotalRunning     int            `json:"total_running"`
		TotalSuccess24h  int            `json:"total_success_24h"`
		TotalFailed24h   int            `json:"total_failed_24h"`
		LastUpdated      time.Time      `json:"last_updated"`
	}
}

func (h *MonitoringHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-workflow-overview",
		Method:      http.MethodGet,
		Path:        "/v1/monitoring/workflows",
		Summary:     "Get Temporal workflow overview",
		Description: "Query Temporal for running and recent workflows with article details",
		Tags:        []string{"Monitoring"},
	}, h.GetWorkflowOverview)
}

func (h *MonitoringHandlers) GetWorkflowOverview(
	ctx context.Context,
	_ *struct{},
) (*WorkflowOverviewResponse, error) {
	// Query running workflows
	runningWorkflows, err := h.queryTemporalWorkflows(ctx, "ExecutionStatus = 'Running'", 50)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query running workflows")
		return nil, huma.Error500InternalServerError("Failed to query running workflows", err)
	}

	// Query completed workflows (standard visibility doesn't support time filtering)
	completedWorkflows, err := h.queryTemporalWorkflows(ctx, "ExecutionStatus = 'Completed'", 100)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query completed workflows")
		return nil, huma.Error500InternalServerError("Failed to query completed workflows", err)
	}

	// Query failed workflows (standard visibility doesn't support time filtering)
	failedWorkflows, err := h.queryTemporalWorkflows(ctx, "ExecutionStatus = 'Failed'", 100)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query failed workflows")
		return nil, huma.Error500InternalServerError("Failed to query failed workflows", err)
	}

	// Filter by time (last 24h) in application since standard visibility doesn't support time queries
	cutoffTime := time.Now().Add(-24 * time.Hour)
	completedWorkflows = h.filterByCloseTime(completedWorkflows, cutoffTime)
	failedWorkflows = h.filterByCloseTime(failedWorkflows, cutoffTime)

	// Enrich with article data
	runningInfos := h.enrichWithArticleData(ctx, runningWorkflows)
	completedInfos := h.enrichWithArticleData(ctx, completedWorkflows)
	failedInfos := h.enrichWithArticleData(ctx, failedWorkflows)

	// Build response
	resp := &WorkflowOverviewResponse{}
	resp.Body.Running = runningInfos
	resp.Body.RecentSuccess = completedInfos
	resp.Body.RecentFailed = failedInfos
	resp.Body.TotalRunning = len(runningInfos)
	resp.Body.TotalSuccess24h = len(completedInfos)
	resp.Body.TotalFailed24h = len(failedInfos)
	resp.Body.LastUpdated = time.Now()

	return resp, nil
}

// filterByCloseTime filters workflows that closed after the cutoff time
func (h *MonitoringHandlers) filterByCloseTime(
	executions []*workflowv1.WorkflowExecutionInfo,
	cutoffTime time.Time,
) []*workflowv1.WorkflowExecutionInfo {
	var filtered []*workflowv1.WorkflowExecutionInfo
	for _, exec := range executions {
		if exec.CloseTime != nil && exec.CloseTime.AsTime().After(cutoffTime) {
			filtered = append(filtered, exec)
		}
	}
	return filtered
}

// queryTemporalWorkflows queries Temporal with the given query string and limit
func (h *MonitoringHandlers) queryTemporalWorkflows(
	ctx context.Context,
	query string,
	limit int,
) ([]*workflowv1.WorkflowExecutionInfo, error) {
	// List workflows with query
	resp, err := h.temporalClient.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Query:    query,
		PageSize: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	return resp.Executions, nil
}

// enrichWithArticleData enriches workflow execution info with article data from database
func (h *MonitoringHandlers) enrichWithArticleData(
	ctx context.Context,
	executions []*workflowv1.WorkflowExecutionInfo,
) []WorkflowInfo {
	var results []WorkflowInfo

	for _, exec := range executions {
		info := WorkflowInfo{
			WorkflowID:   exec.Execution.WorkflowId,
			WorkflowType: exec.Type.Name,
			Status:       exec.Status.String(),
		}

		// Set timestamps
		if exec.StartTime != nil {
			startTime := exec.StartTime.AsTime()
			info.StartTime = &startTime
		}
		if exec.CloseTime != nil {
			closeTime := exec.CloseTime.AsTime()
			info.CloseTime = &closeTime

			// Calculate execution time
			if exec.StartTime != nil {
				executionMs := closeTime.Sub(exec.StartTime.AsTime()).Milliseconds()
				info.ExecutionTimeMs = &executionMs
			}
		}

		// Parse workflow ID to extract entity information
		entityType, entityID := h.parseWorkflowID(exec.Execution.WorkflowId)

		// Enrich with article data if this is an article workflow
		if entityType == "article" && entityID != nil {
			articleIDStr := entityID.String()
			info.ArticleID = &articleIDStr

			// Fetch article from database
			article, err := h.articleRepo.FindByID(ctx, *entityID)
			if err != nil {
				log.Warn().Err(err).Str("articleID", entityID.String()).Msg("Article not found for workflow")
			} else {
				info.ArticleTitle = &article.Title
				info.SourceType = &article.SourceType

				// Get source name (feed or email)
				if article.FeedID != nil {
					if feed, err := h.feedRepo.FindByID(ctx, *article.FeedID); err == nil {
						info.SourceName = &feed.Title
					}
				} else if article.EmailSourceID != nil {
					if emailSource, err := h.emailSourceRepo.FindByID(ctx, *article.EmailSourceID); err == nil {
						info.SourceName = &emailSource.EmailAddress
					}
				}
			}
		}

		// Enrich with feed data if this is a feed workflow
		if entityType == "feed" && entityID != nil {
			if feed, err := h.feedRepo.FindByID(ctx, *entityID); err == nil {
				sourceName := feed.Title
				sourceType := "rss"
				info.SourceName = &sourceName
				info.SourceType = &sourceType
			}
		}

		// Enrich with email source data if this is an email workflow
		if entityType == "email" && entityID != nil {
			if emailSource, err := h.emailSourceRepo.FindByID(ctx, *entityID); err == nil {
				sourceName := emailSource.EmailAddress
				sourceType := "email"
				info.SourceName = &sourceName
				info.SourceType = &sourceType
			}
		}

		results = append(results, info)
	}

	return results
}

// parseWorkflowID extracts entity type and ID from workflow ID
// Returns (entityType, entityID) or ("system", nil) for system workflows
func (h *MonitoringHandlers) parseWorkflowID(workflowID string) (string, *uuid.UUID) {
	patterns := map[string]*regexp.Regexp{
		"article": regexp.MustCompile(`^summarize-article-([0-9a-f-]{36})$`),
		"feed":    regexp.MustCompile(`^process-feed-([0-9a-f-]{36})$`),
		"email":   regexp.MustCompile(`^process-email-source-([0-9a-f-]{36})$`),
	}

	for entityType, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(workflowID); len(matches) == 2 {
			if id, err := uuid.Parse(matches[1]); err == nil {
				return entityType, &id
			}
		}
	}

	// System workflows (feed-poller-*, email-poller-*)
	return "system", nil
}
