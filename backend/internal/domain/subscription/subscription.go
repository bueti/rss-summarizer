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

// SubscribedFeed is a joined view of a feed plus the caller's per-user
// subscription settings. EffectivePollFrequencyMinutes is the feed default
// unless the subscription overrides it.
type SubscribedFeed struct {
	ID                           uuid.UUID  `db:"id"`
	URL                          string     `db:"url"`
	Title                        string     `db:"title"`
	Description                  string     `db:"description"`
	EffectivePollFrequencyMinutes int       `db:"effective_poll_frequency_minutes"`
	LastPolledAt                 *time.Time `db:"last_polled_at"`
	IsActive                     bool       `db:"is_active"`
	Status                       string     `db:"status"`
	LastError                    *string    `db:"last_error"`
	ErrorCount                   int        `db:"error_count"`
	CreatedAt                    time.Time  `db:"created_at"`
	UpdatedAt                    time.Time  `db:"updated_at"`
}
