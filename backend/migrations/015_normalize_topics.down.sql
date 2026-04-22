-- Migration 015 down: Remove normalize_topics function
-- Note: We cannot revert the normalization of existing data

DROP FUNCTION IF EXISTS normalize_topics(TEXT[]);
