CREATE TABLE
    organizations (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        name VARCHAR NOT NULL UNIQUE,
        is_active BOOLEAN NOT NULL DEFAULT true,
        plan_name text NOT NULL DEFAULT 'free',
        image_url text,
        is_subscription_active BOOLEAN NOT NULL DEFAULT true,
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
