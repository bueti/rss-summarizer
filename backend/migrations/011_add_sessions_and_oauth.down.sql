-- Drop function
DROP FUNCTION IF EXISTS delete_expired_sessions();

-- Drop sessions table
DROP TABLE IF EXISTS sessions;

-- Drop index and columns from users table
DROP INDEX IF EXISTS idx_users_google_id;
ALTER TABLE users DROP COLUMN IF EXISTS picture_url;
ALTER TABLE users DROP COLUMN IF EXISTS google_id;
