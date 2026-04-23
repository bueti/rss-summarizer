package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	"github.com/bbu/rss-summarizer/backend/internal/domain/topic"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type TopicHandlers struct {
	repo repository.TopicRepository
}

func NewTopicHandlers(repo repository.TopicRepository) *TopicHandlers {
	return &TopicHandlers{repo: repo}
}

type ListTopicsResponse struct {
	Body struct {
		Topics []topic.TopicWithPreference `json:"topics"`
	}
}

type GetTopicRequest struct {
	ID string `path:"id" format:"uuid" doc:"Topic ID"`
}

type GetTopicResponse struct {
	Body topic.TopicWithPreference
}

type CreateTopicRequest struct {
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"255" doc:"Topic name"`
	}
}

type CreateTopicResponse struct {
	Body topic.Topic
}

type UpdateTopicPreferenceRequest struct {
	ID   string `path:"id" format:"uuid" doc:"Topic ID"`
	Body struct {
		Preference string `json:"preference" enum:"high,normal,hide" doc:"Topic preference"`
	}
}

type UpdateTopicPreferenceResponse struct {
	Body topic.TopicWithPreference
}

type DeleteTopicRequest struct {
	ID string `path:"id" format:"uuid" doc:"Topic ID"`
}

func (h *TopicHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-topics",
		Method:      http.MethodGet,
		Path:        "/v1/topics",
		Summary:     "List topics",
		Description: "Get all topics from user's subscribed feeds with their preferences",
		Tags:        []string{"Topics"},
	}, h.ListTopics)

	huma.Register(api, huma.Operation{
		OperationID: "get-topic",
		Method:      http.MethodGet,
		Path:        "/v1/topics/{id}",
		Summary:     "Get topic by ID",
		Description: "Get a specific topic with user's preference",
		Tags:        []string{"Topics"},
	}, h.GetTopic)

	huma.Register(api, huma.Operation{
		OperationID: "create-topic",
		Method:      http.MethodPost,
		Path:        "/v1/topics",
		Summary:     "Create custom topic",
		Description: "Create a new custom topic",
		Tags:        []string{"Topics"},
	}, h.CreateTopic)

	huma.Register(api, huma.Operation{
		OperationID: "update-topic-preference",
		Method:      http.MethodPut,
		Path:        "/v1/topics/{id}/preference",
		Summary:     "Update topic preference",
		Description: "Update the user's preference for a topic",
		Tags:        []string{"Topics"},
	}, h.UpdateTopicPreference)

	huma.Register(api, huma.Operation{
		OperationID: "delete-topic",
		Method:      http.MethodDelete,
		Path:        "/v1/topics/{id}",
		Summary:     "Delete custom topic",
		Description: "Delete a custom topic (only if it has no user preferences)",
		Tags:        []string{"Topics"},
	}, h.DeleteTopic)
}

func (h *TopicHandlers) ListTopics(ctx context.Context, _ *struct{}) (*ListTopicsResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	topics, err := h.repo.FindTopicsWithPreferences(ctx, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list topics", err)
	}

	resp := &ListTopicsResponse{}
	resp.Body.Topics = make([]topic.TopicWithPreference, 0, len(topics))
	for _, t := range topics {
		resp.Body.Topics = append(resp.Body.Topics, *t)
	}

	return resp, nil
}

func (h *TopicHandlers) GetTopic(ctx context.Context, input *GetTopicRequest) (*GetTopicResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	id, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid topic ID")
	}

	// Get the global topic
	t, err := h.repo.GetTopicByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error404NotFound("Topic not found")
		}
		return nil, huma.Error500InternalServerError("Failed to get topic", err)
	}

	// Get user's preference (or use 'normal' as default)
	pref, err := h.repo.GetPreference(ctx, userID, id)
	preference := "normal"
	if err == nil {
		preference = pref.Preference
	}

	return &GetTopicResponse{
		Body: topic.TopicWithPreference{
			ID:         t.ID,
			Name:       t.Name,
			IsCustom:   t.IsCustom,
			Preference: preference,
			CreatedAt:  t.CreatedAt,
			UpdatedAt:  t.UpdatedAt,
		},
	}, nil
}

func (h *TopicHandlers) CreateTopic(ctx context.Context, input *CreateTopicRequest) (*CreateTopicResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	// Check if topic already exists
	existing, err := h.repo.GetTopicByName(ctx, input.Body.Name)
	if err == nil {
		// Topic exists, just return it
		return &CreateTopicResponse{Body: *existing}, nil
	}

	// Create new custom topic
	t, err := h.repo.CreateCustomTopic(ctx, input.Body.Name)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to create topic", err)
	}

	// Seed the creator's preference to 'normal'. If this fails the frontend
	// still renders the topic because queries default missing preferences to
	// 'normal' — so log and continue rather than failing the create.
	if err := h.repo.UpsertPreference(ctx, userID, t.ID, "normal"); err != nil {
		log.Error().Err(err).Str("topic_id", t.ID.String()).Str("user_id", userID.String()).Msg("Failed to seed topic preference")
	}

	return &CreateTopicResponse{Body: *t}, nil
}

func (h *TopicHandlers) UpdateTopicPreference(ctx context.Context, input *UpdateTopicPreferenceRequest) (*UpdateTopicPreferenceResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	id, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid topic ID")
	}

	if !topic.IsValidPreference(input.Body.Preference) {
		return nil, huma.Error400BadRequest("Invalid preference value")
	}

	// Verify topic exists
	t, err := h.repo.GetTopicByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error404NotFound("Topic not found")
		}
		return nil, huma.Error500InternalServerError("Failed to get topic", err)
	}

	// Update user's preference
	if err := h.repo.UpsertPreference(ctx, userID, id, input.Body.Preference); err != nil {
		return nil, huma.Error500InternalServerError("Failed to update preference", err)
	}

	return &UpdateTopicPreferenceResponse{
		Body: topic.TopicWithPreference{
			ID:         t.ID,
			Name:       t.Name,
			IsCustom:   t.IsCustom,
			Preference: input.Body.Preference,
			CreatedAt:  t.CreatedAt,
			UpdatedAt:  t.UpdatedAt,
		},
	}, nil
}

func (h *TopicHandlers) DeleteTopic(ctx context.Context, input *DeleteTopicRequest) (*struct{}, error) {
	_, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	id, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid topic ID")
	}

	if err := h.repo.DeleteCustomTopic(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error404NotFound("Topic not found or not custom")
		}
		return nil, huma.Error500InternalServerError("Failed to delete topic", err)
	}

	return &struct{}{}, nil
}
