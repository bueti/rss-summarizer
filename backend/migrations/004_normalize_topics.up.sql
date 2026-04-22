-- Clean up duplicate topics with case-insensitive matching
-- This migration normalizes topic names and removes duplicates

-- Step 1: Create a temp table with normalized topic names and their canonical ID
CREATE TEMP TABLE topic_mapping AS
SELECT DISTINCT ON (user_id, LOWER(TRIM(name)))
    id as canonical_id,
    user_id,
    INITCAP(LOWER(TRIM(name))) as normalized_name,
    name as original_name,
    preference,
    is_custom,
    created_at
FROM topics
ORDER BY user_id, LOWER(TRIM(name)), 
    -- Prefer custom topics
    is_custom DESC,
    -- Prefer topics with non-normal preference
    CASE WHEN preference != 'normal' THEN 0 ELSE 1 END,
    -- Prefer older topics
    created_at ASC;

-- Step 2: Delete all topics that are NOT the canonical version
DELETE FROM topics
WHERE id NOT IN (SELECT canonical_id FROM topic_mapping);

-- Step 3: Update remaining topics with normalized names
UPDATE topics t
SET name = tm.normalized_name
FROM topic_mapping tm
WHERE t.id = tm.canonical_id
AND t.name != tm.normalized_name;

-- Step 4: Add comment
COMMENT ON TABLE topics IS 'Topics are stored with normalized names (Title Case, trimmed)';
