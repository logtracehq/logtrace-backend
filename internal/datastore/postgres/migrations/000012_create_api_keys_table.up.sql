CREATE TYPE api_key_scope AS ENUM ('readonly', 'readwrite', 'writeonly');

CREATE TABLE
    api_keys (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        created_by UUID REFERENCES users (id),
        value VARCHAR NOT NULL,
        organization_id UUID REFERENCES organizations (id),
        resource_id UUID REFERENCES resources (id),
        scope api_key_scope NOT NULL,
        last_used_at TIMESTAMP
        WITH
            TIME ZONE,
            name VARCHAR NOT NULL,
            expires_at TIMESTAMP
        WITH
            TIME ZONE,
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
