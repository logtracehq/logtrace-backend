CREATE TABLE
    audit_logs (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        action VARCHAR,
        organization_id UUID REFERENCES organizations (id),
        ip_address VARCHAR,
        user_id VARCHAR,
        username VARCHAR,
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
