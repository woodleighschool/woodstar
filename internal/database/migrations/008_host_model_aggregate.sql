-- +goose Up

ALTER TABLE hosts
    ADD COLUMN IF NOT EXISTS os_platform TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS enrollment_agent TEXT NOT NULL DEFAULT '';

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'hardware_model')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'hardware_model_identifier') THEN
        ALTER TABLE hosts RENAME COLUMN hardware_model TO hardware_model_identifier;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'kernel_version')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'os_kernel_version') THEN
        ALTER TABLE hosts RENAME COLUMN kernel_version TO os_kernel_version;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'physical_memory')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'memory_bytes') THEN
        ALTER TABLE hosts RENAME COLUMN physical_memory TO memory_bytes;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'disk_space_available_bytes')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'boot_volume_available_bytes') THEN
        ALTER TABLE hosts RENAME COLUMN disk_space_available_bytes TO boot_volume_available_bytes;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'disk_space_total_bytes')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'boot_volume_total_bytes') THEN
        ALTER TABLE hosts RENAME COLUMN disk_space_total_bytes TO boot_volume_total_bytes;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'public_ip')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'last_remote_ip') THEN
        ALTER TABLE hosts RENAME COLUMN public_ip TO last_remote_ip;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'distributed_interval')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'osquery_distributed_interval_seconds') THEN
        ALTER TABLE hosts RENAME COLUMN distributed_interval TO osquery_distributed_interval_seconds;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'config_tls_refresh')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'osquery_config_refresh_seconds') THEN
        ALTER TABLE hosts RENAME COLUMN config_tls_refresh TO osquery_config_refresh_seconds;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'detail_query_hash')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'inventory_query_hash') THEN
        ALTER TABLE hosts RENAME COLUMN detail_query_hash TO inventory_query_hash;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'detail_updated_at')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'inventory_updated_at') THEN
        ALTER TABLE hosts RENAME COLUMN detail_updated_at TO inventory_updated_at;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'host_emails')
        AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'host_user_affinity_mappings') THEN
        ALTER TABLE host_emails RENAME TO host_user_affinity_mappings;
    END IF;
END
$$;
-- +goose StatementEnd

UPDATE hosts
SET enrollment_agent = CASE
    WHEN orbit_node_key <> '' THEN 'orbit'
    WHEN osquery_node_key <> '' THEN 'osquery'
    ELSE enrollment_agent
END
WHERE enrollment_agent = '';

DROP INDEX IF EXISTS hosts_detail_stale_idx;
CREATE INDEX IF NOT EXISTS hosts_inventory_stale_idx
    ON hosts (inventory_updated_at NULLS FIRST);

DROP INDEX IF EXISTS host_emails_host_idx;
DROP INDEX IF EXISTS host_emails_email_idx;
CREATE INDEX IF NOT EXISTS host_user_affinity_mappings_host_idx
    ON host_user_affinity_mappings (host_id);
CREATE INDEX IF NOT EXISTS host_user_affinity_mappings_email_idx
    ON host_user_affinity_mappings (email);

ALTER TABLE hosts
    DROP COLUMN IF EXISTS hardware_version,
    DROP COLUMN IF EXISTS label_updated_at,
    DROP COLUMN IF EXISTS software_updated_at;

-- +goose Down

ALTER TABLE hosts
    ADD COLUMN IF NOT EXISTS hardware_version TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS label_updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS software_updated_at TIMESTAMPTZ;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'hardware_model_identifier')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'hardware_model') THEN
        ALTER TABLE hosts RENAME COLUMN hardware_model_identifier TO hardware_model;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'os_kernel_version')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'kernel_version') THEN
        ALTER TABLE hosts RENAME COLUMN os_kernel_version TO kernel_version;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'memory_bytes')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'physical_memory') THEN
        ALTER TABLE hosts RENAME COLUMN memory_bytes TO physical_memory;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'boot_volume_available_bytes')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'disk_space_available_bytes') THEN
        ALTER TABLE hosts RENAME COLUMN boot_volume_available_bytes TO disk_space_available_bytes;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'boot_volume_total_bytes')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'disk_space_total_bytes') THEN
        ALTER TABLE hosts RENAME COLUMN boot_volume_total_bytes TO disk_space_total_bytes;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'last_remote_ip')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'public_ip') THEN
        ALTER TABLE hosts RENAME COLUMN last_remote_ip TO public_ip;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'osquery_distributed_interval_seconds')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'distributed_interval') THEN
        ALTER TABLE hosts RENAME COLUMN osquery_distributed_interval_seconds TO distributed_interval;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'osquery_config_refresh_seconds')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'config_tls_refresh') THEN
        ALTER TABLE hosts RENAME COLUMN osquery_config_refresh_seconds TO config_tls_refresh;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'inventory_query_hash')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'detail_query_hash') THEN
        ALTER TABLE hosts RENAME COLUMN inventory_query_hash TO detail_query_hash;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'inventory_updated_at')
        AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'hosts' AND column_name = 'detail_updated_at') THEN
        ALTER TABLE hosts RENAME COLUMN inventory_updated_at TO detail_updated_at;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'host_user_affinity_mappings')
        AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'host_emails') THEN
        ALTER TABLE host_user_affinity_mappings RENAME TO host_emails;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE hosts
    DROP COLUMN IF EXISTS os_platform,
    DROP COLUMN IF EXISTS enrollment_agent;

DROP INDEX IF EXISTS hosts_inventory_stale_idx;
CREATE INDEX IF NOT EXISTS hosts_detail_stale_idx
    ON hosts (detail_updated_at NULLS FIRST);

DROP INDEX IF EXISTS host_user_affinity_mappings_host_idx;
DROP INDEX IF EXISTS host_user_affinity_mappings_email_idx;
CREATE INDEX IF NOT EXISTS host_emails_host_idx ON host_emails (host_id);
CREATE INDEX IF NOT EXISTS host_emails_email_idx ON host_emails (email);
