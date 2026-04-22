package email_source

import (
	"time"

	"github.com/google/uuid"
)

// EmailSource represents a connected email account for fetching newsletters
type EmailSource struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	UserID         uuid.UUID  `db:"user_id" json:"user_id"`
	EmailAddress   string     `db:"email_address" json:"email_address"`
	Provider       string     `db:"provider" json:"provider"` // gmail, outlook, imap
	AccessToken    string     `db:"access_token" json:"-"`    // encrypted, never expose in JSON
	RefreshToken   string     `db:"refresh_token" json:"-"`   // encrypted, never expose in JSON
	TokenExpiresAt time.Time  `db:"token_expires_at" json:"token_expires_at"`
	LastFetchedAt  *time.Time `db:"last_fetched_at" json:"last_fetched_at,omitempty"`
	IsActive       bool       `db:"is_active" json:"is_active"`
	LastError      *string    `db:"last_error" json:"last_error,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}

// Provider constants
const (
	ProviderGmail   = "gmail"
	ProviderOutlook = "outlook"
	ProviderIMAP    = "imap"
)

// CreateEmailSourceInput contains the data needed to create a new email source
type CreateEmailSourceInput struct {
	UserID         uuid.UUID
	EmailAddress   string
	Provider       string
	AccessToken    string // Will be encrypted before storage
	RefreshToken   string // Will be encrypted before storage
	TokenExpiresAt time.Time
}

// UpdateEmailSourceInput contains the data that can be updated on an email source
type UpdateEmailSourceInput struct {
	AccessToken    *string    // Will be encrypted before storage
	RefreshToken   *string    // Will be encrypted before storage
	TokenExpiresAt *time.Time
	LastFetchedAt  *time.Time
	IsActive       *bool
	LastError      *string
}
