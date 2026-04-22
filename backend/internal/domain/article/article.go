package article

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Article represents a global article shared across all users
type Article struct {
	ID               uuid.UUID      `db:"id" json:"id"`
	FeedID           *uuid.UUID     `db:"feed_id" json:"feed_id,omitempty"` // RSS feed ID (null for email-sourced articles)
	EmailSourceID    *uuid.UUID     `db:"email_source_id" json:"email_source_id,omitempty"` // Email source for email-sourced articles
	Title            string         `db:"title" json:"title"`
	URL              string         `db:"url" json:"url"`
	PublishedAt      *time.Time     `db:"published_at" json:"published_at,omitempty"`
	OriginalContent  string         `db:"original_content" json:"original_content,omitempty"`
	FullText         string         `db:"full_text" json:"full_text,omitempty"`
	Summary          string         `db:"summary" json:"summary,omitempty"`
	KeyPoints        pq.StringArray `db:"key_points" json:"key_points,omitempty"`
	ImportanceScore  *int           `db:"importance_score" json:"importance_score,omitempty"`
	Topics           pq.StringArray `db:"topics" json:"topics,omitempty"`
	ProcessingStatus string         `db:"processing_status" json:"processing_status"`
	ProcessingError  *string        `db:"processing_error" json:"processing_error,omitempty"`
	SourceType       string         `db:"source_type" json:"source_type"`           // "rss" or "email"
	EmailMessageID   *string        `db:"email_message_id" json:"email_message_id,omitempty"` // Gmail message ID for email-sourced articles
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at" json:"updated_at"`
}

// ArticleWithUserState combines a global article with per-user state
type ArticleWithUserState struct {
	Article
	IsRead     bool `db:"is_read" json:"is_read"`
	IsSaved    bool `db:"is_saved" json:"is_saved"`
	IsArchived bool `db:"is_archived" json:"is_archived"`
}

// Article processing status constants
const (
	ProcessingPending    = "pending"
	ProcessingProcessing = "processing"
	ProcessingCompleted  = "completed"
	ProcessingFailed     = "failed"
)

type ArticleFilters struct {
	FeedID           *uuid.UUID
	EmailSourceID    *uuid.UUID
	MinImportance    *int
	Topic            *string
	IsRead           *bool
	IsSaved          *bool
	IsArchived       *bool
	ProcessingStatus *string
	SortBy           *string // "date", "importance"
	Limit            int
	Offset           int
}
