-- User Preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    default_poll_interval INTEGER NOT NULL DEFAULT 30,
    llm_provider VARCHAR(50) NOT NULL DEFAULT 'anthropic',
    llm_model VARCHAR(100) NOT NULL DEFAULT 'claude-3-5-sonnet-20241022',
    max_articles_per_feed INTEGER NOT NULL DEFAULT 20,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);

-- Topics table
CREATE TABLE IF NOT EXISTS topics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    preference VARCHAR(20) NOT NULL DEFAULT 'normal' CHECK (preference IN ('high', 'normal', 'hide')),
    is_custom BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_topics_user_id ON topics(user_id);
CREATE INDEX idx_topics_preference ON topics(preference);

-- Add feed status tracking fields
ALTER TABLE feeds
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'healthy' CHECK (status IN ('healthy', 'warning', 'error', 'paused')),
    ADD COLUMN IF NOT EXISTS last_error TEXT,
    ADD COLUMN IF NOT EXISTS error_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_feeds_status ON feeds(status);

-- Add article processing status fields
ALTER TABLE articles
    ADD COLUMN IF NOT EXISTS processing_status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (processing_status IN ('pending', 'processing', 'completed', 'failed')),
    ADD COLUMN IF NOT EXISTS processing_error TEXT;

CREATE INDEX IF NOT EXISTS idx_articles_processing_status ON articles(processing_status);

-- Insert default preferences for existing dev user
INSERT INTO user_preferences (user_id, default_poll_interval, llm_provider, llm_model, max_articles_per_feed)
VALUES ('00000000-0000-0000-0000-000000000001', 30, 'anthropic', 'claude-3-5-sonnet-20241022', 20)
ON CONFLICT (user_id) DO NOTHING;
