-- Add email_source_id to articles table for filtering email newsletters
ALTER TABLE articles ADD COLUMN email_source_id UUID REFERENCES email_sources(id) ON DELETE SET NULL;

-- Index for filtering by email source
CREATE INDEX idx_articles_email_source_id ON articles(email_source_id) WHERE email_source_id IS NOT NULL;

COMMENT ON COLUMN articles.email_source_id IS 'Email source for email-sourced articles (NULL for RSS articles)';
