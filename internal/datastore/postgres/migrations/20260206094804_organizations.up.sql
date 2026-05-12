CREATE TABLE
    organizations (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        name VARCHAR NOT NULL UNIQUE,
        is_active BOOLEAN NOT NULL DEFAULT true,
        plan_name text NOT NULL DEFAULT 'free',
        image_url text,
        is_subscription_active BOOLEAN NOT NULL DEFAULT true,
        subscription_expires_at TIMESTAMP WITH TIME ZONE,
        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at TIMESTAMP WITH TIME ZONE
    );

CREATE INDEX idx_organizations_name ON organizations (name);
CREATE INDEX idx_organizations_is_active ON organizations (is_active);
CREATE INDEX idx_organizations_plan_name ON organizations (plan_name);
CREATE INDEX idx_organizations_subscription_expires_at ON organizations (subscription_expires_at);
