-- Consolidate overly specific topics into broad categories
-- This migration maps specific topics to top-level categories

-- Create a mapping table for topic consolidation
DROP TABLE IF EXISTS topic_mapping;
CREATE TEMP TABLE topic_mapping AS
SELECT
    old_name,
    new_name
FROM (VALUES
    -- Programming Languages (consolidate versions/frameworks)
    ('Golang', 'Go'),
    ('Go Programming', 'Go'),
    ('Rust Programming', 'Rust'),
    ('Python Programming', 'Python'),
    ('Javascript', 'JavaScript'),
    ('Typescript', 'TypeScript'),

    -- Cloud & Infrastructure
    ('Kubernetes Deployment', 'Kubernetes'),
    ('K8S', 'Kubernetes'),
    ('K8s', 'Kubernetes'),
    ('Docker Containers', 'Docker'),
    ('Container Orchestration', 'Kubernetes'),
    ('Cloud Computing', 'Cloud'),
    ('AWS Services', 'AWS'),
    ('Amazon Web Services', 'AWS'),
    ('Google Cloud Platform', 'GCP'),
    ('Azure Cloud', 'Azure'),
    ('Cloud Infrastructure', 'Cloud'),

    -- DevOps & Tools
    ('Devops', 'DevOps'),
    ('CI/CD Pipeline', 'DevOps'),
    ('Continuous Integration', 'DevOps'),
    ('Infrastructure As Code', 'DevOps'),

    -- Security
    ('Cybersecurity', 'Security'),
    ('Information Security', 'Security'),
    ('Application Security', 'Security'),
    ('Network Security', 'Security'),
    ('Security Vulnerability', 'Security'),

    -- AI & Machine Learning
    ('Artificial Intelligence', 'AI'),
    ('Machine Learning', 'AI'),
    ('Deep Learning', 'AI'),
    ('Neural Networks', 'AI'),
    ('Large Language Models', 'AI'),
    ('LLM', 'AI'),
    ('Llm', 'AI'),
    ('ChatGPT', 'AI'),
    ('GPT', 'AI'),

    -- Databases
    ('PostgreSQL', 'Databases'),
    ('Postgres', 'Databases'),
    ('MySQL', 'Databases'),
    ('MongoDB', 'Databases'),
    ('Database Design', 'Databases'),
    ('SQL', 'Databases'),

    -- Web Development
    ('Web Development', 'Web'),
    ('Frontend Development', 'Web'),
    ('Backend Development', 'Web'),
    ('Full Stack', 'Web'),
    ('API Development', 'APIs'),
    ('REST API', 'APIs'),
    ('GraphQL', 'APIs'),

    -- Software Engineering
    ('Software Development', 'Engineering'),
    ('Software Engineering', 'Engineering'),
    ('Code Quality', 'Engineering'),
    ('Software Architecture', 'Architecture'),
    ('System Design', 'Architecture'),
    ('Microservices', 'Architecture'),

    -- Performance
    ('Performance Optimization', 'Performance'),
    ('Code Optimization', 'Performance'),
    ('System Performance', 'Performance'),

    -- Testing
    ('Software Testing', 'Testing'),
    ('Unit Testing', 'Testing'),
    ('Integration Testing', 'Testing'),
    ('Test Automation', 'Testing'),

    -- General Tech
    ('Technology', 'Tech'),
    ('Open Source', 'Open Source'),
    ('Opensource', 'Open Source'),
    ('Version Control', 'Git'),
    ('Source Control', 'Git')
) AS mapping(old_name, new_name);

-- Update articles to use consolidated topic names
UPDATE articles
SET topics = (
    SELECT array_agg(DISTINCT COALESCE(tm.new_name, t.topic))
    FROM unnest(topics) AS t(topic)
    LEFT JOIN topic_mapping tm ON LOWER(t.topic) = LOWER(tm.old_name)
    WHERE COALESCE(tm.new_name, t.topic) IS NOT NULL
)
WHERE EXISTS (
    SELECT 1
    FROM unnest(topics) AS t(topic)
    INNER JOIN topic_mapping tm ON LOWER(t.topic) = LOWER(tm.old_name)
);

-- Delete specific topics and create consolidated ones
DO $$
DECLARE
    user_rec RECORD;
    topic_rec RECORD;
    existing_id UUID;
BEGIN
    -- For each user
    FOR user_rec IN SELECT DISTINCT user_id FROM topics LOOP
        -- For each topic mapping
        FOR topic_rec IN SELECT DISTINCT new_name FROM topic_mapping LOOP
            -- Check if consolidated topic exists
            SELECT id INTO existing_id
            FROM topics
            WHERE user_id = user_rec.user_id
            AND LOWER(name) = LOWER(topic_rec.new_name);

            -- If doesn't exist, create it
            IF existing_id IS NULL THEN
                -- Check if any old topics exist that should be consolidated
                IF EXISTS (
                    SELECT 1 FROM topics t
                    INNER JOIN topic_mapping tm ON LOWER(t.name) = LOWER(tm.old_name)
                    WHERE t.user_id = user_rec.user_id
                    AND tm.new_name = topic_rec.new_name
                ) THEN
                    -- Create the consolidated topic
                    INSERT INTO topics (id, user_id, name, preference, is_custom, created_at, updated_at)
                    SELECT
                        uuid_generate_v4(),
                        user_rec.user_id,
                        topic_rec.new_name,
                        COALESCE(
                            (SELECT preference FROM topics t
                             INNER JOIN topic_mapping tm ON LOWER(t.name) = LOWER(tm.old_name)
                             WHERE t.user_id = user_rec.user_id
                             AND tm.new_name = topic_rec.new_name
                             AND preference != 'normal'
                             LIMIT 1),
                            'normal'
                        ),
                        false,
                        NOW(),
                        NOW()
                    ON CONFLICT (user_id, LOWER(name)) DO NOTHING;
                END IF;
            END IF;
        END LOOP;

        -- Delete old specific topics that have been consolidated
        DELETE FROM topics
        WHERE user_id = user_rec.user_id
        AND id IN (
            SELECT t.id
            FROM topics t
            INNER JOIN topic_mapping tm ON LOWER(t.name) = LOWER(tm.old_name)
            WHERE t.user_id = user_rec.user_id
        );
    END LOOP;
END $$;

-- Delete unused auto-detected topics
DELETE FROM topics t
WHERE is_custom = false
AND NOT EXISTS (
    SELECT 1 FROM articles a
    WHERE t.name = ANY(a.topics)
    AND a.user_id = t.user_id
);

-- Add comment
COMMENT ON TABLE topics IS 'Topics consolidated to broad, top-level categories only';
