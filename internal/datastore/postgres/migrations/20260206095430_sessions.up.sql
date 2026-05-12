CREATE TABLE
    sessions (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        user_id VARCHAR,
        username VARCHAR,
        token TEXT,
        metadata JSONB,
        organization_id UUID REFERENCES organizations (id),
        login_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
            logout_at TIMESTAMP
        WITH
            TIME ZONE,
            device_info TEXT,
            ip_address TEXT,
            location TEXT,
            status VARCHAR NOT NULL,
            created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
            deleted_at TIMESTAMP
        WITH
            TIME ZONE
    );

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_organization_id ON sessions (organization_id);
CREATE INDEX idx_sessions_status ON sessions (status);
CREATE INDEX idx_sessions_login_at ON sessions (login_at);
CREATE INDEX idx_sessions_logout_at ON sessions (logout_at);
CREATE INDEX idx_sessions_ip_address ON sessions (ip_address);
