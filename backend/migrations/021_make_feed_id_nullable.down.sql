BEGIN;

-- Remove check constraint
ALTER TABLE articles DROP CONSTRAINT IF EXISTS articles_source_check;

-- Drop the new unique index
DROP INDEX IF EXISTS articles_feed_id_url_unique_idx;

-- Delete email articles (they won't be compatible with NOT NULL feed_id)
DELETE FROM articles WHERE feed_id IS NULL;

-- Make feed_id NOT NULL again
ALTER TABLE articles ALTER COLUMN feed_id SET NOT NULL;

-- Restore original unique constraint
ALTER TABLE articles ADD CONSTRAINT articles_global_feed_id_url_key UNIQUE (feed_id, url);

COMMIT;
