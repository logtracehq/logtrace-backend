CREATE TYPE user_role AS ENUM ('admin', 'editor', 'viewer');

CREATE TABLE
    user_roles (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        role_name user_role NOT NULL,
        assigned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
