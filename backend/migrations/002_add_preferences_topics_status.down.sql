-- Remove article processing status fields
DROP INDEX IF EXISTS idx_articles_processing_status;
ALTER TABLE articles
    DROP COLUMN IF EXISTS processing_error,
    DROP COLUMN IF EXISTS processing_status;

-- Remove feed status tracking fields
DROP INDEX IF EXISTS idx_feeds_status;
ALTER TABLE feeds
    DROP COLUMN IF EXISTS error_count,
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS status;

-- Drop topics table
DROP INDEX IF EXISTS idx_topics_preference;
DROP INDEX IF EXISTS idx_topics_user_id;
DROP TABLE IF EXISTS topics;

-- Drop user_preferences table
DROP INDEX IF EXISTS idx_user_preferences_user_id;
DROP TABLE IF EXISTS user_preferences;
