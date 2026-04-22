-- Migration 016 Down: Revert topics to per-user model

-- Step 1: Recreate old topics table
CREATE TABLE IF NOT EXISTS topics_old (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    preference VARCHAR(20) NOT NULL DEFAULT 'normal' CHECK (preference IN ('high', 'normal', 'hide')),
    is_custom BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_topics_old_user_id ON topics_old(user_id);
CREATE INDEX idx_topics_old_preference ON topics_old(preference);

-- Step 2: Migrate data back
INSERT INTO topics_old (user_id, name, preference, is_custom, created_at, updated_at)
SELECT
    utp.user_id,
    t.name,
    utp.preference,
    t.is_custom,
    utp.created_at,
    utp.updated_at
FROM user_topic_preferences utp
INNER JOIN topics t ON utp.topic_id = t.id;

-- Step 3: Drop new tables and restore old
DROP TABLE user_topic_preferences CASCADE;
DROP TABLE topics CASCADE;
ALTER TABLE topics_old RENAME TO topics;
ALTER INDEX idx_topics_old_user_id RENAME TO idx_topics_user_id;
ALTER INDEX idx_topics_old_preference RENAME TO idx_topics_preference;
