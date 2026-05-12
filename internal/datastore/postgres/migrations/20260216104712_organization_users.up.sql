CREATE TABLE organization_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    organization_id UUID NOT NULL,
    user_id VARCHAR,
    username VARCHAR,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_organization_users_organization_id ON organization_users (organization_id);
CREATE INDEX idx_organization_users_user_id ON organization_users (user_id);
