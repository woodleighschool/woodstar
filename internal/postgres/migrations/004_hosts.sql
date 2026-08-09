-- +goose Up

CREATE TYPE host_primary_user_source AS ENUM ('manual', 'orbit_profile');

CREATE TABLE hosts (
    id BIGSERIAL PRIMARY KEY,
    hardware_uuid TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    hostname TEXT NOT NULL DEFAULT '',
    computer_name TEXT NOT NULL DEFAULT '',
    hardware_serial TEXT NOT NULL DEFAULT '',
    hardware_model_identifier TEXT NOT NULL DEFAULT '',
    hardware_vendor TEXT NOT NULL DEFAULT '',
    os_name TEXT NOT NULL DEFAULT '',
    os_version TEXT NOT NULL DEFAULT '',
    os_build TEXT NOT NULL DEFAULT '',
    os_platform TEXT NOT NULL DEFAULT '',
    osquery_version TEXT NOT NULL DEFAULT '',
    orbit_version TEXT NOT NULL DEFAULT '',
    orbit_node_key TEXT NOT NULL DEFAULT '',
    orbit_device_auth_token TEXT NOT NULL DEFAULT '',
    osquery_node_key TEXT NOT NULL DEFAULT '',
    enrollment_agent TEXT NOT NULL DEFAULT '',
    cpu_type TEXT NOT NULL DEFAULT '',
    cpu_subtype TEXT NOT NULL DEFAULT '',
    cpu_brand TEXT NOT NULL DEFAULT '',
    cpu_logical_cores INTEGER NOT NULL DEFAULT 0,
    cpu_physical_cores INTEGER NOT NULL DEFAULT 0,
    memory_bytes BIGINT NOT NULL DEFAULT 0,
    os_kernel_version TEXT NOT NULL DEFAULT '',
    last_restarted_at TIMESTAMPTZ,
    boot_volume_available_bytes BIGINT,
    boot_volume_total_bytes BIGINT,
    last_remote_ip INET,
    primary_ip INET,
    primary_mac TEXT NOT NULL DEFAULT '',
    osquery_distributed_interval_seconds INTEGER,
    osquery_config_refresh_seconds INTEGER,
    inventory_query_hash TEXT NOT NULL DEFAULT '',
    enrolled_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    inventory_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT hosts_orbit_device_auth_token_format CHECK (
        orbit_device_auth_token = ''
        OR orbit_device_auth_token ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    )
);

CREATE UNIQUE INDEX hosts_orbit_node_key_idx
    ON hosts (orbit_node_key)
    WHERE orbit_node_key <> '';
CREATE UNIQUE INDEX hosts_orbit_device_auth_token_idx
    ON hosts (orbit_device_auth_token)
    WHERE orbit_device_auth_token <> '';
CREATE UNIQUE INDEX hosts_osquery_node_key_idx
    ON hosts (osquery_node_key)
    WHERE osquery_node_key <> '';
CREATE UNIQUE INDEX hosts_hardware_serial_idx
    ON hosts (hardware_serial)
    WHERE hardware_serial <> '';
CREATE INDEX hosts_active_seen_idx
    ON hosts (last_seen_at DESC NULLS LAST);
CREATE INDEX hosts_inventory_stale_idx
    ON hosts (inventory_updated_at NULLS FIRST);

CREATE TABLE host_primary_user_sources (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    source host_primary_user_source NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (host_id, source)
);

CREATE INDEX host_primary_user_sources_host_idx ON host_primary_user_sources (host_id);
CREATE INDEX host_primary_user_sources_email_idx ON host_primary_user_sources (email);
