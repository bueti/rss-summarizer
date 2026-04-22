package session

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID           uuid.UUID `db:"id" json:"id"`
	UserID       uuid.UUID `db:"user_id" json:"user_id"`
	SessionToken string    `db:"session_token" json:"-"` // Never expose in JSON
	ExpiresAt    time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type CreateSessionInput struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
}
