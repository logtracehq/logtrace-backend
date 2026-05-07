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
