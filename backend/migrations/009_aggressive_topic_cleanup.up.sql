-- Aggressive topic cleanup and consolidation
-- This migration drastically reduces the number of topics by consolidating similar ones

-- Step 1: Create comprehensive mapping of all variations to canonical topics
DROP TABLE IF EXISTS topic_consolidation;
CREATE TEMP TABLE topic_consolidation AS
SELECT old_topic, new_topic FROM (
    SELECT DISTINCT name as old_topic,
    CASE
        -- Programming Languages
        WHEN LOWER(name) LIKE '%golang%' OR LOWER(name) LIKE '%go %' OR LOWER(name) = 'go' THEN 'Go'
        WHEN LOWER(name) LIKE '%rust%' THEN 'Rust'
        WHEN LOWER(name) LIKE '%python%' THEN 'Python'
        WHEN LOWER(name) LIKE '%javascript%' OR LOWER(name) LIKE '%js%' THEN 'JavaScript'
        WHEN LOWER(name) LIKE '%typescript%' OR LOWER(name) LIKE '%ts%' THEN 'TypeScript'
        WHEN LOWER(name) LIKE '%java%' AND LOWER(name) NOT LIKE '%javascript%' THEN 'Java'
        WHEN LOWER(name) LIKE '%c++%' OR LOWER(name) LIKE '%cpp%' THEN 'C++'

        -- Cloud & Containers
        WHEN LOWER(name) LIKE '%kubernetes%' OR LOWER(name) LIKE '%k8s%' THEN 'Kubernetes'
        WHEN LOWER(name) LIKE '%docker%' OR LOWER(name) LIKE '%container%' THEN 'Docker'
        WHEN LOWER(name) LIKE '%aws%' OR LOWER(name) LIKE '%amazon%' THEN 'AWS'
        WHEN LOWER(name) LIKE '%gcp%' OR LOWER(name) LIKE '%google cloud%' THEN 'GCP'
        WHEN LOWER(name) LIKE '%azure%' OR LOWER(name) LIKE '%microsoft cloud%' THEN 'Azure'
        WHEN LOWER(name) LIKE '%cloud%' THEN 'Cloud'

        -- DevOps & Tools
        WHEN LOWER(name) LIKE '%devops%' OR LOWER(name) LIKE '%ci/cd%' OR LOWER(name) LIKE '%cicd%'
             OR LOWER(name) LIKE '%continuous%' OR LOWER(name) LIKE '%deployment%' THEN 'DevOps'
        WHEN LOWER(name) LIKE '%git%' OR LOWER(name) LIKE '%version control%' THEN 'Git'
        WHEN LOWER(name) LIKE '%linux%' OR LOWER(name) LIKE '%unix%' THEN 'Linux'

        -- Security
        WHEN LOWER(name) LIKE '%security%' OR LOWER(name) LIKE '%cyber%'
             OR LOWER(name) LIKE '%vulnerability%' OR LOWER(name) LIKE '%exploit%' THEN 'Security'

        -- AI & ML
        WHEN LOWER(name) LIKE '%ai%' OR LOWER(name) LIKE '%artificial%'
             OR LOWER(name) LIKE '%machine learning%' OR LOWER(name) LIKE '%ml%'
             OR LOWER(name) LIKE '%deep learning%' OR LOWER(name) LIKE '%neural%'
             OR LOWER(name) LIKE '%llm%' OR LOWER(name) LIKE '%gpt%'
             OR LOWER(name) LIKE '%chatgpt%' OR LOWER(name) LIKE '%language model%' THEN 'AI'

        -- Databases
        WHEN LOWER(name) LIKE '%database%' OR LOWER(name) LIKE '%sql%'
             OR LOWER(name) LIKE '%postgres%' OR LOWER(name) LIKE '%mysql%'
             OR LOWER(name) LIKE '%mongodb%' OR LOWER(name) LIKE '%redis%' THEN 'Databases'

        -- Web Development
        WHEN LOWER(name) LIKE '%web%' OR LOWER(name) LIKE '%frontend%'
             OR LOWER(name) LIKE '%backend%' OR LOWER(name) LIKE '%react%'
             OR LOWER(name) LIKE '%vue%' OR LOWER(name) LIKE '%angular%'
             OR LOWER(name) LIKE '%html%' OR LOWER(name) LIKE '%css%' THEN 'Web'

        -- APIs
        WHEN LOWER(name) LIKE '%api%' OR LOWER(name) LIKE '%rest%'
             OR LOWER(name) LIKE '%graphql%' OR LOWER(name) LIKE '%grpc%' THEN 'APIs'

        -- Architecture & Design
        WHEN LOWER(name) LIKE '%architecture%' OR LOWER(name) LIKE '%design%'
             OR LOWER(name) LIKE '%microservice%' OR LOWER(name) LIKE '%system design%' THEN 'Architecture'

        -- Performance
        WHEN LOWER(name) LIKE '%performance%' OR LOWER(name) LIKE '%optimization%'
             OR LOWER(name) LIKE '%scalability%' OR LOWER(name) LIKE '%scale%' THEN 'Performance'

        -- Testing
        WHEN LOWER(name) LIKE '%test%' OR LOWER(name) LIKE '%qa%'
             OR LOWER(name) LIKE '%quality%' THEN 'Testing'

        -- Engineering/Development
        WHEN LOWER(name) LIKE '%engineering%' OR LOWER(name) LIKE '%development%'
             OR LOWER(name) LIKE '%software%' OR LOWER(name) LIKE '%code%'
             OR LOWER(name) LIKE '%programming%' THEN 'Engineering'

        -- Open Source
        WHEN LOWER(name) LIKE '%open source%' OR LOWER(name) LIKE '%opensource%' THEN 'Open Source'

        -- Catch-all: If it doesn't match anything, mark for deletion
        ELSE NULL
    END as new_topic
    FROM topics
) AS mapping
WHERE new_topic IS NOT NULL;

-- Step 2: Update articles to use consolidated topics
UPDATE articles
SET topics = (
    SELECT COALESCE(
        array_agg(DISTINCT tc.new_topic),
        ARRAY[]::TEXT[]
    )
    FROM unnest(topics) AS t(topic)
    LEFT JOIN topic_consolidation tc ON t.topic = tc.old_topic
    WHERE tc.new_topic IS NOT NULL
)
WHERE EXISTS (
    SELECT 1 FROM unnest(topics) AS t(topic)
    WHERE t.topic IN (SELECT old_topic FROM topic_consolidation)
);

-- Step 3: Delete ALL existing topics
DELETE FROM topics;

-- Step 4: Create only the canonical topics that are actually used in articles
INSERT INTO topics (id, user_id, name, preference, is_custom, created_at, updated_at)
SELECT DISTINCT
    uuid_generate_v4(),
    a.user_id,
    t.topic,
    'normal',
    false,
    NOW(),
    NOW()
FROM articles a
CROSS JOIN LATERAL unnest(a.topics) AS t(topic)
WHERE t.topic IS NOT NULL AND t.topic != ''
ON CONFLICT (user_id, LOWER(name)) DO NOTHING;

-- Step 5: Show summary
DO $$
DECLARE
    topic_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO topic_count FROM topics;
    RAISE NOTICE 'Topic cleanup complete. Remaining topics: %', topic_count;
END $$;

COMMENT ON TABLE topics IS 'Topics consolidated to 15-20 broad categories only';
