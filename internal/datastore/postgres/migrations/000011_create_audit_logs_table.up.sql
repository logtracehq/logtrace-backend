-- Create audit_logs table
CREATE TABLE
    audit_logs (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        action VARCHAR,
        project_id UUID REFERENCES projects (id),
        organization_id UUID REFERENCES organizations (id),
        ip_address VARCHAR,
        user_id UUID REFERENCES users (id),
        metadata JSONB,
        timestamp TIMESTAMP
        WITH
            TIME ZONE NOT NULL,
            created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
            request_id VARCHAR,
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
            deleted_at TIMESTAMP
        WITH
            TIME ZONE
    );
