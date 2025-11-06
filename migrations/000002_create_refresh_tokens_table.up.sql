-- Create refresh_tokens table
-- This table stores JWT refresh tokens for authentication
-- Following ARCHITECTURE.md section 8

CREATE TABLE IF NOT EXISTS refresh_tokens (
    token TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    ip_address TEXT,
    user_agent TEXT
);

-- Create indexes for performance and cleanup
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_revoked_at ON refresh_tokens(revoked_at) WHERE revoked_at IS NOT NULL;

-- Add comments for documentation
COMMENT ON TABLE refresh_tokens IS 'Stores JWT refresh tokens for user authentication';
COMMENT ON COLUMN refresh_tokens.token IS 'Unique refresh token string (hashed)';
COMMENT ON COLUMN refresh_tokens.user_id IS 'Reference to user who owns this token';
COMMENT ON COLUMN refresh_tokens.expires_at IS 'Token expiration timestamp';
COMMENT ON COLUMN refresh_tokens.created_at IS 'Timestamp when token was created';
COMMENT ON COLUMN refresh_tokens.revoked_at IS 'Timestamp when token was revoked (NULL if active)';
COMMENT ON COLUMN refresh_tokens.ip_address IS 'IP address where token was created (audit trail)';
COMMENT ON COLUMN refresh_tokens.user_agent IS 'User agent where token was created (audit trail)';
