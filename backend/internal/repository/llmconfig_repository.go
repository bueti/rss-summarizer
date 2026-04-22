package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bbu/rss-summarizer/backend/internal/crypto"
	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/bbu/rss-summarizer/backend/internal/domain/llmconfig"
)

type LLMConfigRepository interface {
	Get(ctx context.Context) (*llmconfig.LLMConfig, error)
	Update(ctx context.Context, input *llmconfig.UpdateLLMConfigInput) (*llmconfig.LLMConfig, error)
}

type llmConfigRepository struct {
	db     *database.DB
	crypto *crypto.Service
}

func NewLLMConfigRepository(db *database.DB, crypto *crypto.Service) LLMConfigRepository {
	return &llmConfigRepository{
		db:     db,
		crypto: crypto,
	}
}

func (r *llmConfigRepository) Get(ctx context.Context) (*llmconfig.LLMConfig, error) {
	var config llmconfig.LLMConfig
	query := `SELECT * FROM llm_config LIMIT 1`

	if err := r.db.GetContext(ctx, &config, query); err != nil {
		if err == sql.ErrNoRows {
			return nil, &errors.NotFoundError{Resource: "llm_config", ID: "singleton"}
		}
		return nil, fmt.Errorf("failed to get llm config: %w", err)
	}

	// Decrypt API key if present
	if config.APIKey != "" {
		decrypted, err := r.crypto.Decrypt(config.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt API key: %w", err)
		}
		config.APIKey = decrypted
	}

	return &config, nil
}

func (r *llmConfigRepository) Update(ctx context.Context, input *llmconfig.UpdateLLMConfigInput) (*llmconfig.LLMConfig, error) {
	// Get current config
	config, err := r.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if input.Provider != nil {
		config.Provider = *input.Provider
	}
	if input.Model != nil {
		config.Model = *input.Model
	}
	if input.APIURL != nil {
		config.APIURL = *input.APIURL
	}

	// Handle API key separately (encrypt if provided)
	var encryptedAPIKey *string
	if input.APIKey != nil && *input.APIKey != "" {
		encrypted, err := r.crypto.Encrypt(*input.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		encryptedAPIKey = &encrypted
	}

	// Update the database
	query := `
		UPDATE llm_config
		SET provider = $1, model = $2, api_url = $3, updated_at = NOW()
	`
	args := []interface{}{config.Provider, config.Model, config.APIURL}

	if encryptedAPIKey != nil {
		query += `, api_key = $4`
		args = append(args, *encryptedAPIKey)
	}

	query += ` WHERE singleton_guard = 1 RETURNING updated_at`

	err = r.db.QueryRowContext(ctx, query, args...).Scan(&config.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update llm config: %w", err)
	}

	// Return updated config (with decrypted API key)
	return r.Get(ctx)
}
