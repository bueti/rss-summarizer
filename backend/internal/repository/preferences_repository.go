package repository

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/crypto"
	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/bbu/rss-summarizer/backend/internal/domain/preferences"
	"github.com/google/uuid"
)

type PreferencesRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*preferences.UserPreferences, error)
	Upsert(ctx context.Context, prefs *preferences.UserPreferences) error
}

type preferencesRepository struct {
	db            *database.DB
	cryptoService *crypto.Service
}

func NewPreferencesRepository(db *database.DB, cryptoService *crypto.Service) PreferencesRepository {
	return &preferencesRepository{
		db:            db,
		cryptoService: cryptoService,
	}
}

func (r *preferencesRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*preferences.UserPreferences, error) {
	var prefs preferences.UserPreferences
	query := `
		SELECT id, user_id, default_poll_interval, max_articles_per_feed, created_at, updated_at
		FROM user_preferences
		WHERE user_id = $1
	`

	if err := r.db.GetContext(ctx, &prefs, query, userID); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, &errors.NotFoundError{Resource: "user_preferences", ID: userID.String()}
		}
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}

	return &prefs, nil
}

func (r *preferencesRepository) Upsert(ctx context.Context, prefs *preferences.UserPreferences) error {
	now := time.Now()

	query := `
		INSERT INTO user_preferences (id, user_id, default_poll_interval, max_articles_per_feed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id)
		DO UPDATE SET
			default_poll_interval = EXCLUDED.default_poll_interval,
			max_articles_per_feed = EXCLUDED.max_articles_per_feed,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at, updated_at
	`

	if prefs.ID == uuid.Nil {
		prefs.ID = uuid.New()
	}
	if prefs.CreatedAt.IsZero() {
		prefs.CreatedAt = now
	}
	prefs.UpdatedAt = now

	var returned struct {
		ID        uuid.UUID `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	err := r.db.GetContext(ctx, &returned, query,
		prefs.ID,
		prefs.UserID,
		prefs.DefaultPollInterval,
		prefs.MaxArticlesPerFeed,
		prefs.CreatedAt,
		prefs.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert user preferences: %w", err)
	}

	prefs.ID = returned.ID
	prefs.CreatedAt = returned.CreatedAt
	prefs.UpdatedAt = returned.UpdatedAt

	return nil
}
