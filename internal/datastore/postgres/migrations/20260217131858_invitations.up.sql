CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'declined', 'expired', 'revoked');

CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    email VARCHAR NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    fullname VARCHAR,
    status invitation_status DEFAULT 'pending',
    role VARCHAR,
    token TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
)

CREATE INDEX idx_invitations_organization_id ON invitations (organization_id);
CREATE INDEX idx_invitations_email ON invitations (email);
