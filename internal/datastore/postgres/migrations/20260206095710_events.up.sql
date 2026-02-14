CREATE TABLE
    events (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        type VARCHAR NOT NULL,
        username VARCHAR,
        user_id UUID,
        http_method VARCHAR,
        http_status VARCHAR,
        organization_id UUID NOT NULL,
        http_endpoint VARCHAR,
        client_ip VARCHAR,
        client_user_agent VARCHAR,
        geo_ip_location VARCHAR,
        action_name VARCHAR,
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
