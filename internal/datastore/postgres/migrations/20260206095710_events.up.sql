CREATE TABLE
    events (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        type VARCHAR NOT NULL,
        username VARCHAR,
        user_id VARCHAR,
        organization_id UUID NOT NULL,
        name VARCHAR,
        request_details JSONB DEFAULT '{}'::JSONB,
        metadata JSONB,
        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at TIMESTAMP WITH TIME ZONE
    );

CREATE INDEX idx_events_organization_id ON events (organization_id);
CREATE INDEX idx_events_user_id ON events (user_id);
CREATE INDEX idx_events_type ON events (type);
CREATE INDEX idx_events_created_at ON events (created_at);
