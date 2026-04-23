package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	stderrors "errors"
	"fmt"

	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/bbu/rss-summarizer/backend/internal/domain/session"
	"github.com/google/uuid"
)

type SessionRepository interface {
	Create(ctx context.Context, input *session.CreateSessionInput) (*session.Session, error)
	FindByToken(ctx context.Context, token string) (*session.Session, error)
	DeleteByToken(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type sessionRepository struct {
	db *database.DB
}

func NewSessionRepository(db *database.DB) SessionRepository {
	return &sessionRepository{db: db}
}

// generateSecureToken creates a cryptographically secure random token
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (r *sessionRepository) Create(ctx context.Context, input *session.CreateSessionInput) (*session.Session, error) {
	token, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	s := &session.Session{
		ID:           uuid.New(),
		UserID:       input.UserID,
		SessionToken: token,
		ExpiresAt:    input.ExpiresAt,
	}

	query := `
		INSERT INTO sessions (id, user_id, session_token, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err = r.db.QueryRowContext(ctx, query,
		s.ID, s.UserID, s.SessionToken, s.ExpiresAt,
	).Scan(&s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return s, nil
}

func (r *sessionRepository) FindByToken(ctx context.Context, token string) (*session.Session, error) {
	var s session.Session
	query := `
		SELECT id, user_id, session_token, expires_at, created_at, updated_at
		FROM sessions
		WHERE session_token = $1 AND expires_at > NOW()
	`

	if err := r.db.GetContext(ctx, &s, query, token); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, &errors.NotFoundError{Resource: "session", ID: "token"}
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	return &s, nil
}

func (r *sessionRepository) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE session_token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func (r *sessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	count, _ := result.RowsAffected()
	return count, nil
}
