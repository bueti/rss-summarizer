package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bbu/rss-summarizer/backend/internal/crypto"
	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/email_source"
	"github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type EmailSourceRepository struct {
	db     *database.DB
	crypto *crypto.Service
}

func NewEmailSourceRepository(db *database.DB, crypto *crypto.Service) email_source.Repository {
	return &EmailSourceRepository{
		db:     db,
		crypto: crypto,
	}
}

func (r *EmailSourceRepository) Create(ctx context.Context, input *email_source.CreateEmailSourceInput) (*email_source.EmailSource, error) {
	// Encrypt tokens before storage
	encryptedAccessToken, err := r.crypto.Encrypt(input.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %w", err)
	}

	encryptedRefreshToken, err := r.crypto.Encrypt(input.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	source := &email_source.EmailSource{
		ID:             uuid.New(),
		UserID:         input.UserID,
		EmailAddress:   input.EmailAddress,
		Provider:       input.Provider,
		AccessToken:    encryptedAccessToken,
		RefreshToken:   encryptedRefreshToken,
		TokenExpiresAt: input.TokenExpiresAt,
		IsActive:       true,
	}

	query := `
		INSERT INTO email_sources (
			id, user_id, email_address, provider, access_token, refresh_token,
			token_expires_at, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err = r.db.QueryRowContext(ctx, query,
		source.ID, source.UserID, source.EmailAddress, source.Provider,
		source.AccessToken, source.RefreshToken, source.TokenExpiresAt, source.IsActive,
	).Scan(&source.CreatedAt, &source.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return nil, &errors.DuplicateError{
				Resource: "email_source",
				Field:    "email_address",
				Value:    input.EmailAddress,
			}
		}
		return nil, fmt.Errorf("failed to create email source: %w", err)
	}

	return source, nil
}

func (r *EmailSourceRepository) FindByID(ctx context.Context, id uuid.UUID) (*email_source.EmailSource, error) {
	var source email_source.EmailSource
	query := `SELECT * FROM email_sources WHERE id = $1`

	if err := r.db.GetContext(ctx, &source, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, &errors.NotFoundError{Resource: "email_source", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to find email source: %w", err)
	}

	// Decrypt tokens
	if err := r.decryptTokens(&source); err != nil {
		return nil, err
	}

	return &source, nil
}

func (r *EmailSourceRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*email_source.EmailSource, error) {
	var sources []*email_source.EmailSource
	query := `SELECT * FROM email_sources WHERE user_id = $1 ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &sources, query, userID); err != nil {
		return nil, fmt.Errorf("failed to find email sources by user ID: %w", err)
	}

	// Decrypt tokens for all sources
	for _, source := range sources {
		if err := r.decryptTokens(source); err != nil {
			return nil, err
		}
	}

	return sources, nil
}

func (r *EmailSourceRepository) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*email_source.EmailSource, error) {
	var sources []*email_source.EmailSource
	query := `SELECT * FROM email_sources WHERE user_id = $1 AND is_active = true ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &sources, query, userID); err != nil {
		return nil, fmt.Errorf("failed to find active email sources by user ID: %w", err)
	}

	// Decrypt tokens for all sources
	for _, source := range sources {
		if err := r.decryptTokens(source); err != nil {
			return nil, err
		}
	}

	return sources, nil
}

func (r *EmailSourceRepository) FindAllActive(ctx context.Context) ([]*email_source.EmailSource, error) {
	var sources []*email_source.EmailSource
	query := `SELECT * FROM email_sources WHERE is_active = true ORDER BY last_fetched_at ASC NULLS FIRST`

	if err := r.db.SelectContext(ctx, &sources, query); err != nil {
		return nil, fmt.Errorf("failed to find all active email sources: %w", err)
	}

	// Decrypt tokens for all sources
	for _, source := range sources {
		if err := r.decryptTokens(source); err != nil {
			return nil, err
		}
	}

	return sources, nil
}

func (r *EmailSourceRepository) Update(ctx context.Context, id uuid.UUID, input *email_source.UpdateEmailSourceInput) (*email_source.EmailSource, error) {
	// Start building update query dynamically
	query := `UPDATE email_sources SET updated_at = NOW()`
	args := []any{}
	argPos := 1

	if input.AccessToken != nil {
		encrypted, err := r.crypto.Encrypt(*input.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt access token: %w", err)
		}
		query += fmt.Sprintf(`, access_token = $%d`, argPos)
		args = append(args, encrypted)
		argPos++
	}

	if input.RefreshToken != nil {
		encrypted, err := r.crypto.Encrypt(*input.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
		query += fmt.Sprintf(`, refresh_token = $%d`, argPos)
		args = append(args, encrypted)
		argPos++
	}

	if input.TokenExpiresAt != nil {
		query += fmt.Sprintf(`, token_expires_at = $%d`, argPos)
		args = append(args, *input.TokenExpiresAt)
		argPos++
	}

	if input.LastFetchedAt != nil {
		query += fmt.Sprintf(`, last_fetched_at = $%d`, argPos)
		args = append(args, *input.LastFetchedAt)
		argPos++
	}

	if input.IsActive != nil {
		query += fmt.Sprintf(`, is_active = $%d`, argPos)
		args = append(args, *input.IsActive)
		argPos++
	}

	if input.LastError != nil {
		query += fmt.Sprintf(`, last_error = $%d`, argPos)
		args = append(args, *input.LastError)
		argPos++
	}

	query += fmt.Sprintf(` WHERE id = $%d`, argPos)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update email source: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return nil, &errors.NotFoundError{Resource: "email_source", ID: id.String()}
	}

	// Return updated source
	return r.FindByID(ctx, id)
}

func (r *EmailSourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM email_sources WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete email source: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return &errors.NotFoundError{Resource: "email_source", ID: id.String()}
	}

	return nil
}

// decryptTokens decrypts the access and refresh tokens in place
func (r *EmailSourceRepository) decryptTokens(source *email_source.EmailSource) error {
	if source.AccessToken != "" {
		decrypted, err := r.crypto.Decrypt(source.AccessToken)
		if err != nil {
			return fmt.Errorf("failed to decrypt access token: %w", err)
		}
		source.AccessToken = decrypted
	}

	if source.RefreshToken != "" {
		decrypted, err := r.crypto.Decrypt(source.RefreshToken)
		if err != nil {
			return fmt.Errorf("failed to decrypt refresh token: %w", err)
		}
		source.RefreshToken = decrypted
	}

	return nil
}
