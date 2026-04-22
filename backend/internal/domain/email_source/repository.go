package email_source

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for email source data access
type Repository interface {
	// Create creates a new email source
	Create(ctx context.Context, input *CreateEmailSourceInput) (*EmailSource, error)

	// FindByID retrieves an email source by ID
	FindByID(ctx context.Context, id uuid.UUID) (*EmailSource, error)

	// FindByUserID retrieves all email sources for a user
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*EmailSource, error)

	// FindActiveByUserID retrieves all active email sources for a user
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*EmailSource, error)

	// FindAllActive retrieves all active email sources across all users
	FindAllActive(ctx context.Context) ([]*EmailSource, error)

	// Update updates an email source
	Update(ctx context.Context, id uuid.UUID, input *UpdateEmailSourceInput) (*EmailSource, error)

	// Delete deletes an email source
	Delete(ctx context.Context, id uuid.UUID) error
}
