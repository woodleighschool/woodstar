-- +goose Up
ALTER TABLE hosts
    ADD COLUMN orbit_node_key     TEXT UNIQUE,
    ADD COLUMN osquery_node_key   TEXT UNIQUE,
    ADD COLUMN platform           TEXT NOT NULL DEFAULT '',
    ADD COLUMN platform_like      TEXT NOT NULL DEFAULT '',
    ADD COLUMN enrolled_at        TIMESTAMPTZ,
    ADD COLUMN cpu_brand          TEXT NOT NULL DEFAULT '',
    ADD COLUMN cpu_logical_cores  INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN cpu_physical_cores INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN physical_memory    BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN hardware_vendor    TEXT NOT NULL DEFAULT '',
    ADD COLUMN kernel_version     TEXT NOT NULL DEFAULT '';

CREATE INDEX hosts_active_seen_idx
    ON hosts (last_seen_at DESC NULLS LAST)
    WHERE deleted_at IS NULL;

CREATE INDEX hosts_detail_stale_idx
    ON hosts (detail_updated_at NULLS FIRST)
    WHERE deleted_at IS NULL;

CREATE TABLE host_emails (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (host_id, source)
);

CREATE INDEX host_emails_host_idx ON host_emails (host_id);
CREATE INDEX host_emails_email_idx ON host_emails (email);

CREATE TABLE software (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    bundle_identifier TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (name, version, source, bundle_identifier)
);

CREATE INDEX software_name_idx ON software (name);

CREATE TABLE host_software (
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    software_id BIGINT NOT NULL REFERENCES software (id) ON DELETE CASCADE,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_opened_at TIMESTAMPTZ,
    PRIMARY KEY (host_id, software_id)
);

CREATE INDEX host_software_software_idx ON host_software (software_id);

-- +goose Down
DROP TABLE host_software;
DROP TABLE software;
DROP TABLE host_emails;
DROP INDEX hosts_detail_stale_idx;
DROP INDEX hosts_active_seen_idx;
ALTER TABLE hosts
    DROP COLUMN kernel_version,
    DROP COLUMN hardware_vendor,
    DROP COLUMN physical_memory,
    DROP COLUMN cpu_physical_cores,
    DROP COLUMN cpu_logical_cores,
    DROP COLUMN cpu_brand,
    DROP COLUMN enrolled_at,
    DROP COLUMN platform_like,
    DROP COLUMN platform,
    DROP COLUMN osquery_node_key,
    DROP COLUMN orbit_node_key;
