package newsletter_filter

import (
	"time"

	"github.com/google/uuid"
)

// NewsletterFilter represents a rule for identifying and filtering newsletter emails
type NewsletterFilter struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	UserID        uuid.UUID  `db:"user_id" json:"user_id"`
	EmailSourceID uuid.UUID  `db:"email_source_id" json:"email_source_id"`
	Name          string     `db:"name" json:"name"`
	SenderPattern string     `db:"sender_pattern" json:"sender_pattern"` // e.g., "*@substack.com" or "newsletter@example.com"
	SubjectPattern *string   `db:"subject_pattern" json:"subject_pattern,omitempty"` // optional regex
	LabelOrFolder *string    `db:"label_or_folder" json:"label_or_folder,omitempty"` // Gmail label or Outlook folder
	IsActive      bool       `db:"is_active" json:"is_active"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

// CreateNewsletterFilterInput contains the data needed to create a new newsletter filter
type CreateNewsletterFilterInput struct {
	UserID         uuid.UUID
	EmailSourceID  uuid.UUID
	Name           string
	SenderPattern  string
	SubjectPattern *string
	LabelOrFolder  *string
}

// UpdateNewsletterFilterInput contains the data that can be updated on a newsletter filter
type UpdateNewsletterFilterInput struct {
	Name           *string
	SenderPattern  *string
	SubjectPattern *string
	LabelOrFolder  *string
	IsActive       *bool
}
