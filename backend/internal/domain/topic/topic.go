package topic

import (
	"time"

	"github.com/google/uuid"
)

// Topic represents a global topic that can be associated with articles
type Topic struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	IsCustom  bool      `db:"is_custom" json:"is_custom"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// UserTopicPreference represents a user's preference for a specific topic
type UserTopicPreference struct {
	ID         uuid.UUID `db:"id" json:"id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	TopicID    uuid.UUID `db:"topic_id" json:"topic_id"`
	Preference string    `db:"preference" json:"preference"` // "high", "normal", "hide"
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// TopicWithPreference combines topic information with a user's preference
type TopicWithPreference struct {
	ID         uuid.UUID `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	IsCustom   bool      `db:"is_custom" json:"is_custom"`
	Preference string    `db:"preference" json:"preference"` // User's preference for this topic
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// ValidPreferences lists all valid topic preference values
var ValidPreferences = []string{"high", "normal", "hide"}

// IsValidPreference checks if a preference value is valid
func IsValidPreference(pref string) bool {
	for _, valid := range ValidPreferences {
		if pref == valid {
			return true
		}
	}
	return false
}
