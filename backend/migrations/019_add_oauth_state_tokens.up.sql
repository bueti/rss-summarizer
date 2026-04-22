-- Create table for storing OAuth state tokens
CREATE TABLE oauth_state_tokens (
    state VARCHAR(255) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for cleanup of expired tokens
CREATE INDEX idx_oauth_state_tokens_created_at ON oauth_state_tokens(created_at);
