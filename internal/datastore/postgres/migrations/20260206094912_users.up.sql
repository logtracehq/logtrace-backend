CREATE TYPE user_status AS ENUM ('active', 'inactive', 'pending');

CREATE TABLE
    users (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        email VARCHAR NOT NULL UNIQUE,
        full_name VARCHAR NOT NULL,
        metadata JSONB,
        status user_status NOT NULL DEFAULT 'pending',
        email_verified_at TIMESTAMP
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
