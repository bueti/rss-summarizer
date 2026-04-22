package user_article

import (
	"time"

	"github.com/google/uuid"
)

// UserArticle represents per-user state for a global article
type UserArticle struct {
	ID         uuid.UUID `db:"id" json:"id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	ArticleID  uuid.UUID `db:"article_id" json:"article_id"`
	IsRead     bool      `db:"is_read" json:"is_read"`
	IsSaved    bool      `db:"is_saved" json:"is_saved"`
	IsArchived bool      `db:"is_archived" json:"is_archived"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}
