package repository

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"

	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/bbu/rss-summarizer/backend/internal/domain/subscription"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, input *subscription.CreateSubscriptionInput) (*subscription.UserFeedSubscription, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*subscription.UserFeedSubscription, error)
	FindByFeedID(ctx context.Context, feedID uuid.UUID) ([]*subscription.UserFeedSubscription, error)
	FindByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) (*subscription.UserFeedSubscription, error)
	ListSubscribedFeeds(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*subscription.SubscribedFeed, int, error)
	Update(ctx context.Context, sub *subscription.UserFeedSubscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) error
	GetSubscriberCount(ctx context.Context, feedID uuid.UUID) (int, error)
}

type subscriptionRepository struct {
	db *database.DB
}

func NewSubscriptionRepository(db *database.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) Create(ctx context.Context, input *subscription.CreateSubscriptionInput) (*subscription.UserFeedSubscription, error) {
	sub := &subscription.UserFeedSubscription{
		ID:                    uuid.New(),
		UserID:                input.UserID,
		FeedID:                input.FeedID,
		PollFrequencyOverride: input.PollFrequencyOverride,
		IsActive:              true,
	}

	query := `
		INSERT INTO user_feed_subscriptions (id, user_id, feed_id, poll_frequency_override, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		sub.ID, sub.UserID, sub.FeedID, sub.PollFrequencyOverride, sub.IsActive,
	).Scan(&sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return nil, &errors.DuplicateError{Resource: "subscription", Field: "user_id,feed_id", Value: fmt.Sprintf("%s,%s", input.UserID, input.FeedID)}
		}
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return sub, nil
}

func (r *subscriptionRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*subscription.UserFeedSubscription, error) {
	var subs []*subscription.UserFeedSubscription
	query := `SELECT * FROM user_feed_subscriptions WHERE user_id = $1 AND is_active = true ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &subs, query, userID); err != nil {
		return nil, fmt.Errorf("failed to find subscriptions: %w", err)
	}

	return subs, nil
}

func (r *subscriptionRepository) FindByFeedID(ctx context.Context, feedID uuid.UUID) ([]*subscription.UserFeedSubscription, error) {
	var subs []*subscription.UserFeedSubscription
	query := `SELECT * FROM user_feed_subscriptions WHERE feed_id = $1 AND is_active = true ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &subs, query, feedID); err != nil {
		return nil, fmt.Errorf("failed to find subscriptions: %w", err)
	}

	return subs, nil
}

// ListSubscribedFeeds returns a page of feeds the user is actively subscribed
// to, joined with each subscription's poll-frequency override, plus the total
// active-subscription count. Replaces the old N+1 FindByUserID + FindByID loop.
func (r *subscriptionRepository) ListSubscribedFeeds(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*subscription.SubscribedFeed, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM user_feed_subscriptions WHERE user_id = $1 AND is_active = true`
	if err := r.db.GetContext(ctx, &total, countQuery, userID); err != nil {
		return nil, 0, fmt.Errorf("failed to count subscribed feeds: %w", err)
	}

	rows := []*subscription.SubscribedFeed{}
	query := `
		SELECT
			f.id,
			f.url,
			f.title,
			f.description,
			COALESCE(s.poll_frequency_override, f.poll_frequency_minutes) AS effective_poll_frequency_minutes,
			f.last_polled_at,
			f.is_active,
			f.status,
			f.last_error,
			f.error_count,
			s.created_at,
			s.updated_at
		FROM user_feed_subscriptions s
		JOIN feeds f ON f.id = s.feed_id
		WHERE s.user_id = $1 AND s.is_active = true
		ORDER BY s.created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := r.db.SelectContext(ctx, &rows, query, userID, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("failed to list subscribed feeds: %w", err)
	}
	return rows, total, nil
}

func (r *subscriptionRepository) FindByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) (*subscription.UserFeedSubscription, error) {
	var sub subscription.UserFeedSubscription
	query := `SELECT * FROM user_feed_subscriptions WHERE user_id = $1 AND feed_id = $2`

	if err := r.db.GetContext(ctx, &sub, query, userID, feedID); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, &errors.NotFoundError{Resource: "subscription", ID: fmt.Sprintf("user=%s,feed=%s", userID, feedID)}
		}
		return nil, fmt.Errorf("failed to find subscription: %w", err)
	}

	return &sub, nil
}

func (r *subscriptionRepository) Update(ctx context.Context, sub *subscription.UserFeedSubscription) error {
	query := `
		UPDATE user_feed_subscriptions
		SET poll_frequency_override = $2, is_active = $3, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, sub.ID, sub.PollFrequencyOverride, sub.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &errors.NotFoundError{Resource: "subscription", ID: sub.ID.String()}
	}

	return nil
}

func (r *subscriptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_feed_subscriptions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &errors.NotFoundError{Resource: "subscription", ID: id.String()}
	}

	return nil
}

func (r *subscriptionRepository) DeleteByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) error {
	query := `DELETE FROM user_feed_subscriptions WHERE user_id = $1 AND feed_id = $2`

	result, err := r.db.ExecContext(ctx, query, userID, feedID)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &errors.NotFoundError{Resource: "subscription", ID: fmt.Sprintf("user=%s,feed=%s", userID, feedID)}
	}

	return nil
}

func (r *subscriptionRepository) GetSubscriberCount(ctx context.Context, feedID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM user_feed_subscriptions WHERE feed_id = $1 AND is_active = true`

	if err := r.db.GetContext(ctx, &count, query, feedID); err != nil {
		return 0, fmt.Errorf("failed to count subscribers: %w", err)
	}

	return count, nil
}
