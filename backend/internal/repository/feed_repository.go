package repository

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"

	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/bbu/rss-summarizer/backend/internal/domain/feed"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type FeedRepository interface {
	Create(ctx context.Context, input *feed.CreateFeedInput) (*feed.Feed, error)
	FindByID(ctx context.Context, id uuid.UUID) (*feed.Feed, error)
	FindByURL(ctx context.Context, url string) (*feed.Feed, error)
	FindOrCreate(ctx context.Context, url string, pollFreq int) (*feed.Feed, error)
	Update(ctx context.Context, f *feed.Feed) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, lastError *string, errorCount int) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindActiveFeedsDueForPoll(ctx context.Context) ([]*feed.Feed, error)
}

type feedRepository struct {
	db *database.DB
}

func NewFeedRepository(db *database.DB) FeedRepository {
	return &feedRepository{db: db}
}

func (r *feedRepository) Create(ctx context.Context, input *feed.CreateFeedInput) (*feed.Feed, error) {
	f := &feed.Feed{
		ID:                   uuid.New(),
		URL:                  input.URL,
		PollFrequencyMinutes: input.PollFrequencyMinutes,
		IsActive:             true,
		Status:               feed.StatusHealthy,
	}

	query := `
		INSERT INTO feeds (id, url, title, description, poll_frequency_minutes, is_active, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		f.ID, f.URL, f.Title, f.Description, f.PollFrequencyMinutes, f.IsActive, f.Status,
	).Scan(&f.CreatedAt, &f.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return nil, &errors.DuplicateError{Resource: "feed", Field: "url", Value: input.URL}
		}
		return nil, fmt.Errorf("failed to create feed: %w", err)
	}

	return f, nil
}

func (r *feedRepository) FindByID(ctx context.Context, id uuid.UUID) (*feed.Feed, error) {
	var f feed.Feed
	query := `SELECT * FROM feeds WHERE id = $1`

	if err := r.db.GetContext(ctx, &f, query, id); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, &errors.NotFoundError{Resource: "feed", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to find feed: %w", err)
	}

	return &f, nil
}

func (r *feedRepository) FindByURL(ctx context.Context, url string) (*feed.Feed, error) {
	var f feed.Feed
	query := `SELECT * FROM feeds WHERE url = $1`

	if err := r.db.GetContext(ctx, &f, query, url); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, &errors.NotFoundError{Resource: "feed", ID: url}
		}
		return nil, fmt.Errorf("failed to find feed: %w", err)
	}

	return &f, nil
}

func (r *feedRepository) FindOrCreate(ctx context.Context, url string, pollFreq int) (*feed.Feed, error) {
	// Try to find existing feed first
	f, err := r.FindByURL(ctx, url)
	if err == nil {
		return f, nil
	}

	// If not found, create new feed
	if _, ok := err.(*errors.NotFoundError); ok {
		return r.Create(ctx, &feed.CreateFeedInput{
			URL:                  url,
			PollFrequencyMinutes: pollFreq,
		})
	}

	return nil, err
}

func (r *feedRepository) Update(ctx context.Context, f *feed.Feed) error {
	query := `
		UPDATE feeds
		SET title = $2, description = $3, poll_frequency_minutes = $4,
		    last_polled_at = $5, is_active = $6, status = $7, last_error = $8, error_count = $9, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		f.ID, f.Title, f.Description, f.PollFrequencyMinutes, f.LastPolledAt, f.IsActive, f.Status, f.LastError, f.ErrorCount,
	)
	if err != nil {
		return fmt.Errorf("failed to update feed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return &errors.NotFoundError{Resource: "feed", ID: f.ID.String()}
	}

	return nil
}

func (r *feedRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM feeds WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete feed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return &errors.NotFoundError{Resource: "feed", ID: id.String()}
	}

	return nil
}

func (r *feedRepository) FindActiveFeedsDueForPoll(ctx context.Context) ([]*feed.Feed, error) {
	var feeds []*feed.Feed
	query := `
		SELECT * FROM feeds
		WHERE is_active = true
		AND (
		    last_polled_at IS NULL
		    OR last_polled_at < NOW() - (poll_frequency_minutes || ' minutes')::INTERVAL
		)
		ORDER BY last_polled_at ASC NULLS FIRST
	`

	if err := r.db.SelectContext(ctx, &feeds, query); err != nil {
		return nil, fmt.Errorf("failed to find feeds due for poll: %w", err)
	}

	return feeds, nil
}

func (r *feedRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, lastError *string, errorCount int) error {
	query := `
		UPDATE feeds
		SET status = $2, last_error = $3, error_count = $4, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status, lastError, errorCount)
	if err != nil {
		return fmt.Errorf("failed to update feed status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return &errors.NotFoundError{Resource: "feed", ID: id.String()}
	}

	return nil
}

