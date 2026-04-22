-- Remove indexes
DROP INDEX IF EXISTS idx_user_articles_dashboard;
DROP INDEX IF EXISTS idx_user_articles_is_archived;
DROP INDEX IF EXISTS idx_user_articles_is_saved;

-- Remove columns
ALTER TABLE user_articles
DROP COLUMN IF EXISTS is_archived,
DROP COLUMN IF EXISTS is_saved;
