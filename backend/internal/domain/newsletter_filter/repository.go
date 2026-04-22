package newsletter_filter

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for newsletter filter data access
type Repository interface {
	// Create creates a new newsletter filter
	Create(ctx context.Context, input *CreateNewsletterFilterInput) (*NewsletterFilter, error)

	// FindByID retrieves a newsletter filter by ID
	FindByID(ctx context.Context, id uuid.UUID) (*NewsletterFilter, error)

	// FindByUserID retrieves all newsletter filters for a user
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*NewsletterFilter, error)

	// FindByEmailSourceID retrieves all newsletter filters for an email source
	FindByEmailSourceID(ctx context.Context, emailSourceID uuid.UUID) ([]*NewsletterFilter, error)

	// FindActiveByEmailSourceID retrieves all active newsletter filters for an email source
	FindActiveByEmailSourceID(ctx context.Context, emailSourceID uuid.UUID) ([]*NewsletterFilter, error)

	// Update updates a newsletter filter
	Update(ctx context.Context, id uuid.UUID, input *UpdateNewsletterFilterInput) (*NewsletterFilter, error)

	// Delete deletes a newsletter filter
	Delete(ctx context.Context, id uuid.UUID) error
}
