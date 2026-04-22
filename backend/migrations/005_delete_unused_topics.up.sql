-- Delete auto-detected topics that are not referenced by any articles
-- This helps reduce topic clutter

DELETE FROM topics t
WHERE t.is_custom = false
AND NOT EXISTS (
    SELECT 1 FROM articles a
    WHERE t.name = ANY(a.topics)
    AND a.user_id = t.user_id
);

COMMENT ON TABLE topics IS 'Auto-detected topics are automatically cleaned up if not referenced by articles';
