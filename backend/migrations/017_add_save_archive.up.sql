-- Add saved and archived columns to user_articles table
ALTER TABLE user_articles
ADD COLUMN is_saved BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN is_archived BOOLEAN NOT NULL DEFAULT false;

-- Add indexes for query performance
-- Partial index for saved articles view (only index saved articles)
CREATE INDEX idx_user_articles_is_saved
ON user_articles(user_id, is_saved)
WHERE is_saved = true;

-- Partial index for archived articles view (only index archived articles)
CREATE INDEX idx_user_articles_is_archived
ON user_articles(user_id, is_archived)
WHERE is_archived = true;

-- Composite index for dashboard query (unread + not archived)
CREATE INDEX idx_user_articles_dashboard
ON user_articles(user_id, is_read, is_archived)
WHERE is_read = false AND is_archived = false;

-- Add comments for documentation
COMMENT ON COLUMN user_articles.is_saved IS 'User saved article for later reference';
COMMENT ON COLUMN user_articles.is_archived IS 'Article archived (automatically when read and not saved, or manually)';
