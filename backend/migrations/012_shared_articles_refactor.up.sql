-- Migration 012: Refactor to shared articles architecture
-- This migration transforms per-user articles to shared articles across all users

-- =============================================================================
-- STEP 1: Create new global feeds table
-- =============================================================================

CREATE TABLE feeds_global (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    url TEXT UNIQUE NOT NULL,
    title VARCHAR(500),
    description TEXT,
    poll_frequency_minutes INTEGER NOT NULL DEFAULT 60,
    last_polled_at TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT true,
    status VARCHAR(20) NOT NULL DEFAULT 'healthy' CHECK (status IN ('healthy', 'warning', 'error', 'paused')),
    last_error TEXT,
    error_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feeds_global_url ON feeds_global(url);
CREATE INDEX idx_feeds_global_is_active ON feeds_global(is_active);
CREATE INDEX idx_feeds_global_last_polled_at ON feeds_global(last_polled_at) WHERE is_active = true;
CREATE INDEX idx_feeds_global_status ON feeds_global(status);

COMMENT ON TABLE feeds_global IS 'Global RSS feeds - one per unique URL, shared across all users';

-- =============================================================================
-- STEP 2: Create user_feed_subscriptions junction table
-- =============================================================================

CREATE TABLE user_feed_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feed_id UUID NOT NULL REFERENCES feeds_global(id) ON DELETE CASCADE,
    poll_frequency_override INTEGER,  -- NULL means use feed default
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, feed_id)
);

CREATE INDEX idx_user_feed_subscriptions_user_id ON user_feed_subscriptions(user_id);
CREATE INDEX idx_user_feed_subscriptions_feed_id ON user_feed_subscriptions(feed_id);
CREATE INDEX idx_user_feed_subscriptions_is_active ON user_feed_subscriptions(is_active);

COMMENT ON TABLE user_feed_subscriptions IS 'User subscriptions to global feeds';

-- =============================================================================
-- STEP 3: Migrate feeds from per-user to global
-- =============================================================================

-- Deduplicate by URL, taking the first feed for each URL with best data
INSERT INTO feeds_global (
    id,
    url,
    title,
    description,
    poll_frequency_minutes,
    last_polled_at,
    is_active,
    status,
    last_error,
    error_count,
    created_at,
    updated_at
)
SELECT DISTINCT ON (url)
    uuid_generate_v4() as id,  -- New ID for global feed
    url,
    COALESCE(NULLIF(title, ''), 'Untitled Feed') as title,  -- Handle empty titles
    description,
    poll_frequency_minutes,
    last_polled_at,
    is_active,
    status,
    last_error,
    error_count,
    MIN(created_at) OVER (PARTITION BY url) as created_at,  -- Use earliest creation time
    NOW() as updated_at
FROM feeds
WHERE url IS NOT NULL AND url != ''
ORDER BY
    url,
    -- Prefer completed feeds with content
    CASE WHEN status = 'healthy' THEN 0
         WHEN status = 'warning' THEN 1
         ELSE 2 END,
    -- Prefer feeds with titles
    CASE WHEN title IS NOT NULL AND title != '' THEN 0 ELSE 1 END,
    created_at ASC;

-- =============================================================================
-- STEP 4: Create user subscriptions from old feeds
-- =============================================================================

-- Map each user's feed to the corresponding global feed
INSERT INTO user_feed_subscriptions (
    user_id,
    feed_id,
    poll_frequency_override,
    is_active,
    created_at,
    updated_at
)
SELECT
    f.user_id,
    fg.id as feed_id,
    -- Only store override if different from global feed default
    CASE
        WHEN f.poll_frequency_minutes != fg.poll_frequency_minutes
        THEN f.poll_frequency_minutes
        ELSE NULL
    END as poll_frequency_override,
    f.is_active,
    f.created_at,
    NOW() as updated_at
FROM feeds f
INNER JOIN feeds_global fg ON f.url = fg.url
WHERE f.url IS NOT NULL AND f.url != ''
ON CONFLICT (user_id, feed_id) DO NOTHING;  -- Skip duplicates if same user subscribed twice

-- =============================================================================
-- STEP 5: Create new global articles table
-- =============================================================================

CREATE TABLE articles_global (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feed_id UUID NOT NULL REFERENCES feeds_global(id) ON DELETE CASCADE,
    title VARCHAR(1000) NOT NULL,
    url TEXT NOT NULL,
    published_at TIMESTAMP,
    original_content TEXT,
    full_text TEXT,
    summary TEXT,
    key_points TEXT[],
    importance_score INTEGER CHECK (importance_score >= 1 AND importance_score <= 5),
    topics TEXT[],
    processing_status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (processing_status IN ('pending', 'processing', 'completed', 'failed')),
    processing_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(feed_id, url)
);

CREATE INDEX idx_articles_global_feed_id ON articles_global(feed_id);
CREATE INDEX idx_articles_global_published_at ON articles_global(published_at DESC);
CREATE INDEX idx_articles_global_importance_score ON articles_global(importance_score DESC);
CREATE INDEX idx_articles_global_topics ON articles_global USING GIN(topics);
CREATE INDEX idx_articles_global_processing_status ON articles_global(processing_status);
CREATE INDEX idx_articles_global_url ON articles_global(url);

COMMENT ON TABLE articles_global IS 'Global articles - one per unique URL per feed, shared across all users';

-- =============================================================================
-- STEP 6: Migrate articles to global table
-- =============================================================================

-- Deduplicate by (feed_url, article_url) and choose best version
INSERT INTO articles_global (
    id,
    feed_id,
    title,
    url,
    published_at,
    original_content,
    full_text,
    summary,
    key_points,
    importance_score,
    topics,
    processing_status,
    processing_error,
    created_at,
    updated_at
)
SELECT DISTINCT ON (fg.id, a.url)
    uuid_generate_v4() as id,  -- New ID for global article
    fg.id as feed_id,  -- Map to global feed
    a.title,
    a.url,
    a.published_at,
    a.original_content,
    a.full_text,
    a.summary,
    a.key_points,
    a.importance_score,
    a.topics,
    a.processing_status,
    a.processing_error,
    MIN(a.created_at) OVER (PARTITION BY fg.id, a.url) as created_at,
    NOW() as updated_at
FROM articles a
INNER JOIN feeds f ON a.feed_id = f.id
INNER JOIN feeds_global fg ON f.url = fg.url
WHERE a.url IS NOT NULL AND a.url != ''
ORDER BY
    fg.id,
    a.url,
    -- Prefer completed articles with summaries
    CASE WHEN a.processing_status = 'completed' THEN 0
         WHEN a.processing_status = 'processing' THEN 1
         WHEN a.processing_status = 'pending' THEN 2
         ELSE 3 END,
    -- Prefer articles with longer summaries (better quality)
    LENGTH(COALESCE(a.summary, '')) DESC,
    -- Prefer newer articles
    a.created_at DESC;

-- =============================================================================
-- STEP 7: Create user_articles table for per-user state
-- =============================================================================

CREATE TABLE user_articles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_id UUID NOT NULL REFERENCES articles_global(id) ON DELETE CASCADE,
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, article_id)
);

CREATE INDEX idx_user_articles_user_id ON user_articles(user_id);
CREATE INDEX idx_user_articles_article_id ON user_articles(article_id);
CREATE INDEX idx_user_articles_is_read ON user_articles(is_read);
CREATE INDEX idx_user_articles_user_article ON user_articles(user_id, article_id);

COMMENT ON TABLE user_articles IS 'Per-user article state (read/unread, etc)';

-- =============================================================================
-- STEP 8: Migrate user article states
-- =============================================================================

-- Map each user's article state to the global article
INSERT INTO user_articles (
    user_id,
    article_id,
    is_read,
    created_at,
    updated_at
)
SELECT
    a.user_id,
    ag.id as article_id,
    a.is_read,
    a.created_at,
    NOW() as updated_at
FROM articles a
INNER JOIN feeds f ON a.feed_id = f.id
INNER JOIN feeds_global fg ON f.url = fg.url
INNER JOIN articles_global ag ON ag.feed_id = fg.id AND ag.url = a.url
WHERE a.url IS NOT NULL AND a.url != ''
ON CONFLICT (user_id, article_id) DO UPDATE
    -- If any user instance marked it read, keep it read
    SET is_read = EXCLUDED.is_read OR user_articles.is_read;

-- =============================================================================
-- STEP 9: Rename old tables (keep for rollback safety)
-- =============================================================================

ALTER TABLE feeds RENAME TO feeds_old;
ALTER TABLE articles RENAME TO articles_old;

-- Rename old indexes to avoid conflicts
ALTER INDEX IF EXISTS idx_feeds_url RENAME TO idx_feeds_old_url;
ALTER INDEX IF EXISTS idx_feeds_is_active RENAME TO idx_feeds_old_is_active;
ALTER INDEX IF EXISTS idx_feeds_last_polled_at RENAME TO idx_feeds_old_last_polled_at;
ALTER INDEX IF EXISTS idx_feeds_status RENAME TO idx_feeds_old_status;
ALTER INDEX IF EXISTS idx_feeds_user_id RENAME TO idx_feeds_old_user_id;

ALTER INDEX IF EXISTS idx_articles_feed_id RENAME TO idx_articles_old_feed_id;
ALTER INDEX IF EXISTS idx_articles_published_at RENAME TO idx_articles_old_published_at;
ALTER INDEX IF EXISTS idx_articles_importance_score RENAME TO idx_articles_old_importance_score;
ALTER INDEX IF EXISTS idx_articles_topics RENAME TO idx_articles_old_topics;
ALTER INDEX IF EXISTS idx_articles_processing_status RENAME TO idx_articles_old_processing_status;
ALTER INDEX IF EXISTS idx_articles_url RENAME TO idx_articles_old_url;
ALTER INDEX IF EXISTS idx_articles_user_id RENAME TO idx_articles_old_user_id;

-- =============================================================================
-- STEP 10: Rename new tables to official names
-- =============================================================================

ALTER TABLE feeds_global RENAME TO feeds;
ALTER TABLE articles_global RENAME TO articles;

-- Update index names to match new table names
ALTER INDEX idx_feeds_global_url RENAME TO idx_feeds_url;
ALTER INDEX idx_feeds_global_is_active RENAME TO idx_feeds_is_active;
ALTER INDEX idx_feeds_global_last_polled_at RENAME TO idx_feeds_last_polled_at;
ALTER INDEX idx_feeds_global_status RENAME TO idx_feeds_status;

ALTER INDEX idx_articles_global_feed_id RENAME TO idx_articles_feed_id;
ALTER INDEX idx_articles_global_published_at RENAME TO idx_articles_published_at;
ALTER INDEX idx_articles_global_importance_score RENAME TO idx_articles_importance_score;
ALTER INDEX idx_articles_global_topics RENAME TO idx_articles_topics;
ALTER INDEX idx_articles_global_processing_status RENAME TO idx_articles_processing_status;
ALTER INDEX idx_articles_global_url RENAME TO idx_articles_url;

-- =============================================================================
-- STEP 11: Update foreign key references in feeds table sequences
-- =============================================================================

-- Update sequence ownership (PostgreSQL housekeeping)
ALTER SEQUENCE IF EXISTS feeds_global_id_seq RENAME TO feeds_id_seq;
ALTER SEQUENCE IF EXISTS articles_global_id_seq RENAME TO articles_id_seq;

-- =============================================================================
-- Migration complete!
-- =============================================================================

-- Log migration stats
DO $$
DECLARE
    old_feed_count INT;
    new_feed_count INT;
    subscription_count INT;
    old_article_count INT;
    new_article_count INT;
    user_article_count INT;
BEGIN
    SELECT COUNT(*) INTO old_feed_count FROM feeds_old;
    SELECT COUNT(*) INTO new_feed_count FROM feeds;
    SELECT COUNT(*) INTO subscription_count FROM user_feed_subscriptions;
    SELECT COUNT(*) INTO old_article_count FROM articles_old;
    SELECT COUNT(*) INTO new_article_count FROM articles;
    SELECT COUNT(*) INTO user_article_count FROM user_articles;

    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Migration 012 Complete!';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Feeds: % per-user feeds → % global feeds', old_feed_count, new_feed_count;
    RAISE NOTICE 'Subscriptions: % user subscriptions created', subscription_count;
    RAISE NOTICE 'Articles: % per-user articles → % global articles', old_article_count, new_article_count;
    RAISE NOTICE 'User states: % user_article records created', user_article_count;
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Reduction: %.1f%% fewer feeds, %.1f%% fewer articles',
        (1 - new_feed_count::FLOAT / NULLIF(old_feed_count, 0)) * 100,
        (1 - new_article_count::FLOAT / NULLIF(old_article_count, 0)) * 100;
    RAISE NOTICE '=============================================================================';
END $$;
