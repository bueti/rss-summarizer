-- Migration 016: Refactor topics to be shared across users with separate preferences

-- Step 1: Create new global topics table (without user_id)
CREATE TABLE IF NOT EXISTS topics_new (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    is_custom BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_topics_new_name_lower ON topics_new(LOWER(name));
CREATE INDEX idx_topics_new_name ON topics_new(name);

-- Step 2: Create user_topic_preferences junction table
CREATE TABLE IF NOT EXISTS user_topic_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics_new(id) ON DELETE CASCADE,
    preference VARCHAR(20) NOT NULL DEFAULT 'normal' CHECK (preference IN ('high', 'normal', 'hide')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, topic_id)
);

CREATE INDEX idx_user_topic_preferences_user_id ON user_topic_preferences(user_id);
CREATE INDEX idx_user_topic_preferences_topic_id ON user_topic_preferences(topic_id);
CREATE INDEX idx_user_topic_preferences_preference ON user_topic_preferences(preference);

-- Step 3: Migrate existing data
-- First, insert unique topic names into topics_new
INSERT INTO topics_new (name, is_custom, created_at)
SELECT DISTINCT ON (LOWER(name))
    name,
    bool_or(is_custom) as is_custom,  -- If any user has it as custom, mark as custom
    MIN(created_at) as created_at
FROM topics
GROUP BY LOWER(name), name
ORDER BY LOWER(name), created_at;

-- Then, migrate user preferences
INSERT INTO user_topic_preferences (user_id, topic_id, preference, created_at)
SELECT
    t_old.user_id,
    t_new.id as topic_id,
    t_old.preference,
    t_old.created_at
FROM topics t_old
INNER JOIN topics_new t_new ON LOWER(t_old.name) = LOWER(t_new.name);

-- Step 4: Drop old topics table and rename new one
DROP TABLE topics CASCADE;
ALTER TABLE topics_new RENAME TO topics;
ALTER INDEX idx_topics_new_name RENAME TO idx_topics_name;

-- Log migration results
DO $$
DECLARE
    topic_count INTEGER;
    pref_count INTEGER;
    user_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO topic_count FROM topics;
    SELECT COUNT(*) INTO pref_count FROM user_topic_preferences;
    SELECT COUNT(DISTINCT user_id) INTO user_count FROM user_topic_preferences;

    RAISE NOTICE '============================================================================';
    RAISE NOTICE 'Migration 016 Complete!';
    RAISE NOTICE '============================================================================';
    RAISE NOTICE 'Created % global topics', topic_count;
    RAISE NOTICE 'Migrated % user topic preferences', pref_count;
    RAISE NOTICE 'For % users', user_count;
    RAISE NOTICE '============================================================================';
END $$;
