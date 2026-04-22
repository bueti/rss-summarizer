-- Remove llm_api_key column from user_preferences table
ALTER TABLE user_preferences
DROP COLUMN llm_api_key;
