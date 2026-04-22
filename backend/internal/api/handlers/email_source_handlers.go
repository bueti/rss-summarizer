package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	"github.com/bbu/rss-summarizer/backend/internal/domain/email_source"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type EmailSourceHandlers struct {
	emailSourceRepo email_source.Repository
}

func NewEmailSourceHandlers(emailSourceRepo email_source.Repository) *EmailSourceHandlers {
	return &EmailSourceHandlers{
		emailSourceRepo: emailSourceRepo,
	}
}

type ListEmailSourcesResponse struct {
	Body struct {
		EmailSources []email_source.EmailSource `json:"email_sources"`
	}
}

type GetEmailSourceRequest struct {
	ID uuid.UUID `path:"id" doc:"Email source ID"`
}

type GetEmailSourceResponse struct {
	Body email_source.EmailSource
}

type DeleteEmailSourceRequest struct {
	ID uuid.UUID `path:"id" doc:"Email source ID"`
}

type DeleteEmailSourceResponse struct{}

func (h *EmailSourceHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-email-sources",
		Method:      http.MethodGet,
		Path:        "/v1/email-sources",
		Summary:     "List email sources",
		Description: "Returns all email sources for the current user",
		Tags:        []string{"Email Sources"},
	}, h.ListEmailSources)

	huma.Register(api, huma.Operation{
		OperationID: "get-email-source",
		Method:      http.MethodGet,
		Path:        "/v1/email-sources/{id}",
		Summary:     "Get email source",
		Description: "Returns a single email source by ID",
		Tags:        []string{"Email Sources"},
	}, h.GetEmailSource)

	huma.Register(api, huma.Operation{
		OperationID: "delete-email-source",
		Method:      http.MethodDelete,
		Path:        "/v1/email-sources/{id}",
		Summary:     "Delete email source",
		Description: "Disconnects and deletes an email source",
		Tags:        []string{"Email Sources"},
	}, h.DeleteEmailSource)
}

func (h *EmailSourceHandlers) ListEmailSources(ctx context.Context, input *struct{}) (*ListEmailSourcesResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	sources, err := h.emailSourceRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list email sources: %w", err)
	}

	// Convert to response (exclude tokens from JSON)
	emailSources := make([]email_source.EmailSource, len(sources))
	for i, source := range sources {
		emailSources[i] = *source
	}

	log.Info().
		Str("user_id", userID.String()).
		Int("count", len(emailSources)).
		Msg("Listed email sources")

	return &ListEmailSourcesResponse{
		Body: struct {
			EmailSources []email_source.EmailSource `json:"email_sources"`
		}{
			EmailSources: emailSources,
		},
	}, nil
}

func (h *EmailSourceHandlers) GetEmailSource(ctx context.Context, input *GetEmailSourceRequest) (*GetEmailSourceResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	source, err := h.emailSourceRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Email source not found")
	}

	// Verify ownership
	if source.UserID != userID {
		return nil, huma.Error403Forbidden("Not authorized to access this email source")
	}

	return &GetEmailSourceResponse{
		Body: *source,
	}, nil
}

func (h *EmailSourceHandlers) DeleteEmailSource(ctx context.Context, input *DeleteEmailSourceRequest) (*DeleteEmailSourceResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	// Verify ownership before deleting
	source, err := h.emailSourceRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Email source not found")
	}

	if source.UserID != userID {
		return nil, huma.Error403Forbidden("Not authorized to delete this email source")
	}

	if err := h.emailSourceRepo.Delete(ctx, input.ID); err != nil {
		return nil, fmt.Errorf("failed to delete email source: %w", err)
	}

	log.Info().
		Str("user_id", userID.String()).
		Str("email_source_id", input.ID.String()).
		Str("email", source.EmailAddress).
		Msg("Deleted email source")

	return &DeleteEmailSourceResponse{}, nil
}
