-- Make topics unique constraint case-insensitive
-- This prevents duplicates like "AI" and "ai" from being stored

-- First, clean up any remaining case-insensitive duplicates
-- Keep only the Title Case version of each topic
WITH ranked_topics AS (
    SELECT
        id,
        user_id,
        name,
        ROW_NUMBER() OVER (
            PARTITION BY user_id, LOWER(name)
            ORDER BY
                -- Prefer custom topics
                is_custom DESC,
                -- Prefer non-normal preferences
                CASE WHEN preference != 'normal' THEN 0 ELSE 1 END,
                -- Prefer older topics
                created_at ASC
        ) as rn
    FROM topics
)
DELETE FROM topics
WHERE id IN (
    SELECT id FROM ranked_topics WHERE rn > 1
);

-- Drop the old case-sensitive unique constraint
ALTER TABLE topics DROP CONSTRAINT IF EXISTS topics_user_id_name_key;

-- Create a case-insensitive unique index
-- Note: This is a unique index, not a constraint, so ON CONFLICT needs special syntax
CREATE UNIQUE INDEX IF NOT EXISTS topics_user_id_lower_name_key
ON topics (user_id, LOWER(name));

-- Add a comment
COMMENT ON INDEX topics_user_id_lower_name_key IS 'Case-insensitive unique index on topic names per user';
