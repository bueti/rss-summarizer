package repository

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"

	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/errors"
	"github.com/bbu/rss-summarizer/backend/internal/domain/user"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*user.User, error)
	Update(ctx context.Context, u *user.User) error
	ListAll(ctx context.Context) ([]*user.User, error)
	UpdateRole(ctx context.Context, userID uuid.UUID, role string) error
}

type userRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (id, email, name, google_id, picture_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}

	return r.db.QueryRowContext(ctx, query,
		u.ID, u.Email, u.Name, u.GoogleID, u.PictureURL,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var u user.User
	query := `SELECT * FROM users WHERE id = $1`

	if err := r.db.GetContext(ctx, &u, query, id); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, &errors.NotFoundError{Resource: "user", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &u, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	query := `SELECT * FROM users WHERE email = $1`

	if err := r.db.GetContext(ctx, &u, query, email); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, &errors.NotFoundError{Resource: "user", ID: email}
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &u, nil
}

func (r *userRepository) FindByGoogleID(ctx context.Context, googleID string) (*user.User, error) {
	var u user.User
	query := `SELECT * FROM users WHERE google_id = $1`

	if err := r.db.GetContext(ctx, &u, query, googleID); err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, &errors.NotFoundError{Resource: "user", ID: googleID}
		}
		return nil, fmt.Errorf("failed to find user by google_id: %w", err)
	}

	return &u, nil
}

func (r *userRepository) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users
		SET email = $2, name = $3, google_id = $4, picture_url = $5, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		u.ID, u.Email, u.Name, u.GoogleID, u.PictureURL,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &errors.NotFoundError{Resource: "user", ID: u.ID.String()}
	}

	return nil
}

func (r *userRepository) ListAll(ctx context.Context) ([]*user.User, error) {
	var users []*user.User
	query := `SELECT * FROM users ORDER BY created_at ASC`

	if err := r.db.SelectContext(ctx, &users, query); err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

func (r *userRepository) UpdateRole(ctx context.Context, userID uuid.UUID, role string) error {
	query := `
		UPDATE users
		SET role = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, userID, role)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &errors.NotFoundError{Resource: "user", ID: userID.String()}
	}

	return nil
}
