CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE
    plans (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
        name VARCHAR NOT NULL UNIQUE,
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
