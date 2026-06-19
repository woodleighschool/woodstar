-- +goose Up

ALTER TYPE agent ADD VALUE 'munki';

CREATE UNIQUE INDEX hosts_hardware_serial_idx
    ON hosts (hardware_serial)
    WHERE hardware_serial <> '';

CREATE TABLE munki_host_status (
    host_id BIGINT PRIMARY KEY REFERENCES hosts (id) ON DELETE CASCADE,
    version TEXT NOT NULL DEFAULT '',
    manifest_name TEXT NOT NULL DEFAULT '',
    success BOOLEAN NOT NULL DEFAULT FALSE,
    errors TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    warnings TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    problem_installs TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    run_started_at TIMESTAMPTZ,
    run_ended_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE munki_host_items (
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    installed BOOLEAN NOT NULL DEFAULT FALSE,
    installed_version TEXT NOT NULL DEFAULT '',
    run_ended_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (host_id, name)
);

CREATE INDEX munki_host_items_host_idx
    ON munki_host_items (host_id);
