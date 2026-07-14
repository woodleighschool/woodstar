-- +goose Up

ALTER TABLE hosts
    ADD COLUMN orbit_device_auth_token TEXT NOT NULL DEFAULT '',
    ADD CONSTRAINT hosts_orbit_device_auth_token_format CHECK (
        orbit_device_auth_token = '' OR
        orbit_device_auth_token ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    );

CREATE UNIQUE INDEX hosts_orbit_device_auth_token_idx
    ON hosts (orbit_device_auth_token)
    WHERE orbit_device_auth_token <> '';
