-- Rollback email newsletter support

-- Remove indexes
DROP INDEX IF EXISTS idx_articles_email_message_id;
DROP INDEX IF EXISTS idx_articles_source_type;
DROP INDEX IF EXISTS idx_newsletter_filters_is_active;
DROP INDEX IF EXISTS idx_newsletter_filters_email_source;
DROP INDEX IF EXISTS idx_newsletter_filters_user_id;
DROP INDEX IF EXISTS idx_email_sources_is_active;
DROP INDEX IF EXISTS idx_email_sources_user_id;

-- Remove columns from articles table
ALTER TABLE articles
DROP COLUMN IF EXISTS email_message_id,
DROP COLUMN IF EXISTS source_type;

-- Drop tables (cascade to delete related records)
DROP TABLE IF EXISTS newsletter_filters;
DROP TABLE IF EXISTS email_sources;
