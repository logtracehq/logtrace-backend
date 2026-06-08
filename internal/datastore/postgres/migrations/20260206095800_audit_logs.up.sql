CREATE TABLE
    audit_logs (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        name VARCHAR,
        organization_id UUID REFERENCES organizations (id),
        user_id VARCHAR,
        username VARCHAR,
        metadata JSONB,
        description TEXT,
        request_details JSONB DEFAULT '{}'::JSONB,
        timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at TIMESTAMP WITH TIME ZONE
    );

CREATE INDEX idx_audit_logs_organization_id ON audit_logs (organization_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs (timestamp);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);
