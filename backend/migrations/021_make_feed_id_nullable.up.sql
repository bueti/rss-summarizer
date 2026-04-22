BEGIN;

-- Drop the existing unique constraint that includes feed_id
ALTER TABLE articles DROP CONSTRAINT IF EXISTS articles_global_feed_id_url_key;

-- Make feed_id nullable for email-sourced articles
ALTER TABLE articles ALTER COLUMN feed_id DROP NOT NULL;

-- Add a new unique constraint: for RSS articles (feed_id not null), url must be unique per feed
-- This allows email articles (feed_id null) to have empty/null URLs
CREATE UNIQUE INDEX articles_feed_id_url_unique_idx
ON articles (feed_id, url)
WHERE feed_id IS NOT NULL AND url IS NOT NULL AND url != '';

-- Add check constraint: article must have either feed_id OR email_source_id
ALTER TABLE articles ADD CONSTRAINT articles_source_check
CHECK (
    (feed_id IS NOT NULL AND email_source_id IS NULL) OR
    (feed_id IS NULL AND email_source_id IS NOT NULL)
);

COMMENT ON COLUMN articles.feed_id IS 'RSS feed ID (null for email-sourced articles)';

COMMIT;
