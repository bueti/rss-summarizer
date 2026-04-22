package llmconfig

import (
	"time"

	"github.com/google/uuid"
)

// LLMConfig represents the global LLM configuration for article summarization
type LLMConfig struct {
	ID             uuid.UUID `db:"id" json:"id"`
	Provider       string    `db:"provider" json:"provider"`
	Model          string    `db:"model" json:"model"`
	APIURL         string    `db:"api_url" json:"api_url"`
	APIKey         string    `db:"api_key" json:"-"` // Never serialize to JSON
	SingletonGuard int       `db:"singleton_guard" json:"-"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// UpdateLLMConfigInput represents the input for updating LLM configuration
type UpdateLLMConfigInput struct {
	Provider *string `json:"provider,omitempty"`
	Model    *string `json:"model,omitempty"`
	APIURL   *string `json:"api_url,omitempty"`
	APIKey   *string `json:"api_key,omitempty"` // Optional - only update if provided
}
