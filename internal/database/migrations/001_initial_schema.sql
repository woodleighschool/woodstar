-- +goose Up
CREATE TYPE user_role AS ENUM ('admin', 'viewer');
CREATE TYPE secret_kind AS ENUM ('orbit');

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    role user_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Owned by alexedwards/scs/pgxstore: token, gob-encoded data, expiry.
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BYTEA NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

CREATE TABLE secrets (
    id BIGSERIAL PRIMARY KEY,
    kind secret_kind NOT NULL,
    value TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX secrets_kind_active_idx
    ON secrets (kind, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE hosts (
    id BIGSERIAL PRIMARY KEY,
    hardware_uuid TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    hostname TEXT NOT NULL DEFAULT '',
    computer_name TEXT NOT NULL DEFAULT '',
    hardware_serial TEXT NOT NULL DEFAULT '',
    hardware_model TEXT NOT NULL DEFAULT '',
    os_version TEXT NOT NULL DEFAULT '',
    osquery_version TEXT NOT NULL DEFAULT '',
    orbit_version TEXT NOT NULL DEFAULT '',
    last_seen_at TIMESTAMPTZ,
    detail_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE hosts;
DROP TABLE secrets;
DROP TABLE sessions;
DROP TABLE users;
DROP TYPE secret_kind;
DROP TYPE user_role;
