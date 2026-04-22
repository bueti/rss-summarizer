-- Add llm_api_url column for OpenAI-compatible custom endpoints
ALTER TABLE user_preferences
ADD COLUMN llm_api_url TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN user_preferences.llm_api_url IS 'Custom API URL for OpenAI-compatible endpoints (Ollama, Groq, etc.)';
