-- Preview what the topic cleanup will do
-- Run this to see what topics will be consolidated BEFORE running the migration

-- Show current topic count
SELECT 'Current Topics' AS status, COUNT(*) AS count FROM topics;

-- Show topics that will be consolidated
WITH consolidation AS (
    SELECT name as old_topic,
    CASE
        WHEN LOWER(name) LIKE '%golang%' OR LOWER(name) LIKE '%go %' OR LOWER(name) = 'go' THEN 'Go'
        WHEN LOWER(name) LIKE '%rust%' THEN 'Rust'
        WHEN LOWER(name) LIKE '%python%' THEN 'Python'
        WHEN LOWER(name) LIKE '%javascript%' OR LOWER(name) LIKE '%js%' THEN 'JavaScript'
        WHEN LOWER(name) LIKE '%typescript%' OR LOWER(name) LIKE '%ts%' THEN 'TypeScript'
        WHEN LOWER(name) LIKE '%java%' AND LOWER(name) NOT LIKE '%javascript%' THEN 'Java'
        WHEN LOWER(name) LIKE '%kubernetes%' OR LOWER(name) LIKE '%k8s%' THEN 'Kubernetes'
        WHEN LOWER(name) LIKE '%docker%' OR LOWER(name) LIKE '%container%' THEN 'Docker'
        WHEN LOWER(name) LIKE '%aws%' OR LOWER(name) LIKE '%amazon%' THEN 'AWS'
        WHEN LOWER(name) LIKE '%cloud%' THEN 'Cloud'
        WHEN LOWER(name) LIKE '%devops%' OR LOWER(name) LIKE '%ci/cd%' THEN 'DevOps'
        WHEN LOWER(name) LIKE '%security%' OR LOWER(name) LIKE '%cyber%' THEN 'Security'
        WHEN LOWER(name) LIKE '%ai%' OR LOWER(name) LIKE '%machine learning%' OR LOWER(name) LIKE '%llm%' THEN 'AI'
        WHEN LOWER(name) LIKE '%database%' OR LOWER(name) LIKE '%sql%' OR LOWER(name) LIKE '%postgres%' THEN 'Databases'
        WHEN LOWER(name) LIKE '%web%' OR LOWER(name) LIKE '%frontend%' OR LOWER(name) LIKE '%backend%' THEN 'Web'
        WHEN LOWER(name) LIKE '%api%' OR LOWER(name) LIKE '%rest%' THEN 'APIs'
        WHEN LOWER(name) LIKE '%architecture%' OR LOWER(name) LIKE '%design%' THEN 'Architecture'
        WHEN LOWER(name) LIKE '%performance%' OR LOWER(name) LIKE '%optimization%' THEN 'Performance'
        WHEN LOWER(name) LIKE '%test%' THEN 'Testing'
        WHEN LOWER(name) LIKE '%engineering%' OR LOWER(name) LIKE '%development%' THEN 'Engineering'
        WHEN LOWER(name) LIKE '%git%' THEN 'Git'
        WHEN LOWER(name) LIKE '%linux%' THEN 'Linux'
        ELSE NULL
    END as new_topic
    FROM topics
)
SELECT
    new_topic AS "Will Become",
    COUNT(*) AS "Topic Count",
    string_agg(old_topic, ', ' ORDER BY old_topic) AS "Consolidating These"
FROM consolidation
WHERE new_topic IS NOT NULL
GROUP BY new_topic
ORDER BY COUNT(*) DESC;

-- Show topics that will be DELETED (no mapping)
WITH consolidation AS (
    SELECT name as old_topic,
    CASE
        WHEN LOWER(name) LIKE '%golang%' OR LOWER(name) LIKE '%go %' OR LOWER(name) = 'go' THEN 'Go'
        WHEN LOWER(name) LIKE '%rust%' THEN 'Rust'
        WHEN LOWER(name) LIKE '%python%' THEN 'Python'
        WHEN LOWER(name) LIKE '%javascript%' OR LOWER(name) LIKE '%js%' THEN 'JavaScript'
        WHEN LOWER(name) LIKE '%kubernetes%' OR LOWER(name) LIKE '%k8s%' THEN 'Kubernetes'
        WHEN LOWER(name) LIKE '%docker%' OR LOWER(name) LIKE '%container%' THEN 'Docker'
        WHEN LOWER(name) LIKE '%security%' OR LOWER(name) LIKE '%cyber%' THEN 'Security'
        WHEN LOWER(name) LIKE '%ai%' OR LOWER(name) LIKE '%machine learning%' OR LOWER(name) LIKE '%llm%' THEN 'AI'
        WHEN LOWER(name) LIKE '%database%' OR LOWER(name) LIKE '%sql%' THEN 'Databases'
        WHEN LOWER(name) LIKE '%web%' OR LOWER(name) LIKE '%api%' THEN 'Web'
        ELSE NULL
    END as new_topic
    FROM topics
)
SELECT
    'Will be DELETED' AS status,
    COUNT(*) AS count,
    string_agg(old_topic, ', ' ORDER BY old_topic) AS topics
FROM consolidation
WHERE new_topic IS NULL;

-- Preview final topic list (what will remain)
WITH consolidation AS (
    SELECT DISTINCT
    CASE
        WHEN LOWER(name) LIKE '%golang%' OR LOWER(name) LIKE '%go %' OR LOWER(name) = 'go' THEN 'Go'
        WHEN LOWER(name) LIKE '%rust%' THEN 'Rust'
        WHEN LOWER(name) LIKE '%python%' THEN 'Python'
        WHEN LOWER(name) LIKE '%javascript%' OR LOWER(name) LIKE '%js%' THEN 'JavaScript'
        WHEN LOWER(name) LIKE '%typescript%' OR LOWER(name) LIKE '%ts%' THEN 'TypeScript'
        WHEN LOWER(name) LIKE '%java%' AND LOWER(name) NOT LIKE '%javascript%' THEN 'Java'
        WHEN LOWER(name) LIKE '%kubernetes%' OR LOWER(name) LIKE '%k8s%' THEN 'Kubernetes'
        WHEN LOWER(name) LIKE '%docker%' OR LOWER(name) LIKE '%container%' THEN 'Docker'
        WHEN LOWER(name) LIKE '%aws%' OR LOWER(name) LIKE '%amazon%' THEN 'AWS'
        WHEN LOWER(name) LIKE '%gcp%' OR LOWER(name) LIKE '%google cloud%' THEN 'GCP'
        WHEN LOWER(name) LIKE '%azure%' THEN 'Azure'
        WHEN LOWER(name) LIKE '%cloud%' THEN 'Cloud'
        WHEN LOWER(name) LIKE '%devops%' OR LOWER(name) LIKE '%ci/cd%' THEN 'DevOps'
        WHEN LOWER(name) LIKE '%security%' OR LOWER(name) LIKE '%cyber%' THEN 'Security'
        WHEN LOWER(name) LIKE '%ai%' OR LOWER(name) LIKE '%machine learning%' OR LOWER(name) LIKE '%llm%' THEN 'AI'
        WHEN LOWER(name) LIKE '%database%' OR LOWER(name) LIKE '%sql%' OR LOWER(name) LIKE '%postgres%' THEN 'Databases'
        WHEN LOWER(name) LIKE '%web%' OR LOWER(name) LIKE '%frontend%' OR LOWER(name) LIKE '%backend%' THEN 'Web'
        WHEN LOWER(name) LIKE '%api%' OR LOWER(name) LIKE '%rest%' THEN 'APIs'
        WHEN LOWER(name) LIKE '%architecture%' OR LOWER(name) LIKE '%design%' THEN 'Architecture'
        WHEN LOWER(name) LIKE '%performance%' OR LOWER(name) LIKE '%optimization%' THEN 'Performance'
        WHEN LOWER(name) LIKE '%test%' THEN 'Testing'
        WHEN LOWER(name) LIKE '%engineering%' OR LOWER(name) LIKE '%development%' THEN 'Engineering'
        WHEN LOWER(name) LIKE '%git%' THEN 'Git'
        WHEN LOWER(name) LIKE '%linux%' THEN 'Linux'
        WHEN LOWER(name) LIKE '%open source%' THEN 'Open Source'
        ELSE NULL
    END as new_topic
    FROM topics
)
SELECT
    'Final Topic List' AS status,
    string_agg(new_topic, ', ' ORDER BY new_topic) AS topics
FROM consolidation
WHERE new_topic IS NOT NULL;
