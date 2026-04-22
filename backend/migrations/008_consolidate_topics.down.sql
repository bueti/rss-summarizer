-- This migration cannot be fully reversed as it consolidates data
-- Articles will keep their consolidated topics
COMMENT ON TABLE topics IS 'Auto-detected topics are automatically cleaned up if not referenced by articles';
