-- +goose Up

CREATE TYPE agent AS ENUM ('orbit', 'santa', 'munki');

CREATE TABLE agent_secrets (
    id BIGSERIAL PRIMARY KEY,
    agent agent NOT NULL,
    value TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CHECK (length(btrim(value)) >= 32)
);

CREATE INDEX agent_secrets_active_idx
    ON agent_secrets (agent, created_at DESC)
    WHERE deleted_at IS NULL;
