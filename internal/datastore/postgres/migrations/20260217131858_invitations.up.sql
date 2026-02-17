CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    email VARCHAR NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    fullname VARCHAR,
    status VARCHAR DEFAULT 'pending',
    role VARCHAR,
    token TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
)
