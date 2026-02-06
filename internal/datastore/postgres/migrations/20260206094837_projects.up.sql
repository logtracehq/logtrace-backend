CREATE TABLE
    IF NOT EXISTS projects (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        organization_id UUID NOT NULL,
        name TEXT NOT NULL,
        type TEXT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at TIMESTAMPTZ,
        CONSTRAINT fk_organization FOREIGN KEY (organization_id) REFERENCES organizations (id) ON DELETE CASCADE
    );
