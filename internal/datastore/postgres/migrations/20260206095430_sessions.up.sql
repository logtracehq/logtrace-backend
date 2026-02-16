CREATE TABLE
    sessions (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        user_id VARCHAR,
        username VARCHAR,
        login_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
            organization_id UUID REFERENCES organizations (id),
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
