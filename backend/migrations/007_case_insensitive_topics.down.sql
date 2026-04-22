-- Revert to case-sensitive unique constraint

-- Drop the case-insensitive unique index
DROP INDEX IF EXISTS topics_user_id_lower_name_key;

-- Recreate the old case-sensitive unique constraint
ALTER TABLE topics ADD CONSTRAINT topics_user_id_name_key UNIQUE (user_id, name);
