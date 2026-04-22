package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/topic"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type TopicRepository interface {
	// Find all topics that appear in a user's articles, with their preferences
	FindTopicsWithPreferences(ctx context.Context, userID uuid.UUID) ([]*topic.TopicWithPreference, error)

	// Get a global topic by ID
	GetTopicByID(ctx context.Context, id uuid.UUID) (*topic.Topic, error)

	// Get a global topic by name
	GetTopicByName(ctx context.Context, name string) (*topic.Topic, error)

	// Create or update a user's preference for a topic
	UpsertPreference(ctx context.Context, userID uuid.UUID, topicID uuid.UUID, preference string) error

	// Get a user's preference for a topic
	GetPreference(ctx context.Context, userID uuid.UUID, topicID uuid.UUID) (*topic.UserTopicPreference, error)

	// Delete a user's preference (reverts to 'normal')
	DeletePreference(ctx context.Context, userID uuid.UUID, topicID uuid.UUID) error

	// Ensure topics exist for the given names (create if they don't exist)
	// Used when processing articles to ensure all detected topics exist
	EnsureTopicsExist(ctx context.Context, topicNames []string) error

	// Create a custom topic
	CreateCustomTopic(ctx context.Context, name string) (*topic.Topic, error)

	// Delete a custom topic (only if it has no preferences)
	DeleteCustomTopic(ctx context.Context, topicID uuid.UUID) error
}

type topicRepository struct {
	db *database.DB
}

func NewTopicRepository(db *database.DB) TopicRepository {
	return &topicRepository{db: db}
}

func (r *topicRepository) FindTopicsWithPreferences(ctx context.Context, userID uuid.UUID) ([]*topic.TopicWithPreference, error) {
	var results []*topic.TopicWithPreference

	// Get all topics from user's articles and custom topics they've created
	query := `
		WITH user_article_topics AS (
			-- Get all unique topics from articles in feeds the user subscribes to
			SELECT DISTINCT UNNEST(a.topics) as topic_name
			FROM articles a
			INNER JOIN user_feed_subscriptions s ON a.feed_id = s.feed_id
			WHERE s.user_id = $1 AND a.topics IS NOT NULL
		),
		relevant_topics AS (
			-- Topics from user's articles
			SELECT t.id
			FROM topics t
			INNER JOIN user_article_topics uat ON LOWER(t.name) = LOWER(uat.topic_name)

			UNION

			-- Custom topics the user has created or has preferences for
			SELECT t.id
			FROM topics t
			INNER JOIN user_topic_preferences utp ON utp.topic_id = t.id
			WHERE utp.user_id = $1
		)
		SELECT DISTINCT
			t.id,
			t.name,
			t.is_custom,
			COALESCE(utp.preference, 'normal') as preference,
			COALESCE(utp.created_at, t.created_at) as created_at,
			COALESCE(utp.updated_at, t.updated_at) as updated_at
		FROM topics t
		INNER JOIN relevant_topics rt ON rt.id = t.id
		LEFT JOIN user_topic_preferences utp ON utp.topic_id = t.id AND utp.user_id = $1
		ORDER BY t.name ASC
	`

	if err := r.db.SelectContext(ctx, &results, query, userID); err != nil {
		return nil, fmt.Errorf("failed to find topics with preferences: %w", err)
	}

	return results, nil
}

func (r *topicRepository) GetTopicByID(ctx context.Context, id uuid.UUID) (*topic.Topic, error) {
	var t topic.Topic
	query := `SELECT id, name, is_custom, created_at, updated_at FROM topics WHERE id = $1`

	if err := r.db.GetContext(ctx, &t, query, id); err != nil {
		return nil, fmt.Errorf("failed to get topic: %w", err)
	}

	return &t, nil
}

func (r *topicRepository) GetTopicByName(ctx context.Context, name string) (*topic.Topic, error) {
	var t topic.Topic
	query := `SELECT id, name, is_custom, created_at, updated_at FROM topics WHERE LOWER(name) = LOWER($1)`

	if err := r.db.GetContext(ctx, &t, query, name); err != nil {
		return nil, fmt.Errorf("failed to get topic by name: %w", err)
	}

	return &t, nil
}

func (r *topicRepository) UpsertPreference(ctx context.Context, userID uuid.UUID, topicID uuid.UUID, preference string) error {
	now := time.Now()
	query := `
		INSERT INTO user_topic_preferences (id, user_id, topic_id, preference, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, topic_id)
		DO UPDATE SET preference = $4, updated_at = $6
	`

	_, err := r.db.ExecContext(ctx, query, uuid.New(), userID, topicID, preference, now, now)
	if err != nil {
		return fmt.Errorf("failed to upsert preference: %w", err)
	}

	return nil
}

func (r *topicRepository) GetPreference(ctx context.Context, userID uuid.UUID, topicID uuid.UUID) (*topic.UserTopicPreference, error) {
	var pref topic.UserTopicPreference
	query := `
		SELECT id, user_id, topic_id, preference, created_at, updated_at
		FROM user_topic_preferences
		WHERE user_id = $1 AND topic_id = $2
	`

	if err := r.db.GetContext(ctx, &pref, query, userID, topicID); err != nil {
		return nil, fmt.Errorf("failed to get preference: %w", err)
	}

	return &pref, nil
}

func (r *topicRepository) DeletePreference(ctx context.Context, userID uuid.UUID, topicID uuid.UUID) error {
	query := `DELETE FROM user_topic_preferences WHERE user_id = $1 AND topic_id = $2`

	result, err := r.db.ExecContext(ctx, query, userID, topicID)
	if err != nil {
		return fmt.Errorf("failed to delete preference: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *topicRepository) EnsureTopicsExist(ctx context.Context, topicNames []string) error {
	if len(topicNames) == 0 {
		return nil
	}

	now := time.Now()

	// Use the normalize_topics function to normalize topic names
	// The unique index is on LOWER(name), so we use a subquery to handle conflicts
	query := `
		INSERT INTO topics (id, name, is_custom, created_at, updated_at)
		SELECT uuid_generate_v4(), topic, false, $1, $2
		FROM UNNEST(normalize_topics($3::TEXT[])) AS topic
		WHERE NOT EXISTS (
			SELECT 1 FROM topics WHERE LOWER(name) = LOWER(topic)
		)
	`

	_, err := r.db.ExecContext(ctx, query, now, now, pq.Array(topicNames))
	if err != nil {
		return fmt.Errorf("failed to ensure topics exist: %w", err)
	}

	return nil
}

func (r *topicRepository) CreateCustomTopic(ctx context.Context, name string) (*topic.Topic, error) {
	now := time.Now()
	t := &topic.Topic{
		ID:        uuid.New(),
		Name:      name,
		IsCustom:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	query := `
		INSERT INTO topics (id, name, is_custom, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query, t.ID, t.Name, t.IsCustom, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create custom topic: %w", err)
	}

	return t, nil
}

func (r *topicRepository) DeleteCustomTopic(ctx context.Context, topicID uuid.UUID) error {
	// Only delete if it's custom and has no user preferences
	query := `
		DELETE FROM topics
		WHERE id = $1
		AND is_custom = true
		AND NOT EXISTS (
			SELECT 1 FROM user_topic_preferences WHERE topic_id = $1
		)
	`

	result, err := r.db.ExecContext(ctx, query, topicID)
	if err != nil {
		return fmt.Errorf("failed to delete custom topic: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("topic not found, not custom, or has user preferences")
	}

	return nil
}
