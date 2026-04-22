-- Add llm_api_key column to user_preferences table
ALTER TABLE user_preferences
ADD COLUMN llm_api_key TEXT;

-- Add comment explaining the column
COMMENT ON COLUMN user_preferences.llm_api_key IS 'User-provided LLM API key (encrypted in production)';
