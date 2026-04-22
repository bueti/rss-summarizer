package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	"github.com/bbu/rss-summarizer/backend/internal/domain/email_source"
	"github.com/bbu/rss-summarizer/backend/internal/domain/newsletter_filter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type NewsletterFilterHandlers struct {
	filterRepo      newsletter_filter.Repository
	emailSourceRepo email_source.Repository
}

func NewNewsletterFilterHandlers(
	filterRepo newsletter_filter.Repository,
	emailSourceRepo email_source.Repository,
) *NewsletterFilterHandlers {
	return &NewsletterFilterHandlers{
		filterRepo:      filterRepo,
		emailSourceRepo: emailSourceRepo,
	}
}

type CreateNewsletterFilterRequest struct {
	Body struct {
		EmailSourceID  uuid.UUID `json:"email_source_id" doc:"Email source ID"`
		Name           string    `json:"name" minLength:"1" maxLength:"255" doc:"Filter name"`
		SenderPattern  string    `json:"sender_pattern" minLength:"1" maxLength:"500" doc:"Sender pattern (e.g., *@substack.com)"`
		SubjectPattern *string   `json:"subject_pattern,omitempty" maxLength:"500" doc:"Optional subject regex pattern"`
		LabelOrFolder  *string   `json:"label_or_folder,omitempty" maxLength:"255" doc:"Gmail label or Outlook folder"`
	}
}

type CreateNewsletterFilterResponse struct {
	Body newsletter_filter.NewsletterFilter
}

type ListNewsletterFiltersResponse struct {
	Body struct {
		Filters []newsletter_filter.NewsletterFilter `json:"filters"`
	}
}

type GetNewsletterFilterRequest struct {
	ID uuid.UUID `path:"id" doc:"Newsletter filter ID"`
}

type GetNewsletterFilterResponse struct {
	Body newsletter_filter.NewsletterFilter
}

type UpdateNewsletterFilterRequest struct {
	ID   uuid.UUID `path:"id" doc:"Newsletter filter ID"`
	Body struct {
		Name           *string `json:"name,omitempty" minLength:"1" maxLength:"255" doc:"Filter name"`
		SenderPattern  *string `json:"sender_pattern,omitempty" minLength:"1" maxLength:"500" doc:"Sender pattern"`
		SubjectPattern *string `json:"subject_pattern,omitempty" maxLength:"500" doc:"Subject regex pattern"`
		LabelOrFolder  *string `json:"label_or_folder,omitempty" maxLength:"255" doc:"Gmail label or Outlook folder"`
		IsActive       *bool   `json:"is_active,omitempty" doc:"Whether filter is active"`
	}
}

type UpdateNewsletterFilterResponse struct {
	Body newsletter_filter.NewsletterFilter
}

type DeleteNewsletterFilterRequest struct {
	ID uuid.UUID `path:"id" doc:"Newsletter filter ID"`
}

type DeleteNewsletterFilterResponse struct{}

func (h *NewsletterFilterHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-newsletter-filter",
		Method:      http.MethodPost,
		Path:        "/v1/newsletter-filters",
		Summary:     "Create newsletter filter",
		Description: "Creates a new filter for identifying newsletters",
		Tags:        []string{"Newsletter Filters"},
	}, h.CreateNewsletterFilter)

	huma.Register(api, huma.Operation{
		OperationID: "list-newsletter-filters",
		Method:      http.MethodGet,
		Path:        "/v1/newsletter-filters",
		Summary:     "List newsletter filters",
		Description: "Returns all newsletter filters for the current user",
		Tags:        []string{"Newsletter Filters"},
	}, h.ListNewsletterFilters)

	huma.Register(api, huma.Operation{
		OperationID: "get-newsletter-filter",
		Method:      http.MethodGet,
		Path:        "/v1/newsletter-filters/{id}",
		Summary:     "Get newsletter filter",
		Description: "Returns a single newsletter filter by ID",
		Tags:        []string{"Newsletter Filters"},
	}, h.GetNewsletterFilter)

	huma.Register(api, huma.Operation{
		OperationID: "update-newsletter-filter",
		Method:      http.MethodPut,
		Path:        "/v1/newsletter-filters/{id}",
		Summary:     "Update newsletter filter",
		Description: "Updates an existing newsletter filter",
		Tags:        []string{"Newsletter Filters"},
	}, h.UpdateNewsletterFilter)

	huma.Register(api, huma.Operation{
		OperationID: "delete-newsletter-filter",
		Method:      http.MethodDelete,
		Path:        "/v1/newsletter-filters/{id}",
		Summary:     "Delete newsletter filter",
		Description: "Deletes a newsletter filter",
		Tags:        []string{"Newsletter Filters"},
	}, h.DeleteNewsletterFilter)
}

func (h *NewsletterFilterHandlers) CreateNewsletterFilter(ctx context.Context, input *CreateNewsletterFilterRequest) (*CreateNewsletterFilterResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	// Verify email source belongs to user
	emailSource, err := h.emailSourceRepo.FindByID(ctx, input.Body.EmailSourceID)
	if err != nil {
		return nil, huma.Error404NotFound("Email source not found")
	}

	if emailSource.UserID != userID {
		return nil, huma.Error403Forbidden("Not authorized to create filter for this email source")
	}

	// Create filter
	filter, err := h.filterRepo.Create(ctx, &newsletter_filter.CreateNewsletterFilterInput{
		UserID:         userID,
		EmailSourceID:  input.Body.EmailSourceID,
		Name:           input.Body.Name,
		SenderPattern:  input.Body.SenderPattern,
		SubjectPattern: input.Body.SubjectPattern,
		LabelOrFolder:  input.Body.LabelOrFolder,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create newsletter filter: %w", err)
	}

	log.Info().
		Str("user_id", userID.String()).
		Str("filter_id", filter.ID.String()).
		Str("name", filter.Name).
		Msg("Created newsletter filter")

	return &CreateNewsletterFilterResponse{
		Body: *filter,
	}, nil
}

func (h *NewsletterFilterHandlers) ListNewsletterFilters(ctx context.Context, input *struct{}) (*ListNewsletterFiltersResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	filters, err := h.filterRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list newsletter filters: %w", err)
	}

	// Convert to slice of values
	filterValues := make([]newsletter_filter.NewsletterFilter, len(filters))
	for i, filter := range filters {
		filterValues[i] = *filter
	}

	log.Info().
		Str("user_id", userID.String()).
		Int("count", len(filterValues)).
		Msg("Listed newsletter filters")

	return &ListNewsletterFiltersResponse{
		Body: struct {
			Filters []newsletter_filter.NewsletterFilter `json:"filters"`
		}{
			Filters: filterValues,
		},
	}, nil
}

func (h *NewsletterFilterHandlers) GetNewsletterFilter(ctx context.Context, input *GetNewsletterFilterRequest) (*GetNewsletterFilterResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	filter, err := h.filterRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Newsletter filter not found")
	}

	// Verify ownership
	if filter.UserID != userID {
		return nil, huma.Error403Forbidden("Not authorized to access this filter")
	}

	return &GetNewsletterFilterResponse{
		Body: *filter,
	}, nil
}

func (h *NewsletterFilterHandlers) UpdateNewsletterFilter(ctx context.Context, input *UpdateNewsletterFilterRequest) (*UpdateNewsletterFilterResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	// Verify ownership before updating
	filter, err := h.filterRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Newsletter filter not found")
	}

	if filter.UserID != userID {
		return nil, huma.Error403Forbidden("Not authorized to update this filter")
	}

	// Update filter
	updated, err := h.filterRepo.Update(ctx, input.ID, &newsletter_filter.UpdateNewsletterFilterInput{
		Name:           input.Body.Name,
		SenderPattern:  input.Body.SenderPattern,
		SubjectPattern: input.Body.SubjectPattern,
		LabelOrFolder:  input.Body.LabelOrFolder,
		IsActive:       input.Body.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update newsletter filter: %w", err)
	}

	log.Info().
		Str("user_id", userID.String()).
		Str("filter_id", input.ID.String()).
		Msg("Updated newsletter filter")

	return &UpdateNewsletterFilterResponse{
		Body: *updated,
	}, nil
}

func (h *NewsletterFilterHandlers) DeleteNewsletterFilter(ctx context.Context, input *DeleteNewsletterFilterRequest) (*DeleteNewsletterFilterResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	// Verify ownership before deleting
	filter, err := h.filterRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Newsletter filter not found")
	}

	if filter.UserID != userID {
		return nil, huma.Error403Forbidden("Not authorized to delete this filter")
	}

	if err := h.filterRepo.Delete(ctx, input.ID); err != nil {
		return nil, fmt.Errorf("failed to delete newsletter filter: %w", err)
	}

	log.Info().
		Str("user_id", userID.String()).
		Str("filter_id", input.ID.String()).
		Str("name", filter.Name).
		Msg("Deleted newsletter filter")

	return &DeleteNewsletterFilterResponse{}, nil
}
