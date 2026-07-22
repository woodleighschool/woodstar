-- +goose Up

CREATE TYPE user_role AS ENUM ('admin', 'viewer');
CREATE TYPE directory_source AS ENUM ('local', 'entra');

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    password_hash TEXT,
    role user_role,
    api_key TEXT,
    api_key_created_at TIMESTAMPTZ,
    source directory_source NOT NULL DEFAULT 'local',
    external_id TEXT,
    user_principal_name TEXT UNIQUE,
    mail_nickname TEXT,
    given_name TEXT,
    family_name TEXT,
    department TEXT,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CHECK (
        (source = 'local' AND external_id IS NULL)
        OR (source <> 'local' AND external_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX users_api_key_idx
    ON users (api_key)
    WHERE api_key IS NOT NULL;
CREATE INDEX users_department_idx
    ON users (department)
    WHERE department IS NOT NULL;
CREATE INDEX users_lower_email_idx ON users (lower(email));

-- Owned by alexedwards/scs/pgxstore.
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BYTEA NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);
