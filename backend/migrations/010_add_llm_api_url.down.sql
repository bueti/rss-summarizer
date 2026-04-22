-- Remove llm_api_url column
ALTER TABLE user_preferences
DROP COLUMN llm_api_url;
