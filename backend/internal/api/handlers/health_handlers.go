package handlers

import (
	"context"
	"net/http"

	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/danielgtaylor/huma/v2"
)

type HealthHandlers struct {
	db *database.DB
}

func NewHealthHandlers(db *database.DB) *HealthHandlers {
	return &HealthHandlers{db: db}
}

type HealthResponse struct {
	Body struct {
		Status   string `json:"status" doc:"Overall health status"`
		Database string `json:"database" doc:"Database health status"`
	}
}

func (h *HealthHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check",
		Description: "Check the health status of the API and its dependencies",
		Tags:        []string{"System"},
	}, h.HealthCheck)
}

func (h *HealthHandlers) HealthCheck(ctx context.Context, input *struct{}) (*HealthResponse, error) {
	response := &HealthResponse{}
	response.Body.Status = "healthy"
	response.Body.Database = "healthy"

	// Check database
	if err := h.db.PingContext(ctx); err != nil {
		response.Body.Database = "unhealthy: " + err.Error()
		response.Body.Status = "degraded"
	}

	return response, nil
}
