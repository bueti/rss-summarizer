package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/bbu/rss-summarizer/backend/internal/domain/newsletter_filter"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type NewsletterFilterRepository struct {
	db *database.DB
}

func NewNewsletterFilterRepository(db *database.DB) newsletter_filter.Repository {
	return &NewsletterFilterRepository{db: db}
}

func (r *NewsletterFilterRepository) Create(ctx context.Context, input *newsletter_filter.CreateNewsletterFilterInput) (*newsletter_filter.NewsletterFilter, error) {
	filter := &newsletter_filter.NewsletterFilter{
		ID:             uuid.New(),
		UserID:         input.UserID,
		EmailSourceID:  input.EmailSourceID,
		Name:           input.Name,
		SenderPattern:  input.SenderPattern,
		SubjectPattern: input.SubjectPattern,
		LabelOrFolder:  input.LabelOrFolder,
		IsActive:       true,
	}

	query := `
		INSERT INTO newsletter_filters (
			id, user_id, email_source_id, name, sender_pattern, subject_pattern,
			label_or_folder, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		filter.ID, filter.UserID, filter.EmailSourceID, filter.Name,
		filter.SenderPattern, filter.SubjectPattern, filter.LabelOrFolder, filter.IsActive,
	).Scan(&filter.CreatedAt, &filter.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return nil, &errors.DuplicateError{
				Resource: "newsletter_filter",
				Field:    "name",
				Value:    input.Name,
			}
		}
		return nil, fmt.Errorf("failed to create newsletter filter: %w", err)
	}

	return filter, nil
}

func (r *NewsletterFilterRepository) FindByID(ctx context.Context, id uuid.UUID) (*newsletter_filter.NewsletterFilter, error) {
	var filter newsletter_filter.NewsletterFilter
	query := `SELECT * FROM newsletter_filters WHERE id = $1`

	if err := r.db.GetContext(ctx, &filter, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, &errors.NotFoundError{Resource: "newsletter_filter", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to find newsletter filter: %w", err)
	}

	return &filter, nil
}

func (r *NewsletterFilterRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*newsletter_filter.NewsletterFilter, error) {
	var filters []*newsletter_filter.NewsletterFilter
	query := `SELECT * FROM newsletter_filters WHERE user_id = $1 ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &filters, query, userID); err != nil {
		return nil, fmt.Errorf("failed to find newsletter filters by user ID: %w", err)
	}

	return filters, nil
}

func (r *NewsletterFilterRepository) FindByEmailSourceID(ctx context.Context, emailSourceID uuid.UUID) ([]*newsletter_filter.NewsletterFilter, error) {
	var filters []*newsletter_filter.NewsletterFilter
	query := `SELECT * FROM newsletter_filters WHERE email_source_id = $1 ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &filters, query, emailSourceID); err != nil {
		return nil, fmt.Errorf("failed to find newsletter filters by email source ID: %w", err)
	}

	return filters, nil
}

func (r *NewsletterFilterRepository) FindActiveByEmailSourceID(ctx context.Context, emailSourceID uuid.UUID) ([]*newsletter_filter.NewsletterFilter, error) {
	var filters []*newsletter_filter.NewsletterFilter
	query := `SELECT * FROM newsletter_filters WHERE email_source_id = $1 AND is_active = true ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &filters, query, emailSourceID); err != nil {
		return nil, fmt.Errorf("failed to find active newsletter filters by email source ID: %w", err)
	}

	return filters, nil
}

func (r *NewsletterFilterRepository) Update(ctx context.Context, id uuid.UUID, input *newsletter_filter.UpdateNewsletterFilterInput) (*newsletter_filter.NewsletterFilter, error) {
	// Start building update query dynamically
	query := `UPDATE newsletter_filters SET updated_at = NOW()`
	args := []any{}
	argPos := 1

	if input.Name != nil {
		query += fmt.Sprintf(`, name = $%d`, argPos)
		args = append(args, *input.Name)
		argPos++
	}

	if input.SenderPattern != nil {
		query += fmt.Sprintf(`, sender_pattern = $%d`, argPos)
		args = append(args, *input.SenderPattern)
		argPos++
	}

	if input.SubjectPattern != nil {
		query += fmt.Sprintf(`, subject_pattern = $%d`, argPos)
		args = append(args, *input.SubjectPattern)
		argPos++
	}

	if input.LabelOrFolder != nil {
		query += fmt.Sprintf(`, label_or_folder = $%d`, argPos)
		args = append(args, *input.LabelOrFolder)
		argPos++
	}

	if input.IsActive != nil {
		query += fmt.Sprintf(`, is_active = $%d`, argPos)
		args = append(args, *input.IsActive)
		argPos++
	}

	query += fmt.Sprintf(` WHERE id = $%d`, argPos)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return nil, &errors.DuplicateError{
				Resource: "newsletter_filter",
				Field:    "name",
				Value:    *input.Name,
			}
		}
		return nil, fmt.Errorf("failed to update newsletter filter: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return nil, &errors.NotFoundError{Resource: "newsletter_filter", ID: id.String()}
	}

	// Return updated filter
	return r.FindByID(ctx, id)
}

func (r *NewsletterFilterRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM newsletter_filters WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete newsletter filter: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return &errors.NotFoundError{Resource: "newsletter_filter", ID: id.String()}
	}

	return nil
}
