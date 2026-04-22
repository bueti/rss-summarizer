package subscription

import (
	"time"

	"github.com/google/uuid"
)

// UserFeedSubscription represents a user's subscription to a global feed
type UserFeedSubscription struct {
	ID                    uuid.UUID  `db:"id" json:"id"`
	UserID                uuid.UUID  `db:"user_id" json:"user_id"`
	FeedID                uuid.UUID  `db:"feed_id" json:"feed_id"`
	PollFrequencyOverride *int       `db:"poll_frequency_override" json:"poll_frequency_override,omitempty"` // NULL means use feed default
	IsActive              bool       `db:"is_active" json:"is_active"`
	CreatedAt             time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at" json:"updated_at"`
}

// CreateSubscriptionInput contains the data needed to create a new subscription
type CreateSubscriptionInput struct {
	UserID                uuid.UUID
	FeedID                uuid.UUID
	PollFrequencyOverride *int
}
