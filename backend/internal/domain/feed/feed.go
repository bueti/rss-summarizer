package feed

import (
	"time"

	"github.com/google/uuid"
)

// Feed represents a global RSS feed shared across all users
type Feed struct {
	ID                   uuid.UUID  `db:"id" json:"id"`
	URL                  string     `db:"url" json:"url"`
	Title                string     `db:"title" json:"title"`
	Description          string     `db:"description" json:"description"`
	PollFrequencyMinutes int        `db:"poll_frequency_minutes" json:"poll_frequency_minutes"`
	LastPolledAt         *time.Time `db:"last_polled_at" json:"last_polled_at,omitempty"`
	IsActive             bool       `db:"is_active" json:"is_active"`
	Status               string     `db:"status" json:"status"`
	LastError            *string    `db:"last_error" json:"last_error,omitempty"`
	ErrorCount           int        `db:"error_count" json:"error_count"`
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at" json:"updated_at"`
}

// Feed status constants
const (
	StatusHealthy = "healthy"
	StatusWarning = "warning"
	StatusError   = "error"
	StatusPaused  = "paused"
)

// CreateFeedInput contains the data needed to create a new global feed
type CreateFeedInput struct {
	URL                  string
	PollFrequencyMinutes int
}
