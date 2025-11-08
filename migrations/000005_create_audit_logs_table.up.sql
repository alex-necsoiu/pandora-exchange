-- Create audit_logs table for immutable audit trail
-- This table stores all security-relevant events for compliance and forensics

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Event classification
    event_type VARCHAR(100) NOT NULL,
    event_category VARCHAR(50) NOT NULL, -- authentication, authorization, data_access, data_modification, security, compliance
    severity VARCHAR(20) NOT NULL, -- info, warning, high, critical
    
    -- Actor information
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_type VARCHAR(50) NOT NULL, -- user, system, admin, api
    actor_identifier VARCHAR(255), -- email, API key ID, service name
    
    -- Action details
    action VARCHAR(100) NOT NULL, -- login, logout, create, update, delete, access, etc.
    resource_type VARCHAR(100), -- user, wallet, order, kyc_document, etc.
    resource_id VARCHAR(255), -- ID of the affected resource
    
    -- Request context
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(100), -- For correlation with application logs
    session_id VARCHAR(100),
    
    -- Event payload
    metadata JSONB, -- Additional event-specific data
    previous_state JSONB, -- Before state for data modifications
    new_state JSONB, -- After state for data modifications
    
    -- Security
    status VARCHAR(20) NOT NULL, -- success, failure, error
    failure_reason TEXT, -- For failed operations
    
    -- Compliance
    retention_until TIMESTAMP WITH TIME ZONE, -- For GDPR/compliance retention policies
    is_sensitive BOOLEAN DEFAULT false, -- Flags logs containing PII
    
    -- Timestamps (immutable - no updated_at)
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for common query patterns
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_event_category ON audit_logs(event_category);
CREATE INDEX idx_audit_logs_severity ON audit_logs(severity);
CREATE INDEX idx_audit_logs_actor_identifier ON audit_logs(actor_identifier);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_ip_address ON audit_logs(ip_address);

-- Composite index for common filtering
CREATE INDEX idx_audit_logs_user_created ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_category_created ON audit_logs(event_category, created_at DESC);

-- GIN index for JSONB metadata queries
CREATE INDEX idx_audit_logs_metadata ON audit_logs USING GIN (metadata);

-- Comment for documentation
COMMENT ON TABLE audit_logs IS 'Immutable audit trail for security, compliance, and forensic analysis';
COMMENT ON COLUMN audit_logs.event_type IS 'Specific event identifier (e.g., user.login, kyc.approved)';
COMMENT ON COLUMN audit_logs.event_category IS 'High-level categorization for filtering and reporting';
COMMENT ON COLUMN audit_logs.severity IS 'Event severity for alerting and monitoring';
COMMENT ON COLUMN audit_logs.metadata IS 'Flexible JSONB field for event-specific structured data';
COMMENT ON COLUMN audit_logs.retention_until IS 'Auto-deletion timestamp for compliance with data retention policies';
