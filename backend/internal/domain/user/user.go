package user

import (
	"time"

	"github.com/google/uuid"
)

// Role constants
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID         uuid.UUID `db:"id" json:"id"`
	Email      string    `db:"email" json:"email"`
	Name       string    `db:"name" json:"name"`
	GoogleID   *string   `db:"google_id" json:"google_id,omitempty"`
	PictureURL *string   `db:"picture_url" json:"picture_url,omitempty"`
	Role       string    `db:"role" json:"role"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
