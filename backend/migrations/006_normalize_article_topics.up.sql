-- Normalize topics in articles table to match cleaned topics table
-- This removes topics that no longer exist in the topics table

-- Update each article's topics array to only include topics that exist in the topics table
UPDATE articles
SET topics = (
    SELECT ARRAY_AGG(DISTINCT t.name ORDER BY t.name)
    FROM topics t
    WHERE t.user_id = articles.user_id
    AND t.name = ANY(articles.topics)
)
WHERE topics IS NOT NULL
AND array_length(topics, 1) > 0;

-- Clean up NULL topics (convert to empty array)
UPDATE articles
SET topics = ARRAY[]::TEXT[]
WHERE topics IS NULL;

COMMENT ON COLUMN articles.topics IS 'Article topics - normalized to match topics table';
