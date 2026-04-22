-- Migration 013: Move LLM config from per-user to global admin settings
-- Remove LLM settings from user_preferences, create global llm_config table

-- =============================================================================
-- STEP 1: Create global llm_config table
-- =============================================================================

CREATE TABLE llm_config (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider VARCHAR(50) NOT NULL DEFAULT 'anthropic',
    model VARCHAR(100) NOT NULL DEFAULT 'claude-3-5-sonnet-20241022',
    api_url TEXT NOT NULL DEFAULT 'https://api.anthropic.com/v1',
    api_key TEXT, -- Encrypted
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Only one config row allowed (enforced by unique constraint on a constant)
ALTER TABLE llm_config ADD COLUMN singleton_guard INTEGER DEFAULT 1;
CREATE UNIQUE INDEX llm_config_singleton ON llm_config(singleton_guard);

-- =============================================================================
-- STEP 2: Migrate existing LLM config to global table
-- =============================================================================

-- Insert global config from the first user's preferences (or use defaults)
INSERT INTO llm_config (provider, model, api_url, api_key)
SELECT
    COALESCE(llm_provider, 'anthropic'),
    COALESCE(llm_model, 'claude-3-5-sonnet-20241022'),
    COALESCE(NULLIF(llm_api_url, ''), 'https://api.anthropic.com/v1'),
    llm_api_key
FROM user_preferences
WHERE llm_api_key IS NOT NULL
LIMIT 1;

-- If no user had API key set, insert defaults
INSERT INTO llm_config (provider, model, api_url, api_key)
SELECT 'anthropic', 'claude-3-5-sonnet-20241022', 'https://api.anthropic.com/v1', NULL
WHERE NOT EXISTS (SELECT 1 FROM llm_config);

-- =============================================================================
-- STEP 3: Remove LLM columns from user_preferences
-- =============================================================================

ALTER TABLE user_preferences DROP COLUMN IF EXISTS llm_provider;
ALTER TABLE user_preferences DROP COLUMN IF EXISTS llm_model;
ALTER TABLE user_preferences DROP COLUMN IF EXISTS llm_api_url;
ALTER TABLE user_preferences DROP COLUMN IF EXISTS llm_api_key;

-- =============================================================================
-- Migration complete
-- =============================================================================

DO $$
DECLARE
    config_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO config_count FROM llm_config;

    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Migration 013 Complete!';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Created global LLM config table with % config(s)', config_count;
    RAISE NOTICE 'Removed per-user LLM settings from user_preferences';
    RAISE NOTICE '=============================================================================';
END $$;
