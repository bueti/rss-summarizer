-- Migration 015: Normalize topic names to title case and deduplicate

-- Function to normalize topics array (title case, deduplicate, handle acronyms)
CREATE OR REPLACE FUNCTION normalize_topics(topics TEXT[])
RETURNS TEXT[] AS $$
DECLARE
    result TEXT[] := '{}';
    seen_topics TEXT[] := '{}';
    topic TEXT;
    normalized TEXT;
    lower_topic TEXT;
BEGIN
    IF topics IS NULL THEN
        RETURN NULL;
    END IF;

    FOREACH topic IN ARRAY topics LOOP
        -- Trim whitespace
        topic := TRIM(topic);

        -- Skip empty strings
        IF topic = '' THEN
            CONTINUE;
        END IF;

        lower_topic := LOWER(topic);

        -- Only add if we haven't seen this topic (case-insensitive)
        IF NOT (lower_topic = ANY(seen_topics)) THEN
            seen_topics := array_append(seen_topics, lower_topic);

            -- Handle common acronyms and technical terms
            normalized := CASE lower_topic
                WHEN 'ai' THEN 'AI'
                WHEN 'api' THEN 'API'
                WHEN 'apis' THEN 'APIs'
                WHEN 'aws' THEN 'AWS'
                WHEN 'gcp' THEN 'GCP'
                WHEN 'devops' THEN 'DevOps'
                WHEN 'cicd' THEN 'CI/CD'
                WHEN 'ml' THEN 'ML'
                WHEN 'llm' THEN 'LLM'
                WHEN 'llms' THEN 'LLMs'
                WHEN 'ui' THEN 'UI'
                WHEN 'ux' THEN 'UX'
                WHEN 'css' THEN 'CSS'
                WHEN 'html' THEN 'HTML'
                WHEN 'json' THEN 'JSON'
                WHEN 'xml' THEN 'XML'
                WHEN 'sql' THEN 'SQL'
                WHEN 'nosql' THEN 'NoSQL'
                WHEN 'rest' THEN 'REST'
                WHEN 'graphql' THEN 'GraphQL'
                WHEN 'grpc' THEN 'gRPC'
                WHEN 'http' THEN 'HTTP'
                WHEN 'https' THEN 'HTTPS'
                WHEN 'ssh' THEN 'SSH'
                WHEN 'tcp' THEN 'TCP'
                WHEN 'udp' THEN 'UDP'
                WHEN 'dns' THEN 'DNS'
                WHEN 'cdn' THEN 'CDN'
                WHEN 'saas' THEN 'SaaS'
                WHEN 'paas' THEN 'PaaS'
                WHEN 'iaas' THEN 'IaaS'
                WHEN 'oauth' THEN 'OAuth'
                WHEN 'jwt' THEN 'JWT'
                WHEN 'tls' THEN 'TLS'
                WHEN 'ssl' THEN 'SSL'
                WHEN 'vpn' THEN 'VPN'
                ELSE INITCAP(LOWER(topic))
            END;

            result := array_append(result, normalized);
        END IF;
    END LOOP;

    RETURN result;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Update all articles to normalize their topics
UPDATE articles
SET topics = normalize_topics(topics)
WHERE topics IS NOT NULL;

-- Log results
DO $$
DECLARE
    article_count INTEGER;
    distinct_topics INTEGER;
BEGIN
    SELECT COUNT(*) INTO article_count FROM articles WHERE topics IS NOT NULL;
    SELECT COUNT(DISTINCT topic) INTO distinct_topics
    FROM (SELECT UNNEST(topics) as topic FROM articles) t;

    RAISE NOTICE '============================================================================';
    RAISE NOTICE 'Migration 015 Complete!';
    RAISE NOTICE '============================================================================';
    RAISE NOTICE 'Updated % articles with normalized topics', article_count;
    RAISE NOTICE 'Total distinct topics: %', distinct_topics;
    RAISE NOTICE '============================================================================';
END $$;
