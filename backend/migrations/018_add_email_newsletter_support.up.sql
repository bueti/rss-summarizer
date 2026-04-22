-- Add email newsletter support for receiving and processing newsletters via Gmail

-- Email sources table (connected email accounts)
CREATE TABLE email_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_address VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'gmail', -- 'gmail', 'outlook', 'imap'
    access_token TEXT NOT NULL, -- encrypted OAuth access token
    refresh_token TEXT NOT NULL, -- encrypted OAuth refresh token
    token_expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_fetched_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_user_email_provider UNIQUE (user_id, email_address, provider)
);

-- Newsletter filters table (rules for identifying newsletters)
CREATE TABLE newsletter_filters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_source_id UUID NOT NULL REFERENCES email_sources(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    sender_pattern VARCHAR(500) NOT NULL, -- e.g., "*@substack.com" or "newsletter@example.com"
    subject_pattern VARCHAR(500), -- optional regex for subject line
    label_or_folder VARCHAR(255), -- Gmail label or Outlook folder name (optional)
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_filter_name_per_user UNIQUE (user_id, name)
);

-- Add source_type and email_message_id to articles table
ALTER TABLE articles
ADD COLUMN source_type VARCHAR(10) NOT NULL DEFAULT 'rss',
ADD COLUMN email_message_id VARCHAR(255),
ADD CONSTRAINT check_source_type CHECK (source_type IN ('rss', 'email'));

-- Add indexes for performance
CREATE INDEX idx_email_sources_user_id ON email_sources(user_id);
CREATE INDEX idx_email_sources_is_active ON email_sources(user_id, is_active) WHERE is_active = true;
CREATE INDEX idx_newsletter_filters_user_id ON newsletter_filters(user_id);
CREATE INDEX idx_newsletter_filters_email_source ON newsletter_filters(email_source_id);
CREATE INDEX idx_newsletter_filters_is_active ON newsletter_filters(email_source_id, is_active) WHERE is_active = true;
CREATE INDEX idx_articles_source_type ON articles(source_type);
CREATE INDEX idx_articles_email_message_id ON articles(email_message_id) WHERE email_message_id IS NOT NULL;

-- Add comments for documentation
COMMENT ON TABLE email_sources IS 'Connected email accounts for fetching newsletters via OAuth';
COMMENT ON TABLE newsletter_filters IS 'Rules for identifying and filtering newsletter emails';
COMMENT ON COLUMN email_sources.access_token IS 'Encrypted OAuth access token (1 hour expiry)';
COMMENT ON COLUMN email_sources.refresh_token IS 'Encrypted OAuth refresh token (no expiry)';
COMMENT ON COLUMN email_sources.provider IS 'Email provider: gmail, outlook, or imap';
COMMENT ON COLUMN newsletter_filters.sender_pattern IS 'Pattern for matching sender email (e.g., *@substack.com)';
COMMENT ON COLUMN newsletter_filters.subject_pattern IS 'Optional regex pattern for subject line matching';
COMMENT ON COLUMN articles.source_type IS 'Source of article: rss or email';
COMMENT ON COLUMN articles.email_message_id IS 'Gmail/Outlook message ID for email-sourced articles';
