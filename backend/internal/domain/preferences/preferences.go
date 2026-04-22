package preferences

import (
	"time"

	"github.com/google/uuid"
)

// UserPreferences represents a user's preferences for feed polling
type UserPreferences struct {
	ID                  uuid.UUID `db:"id" json:"id"`
	UserID              uuid.UUID `db:"user_id" json:"user_id"`
	DefaultPollInterval int       `db:"default_poll_interval" json:"default_poll_interval"`
	MaxArticlesPerFeed  int       `db:"max_articles_per_feed" json:"max_articles_per_feed"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}
