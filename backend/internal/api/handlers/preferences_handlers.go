package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	domainerrors "github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/bbu/rss-summarizer/backend/internal/domain/preferences"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/danielgtaylor/huma/v2"
)

type PreferencesHandlers struct {
	repo repository.PreferencesRepository
}

func NewPreferencesHandlers(repo repository.PreferencesRepository) *PreferencesHandlers {
	return &PreferencesHandlers{repo: repo}
}

type PreferencesResponse struct {
	ID                  string `json:"id"`
	UserID              string `json:"user_id"`
	DefaultPollInterval int    `json:"default_poll_interval"`
	MaxArticlesPerFeed  int    `json:"max_articles_per_feed"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

type GetPreferencesResponse struct {
	Body PreferencesResponse
}

type UpdatePreferencesRequest struct {
	Body struct {
		DefaultPollInterval int `json:"default_poll_interval" minimum:"15" maximum:"1440" doc:"Default poll interval in minutes"`
		MaxArticlesPerFeed  int `json:"max_articles_per_feed" minimum:"1" maximum:"100" doc:"Maximum articles to fetch per feed per poll"`
	}
}

type UpdatePreferencesResponse struct {
	Body PreferencesResponse
}

func (h *PreferencesHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-user-preferences",
		Method:      http.MethodGet,
		Path:        "/v1/user/preferences",
		Summary:     "Get user preferences",
		Description: "Get the current user's preferences",
		Tags:        []string{"Preferences"},
	}, h.GetPreferences)

	huma.Register(api, huma.Operation{
		OperationID: "update-user-preferences",
		Method:      http.MethodPut,
		Path:        "/v1/user/preferences",
		Summary:     "Update user preferences",
		Description: "Update the current user's preferences",
		Tags:        []string{"Preferences"},
	}, h.UpdatePreferences)
}

func (h *PreferencesHandlers) GetPreferences(ctx context.Context, _ *struct{}) (*GetPreferencesResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	prefs, err := h.repo.GetByUserID(ctx, userID)
	if err != nil {
		if _, ok := errors.AsType[*domainerrors.NotFoundError](err); ok {
			return nil, huma.Error404NotFound("Preferences not found")
		}
		return nil, huma.Error500InternalServerError("Failed to get preferences", err)
	}

	response := PreferencesResponse{
		ID:                  prefs.ID.String(),
		UserID:              prefs.UserID.String(),
		DefaultPollInterval: prefs.DefaultPollInterval,
		MaxArticlesPerFeed:  prefs.MaxArticlesPerFeed,
		CreatedAt:           prefs.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:           prefs.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return &GetPreferencesResponse{Body: response}, nil
}

func (h *PreferencesHandlers) UpdatePreferences(ctx context.Context, input *UpdatePreferencesRequest) (*UpdatePreferencesResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	prefs := &preferences.UserPreferences{
		UserID:              userID,
		DefaultPollInterval: input.Body.DefaultPollInterval,
		MaxArticlesPerFeed:  input.Body.MaxArticlesPerFeed,
	}

	if err := h.repo.Upsert(ctx, prefs); err != nil {
		return nil, huma.Error500InternalServerError("Failed to update preferences", err)
	}

	response := PreferencesResponse{
		ID:                  prefs.ID.String(),
		UserID:              prefs.UserID.String(),
		DefaultPollInterval: prefs.DefaultPollInterval,
		MaxArticlesPerFeed:  prefs.MaxArticlesPerFeed,
		CreatedAt:           prefs.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:           prefs.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return &UpdatePreferencesResponse{Body: response}, nil
}
