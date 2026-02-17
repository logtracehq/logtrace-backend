CREATE TABLE organization_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    organization_id UUID NOT NULL,
    user_id VARCHAR,
    name VARCHAR,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
