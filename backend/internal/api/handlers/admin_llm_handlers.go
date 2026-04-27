package handlers

import (
	"context"
	"net/http"

	"github.com/bbu/rss-summarizer/backend/internal/domain/llmconfig"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/danielgtaylor/huma/v2"
)

type AdminLLMHandlers struct {
	repo repository.LLMConfigRepository
}

func NewAdminLLMHandlers(repo repository.LLMConfigRepository) *AdminLLMHandlers {
	return &AdminLLMHandlers{repo: repo}
}

type LLMConfigResponse struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	APIURL    string `json:"api_url"`
	HasAPIKey bool   `json:"has_api_key"` // Indicates if API key is set, but doesn't reveal it
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type GetLLMConfigResponse struct {
	Body LLMConfigResponse
}

type UpdateLLMConfigRequest struct {
	Body struct {
		Provider string  `json:"provider,omitempty" enum:"openai,anthropic,poolside" doc:"LLM provider to use"`
		Model    string  `json:"model,omitempty" minLength:"1" maxLength:"100" doc:"LLM model name"`
		APIURL   string  `json:"api_url,omitempty" maxLength:"500" doc:"API URL for the LLM provider"`
		APIKey   *string `json:"api_key,omitempty" doc:"LLM API key (optional, only provide to update)"`
	}
}

type UpdateLLMConfigResponse struct {
	Body LLMConfigResponse
}

func (h *AdminLLMHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-llm-config",
		Method:      http.MethodGet,
		Path:        "/v1/admin/llm-config",
		Summary:     "Get global LLM configuration",
		Description: "Get the current global LLM configuration for article summarization (admin only)",
		Tags:        []string{"Admin"},
	}, h.GetConfig)

	huma.Register(api, huma.Operation{
		OperationID: "update-llm-config",
		Method:      http.MethodPut,
		Path:        "/v1/admin/llm-config",
		Summary:     "Update global LLM configuration",
		Description: "Update the global LLM configuration for article summarization (admin only)",
		Tags:        []string{"Admin"},
	}, h.UpdateConfig)
}

func (h *AdminLLMHandlers) GetConfig(ctx context.Context, _ *struct{}) (*GetLLMConfigResponse, error) {
	// Admin check is enforced by AdminMiddleware on /v1/admin/*.
	config, err := h.repo.Get(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get LLM config", err)
	}

	response := LLMConfigResponse{
		ID:        config.ID.String(),
		Provider:  config.Provider,
		Model:     config.Model,
		APIURL:    config.APIURL,
		HasAPIKey: config.APIKey != "",
		CreatedAt: config.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: config.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return &GetLLMConfigResponse{Body: response}, nil
}

func (h *AdminLLMHandlers) UpdateConfig(ctx context.Context, input *UpdateLLMConfigRequest) (*UpdateLLMConfigResponse, error) {
	// Admin check is enforced by AdminMiddleware on /v1/admin/*.
	updateInput := &llmconfig.UpdateLLMConfigInput{
		Provider: nil,
		Model:    nil,
		APIURL:   nil,
		APIKey:   input.Body.APIKey,
	}

	// Only set fields that were provided
	if input.Body.Provider != "" {
		updateInput.Provider = &input.Body.Provider
	}
	if input.Body.Model != "" {
		updateInput.Model = &input.Body.Model
	}
	if input.Body.APIURL != "" {
		updateInput.APIURL = &input.Body.APIURL
	}

	config, err := h.repo.Update(ctx, updateInput)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update LLM config", err)
	}

	response := LLMConfigResponse{
		ID:        config.ID.String(),
		Provider:  config.Provider,
		Model:     config.Model,
		APIURL:    config.APIURL,
		HasAPIKey: config.APIKey != "",
		CreatedAt: config.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: config.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return &UpdateLLMConfigResponse{Body: response}, nil
}
