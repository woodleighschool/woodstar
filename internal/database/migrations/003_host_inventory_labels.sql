-- +goose Up
ALTER TABLE hosts
    ADD COLUMN os_name                         TEXT NOT NULL DEFAULT '',
    ADD COLUMN os_build                        TEXT NOT NULL DEFAULT '',
    ADD COLUMN hardware_version                TEXT NOT NULL DEFAULT '',
    ADD COLUMN cpu_type                        TEXT NOT NULL DEFAULT '',
    ADD COLUMN cpu_subtype                     TEXT NOT NULL DEFAULT '',
    ADD COLUMN uptime_seconds                  BIGINT,
    ADD COLUMN last_restarted_at               TIMESTAMPTZ,
    ADD COLUMN disk_space_available_bytes      BIGINT,
    ADD COLUMN disk_space_total_bytes          BIGINT,
    ADD COLUMN public_ip                       INET,
    ADD COLUMN primary_ip                      INET,
    ADD COLUMN primary_mac                     TEXT NOT NULL DEFAULT '',
    ADD COLUMN distributed_interval            INTEGER,
    ADD COLUMN config_tls_refresh              INTEGER,
    ADD COLUMN detail_query_hash               TEXT NOT NULL DEFAULT '',
    ADD COLUMN label_updated_at                TIMESTAMPTZ,
    ADD COLUMN software_updated_at             TIMESTAMPTZ;

CREATE INDEX hosts_platform_idx
    ON hosts (platform)
    WHERE deleted_at IS NULL;

CREATE TABLE host_users (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    uid TEXT NOT NULL,
    username TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    directory TEXT NOT NULL DEFAULT '',
    shell TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (host_id, uid, username)
);

CREATE INDEX host_users_host_idx ON host_users (host_id);

CREATE TABLE host_batteries (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    serial_number TEXT NOT NULL,
    manufacturer TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    chemistry TEXT NOT NULL DEFAULT '',
    cycle_count INTEGER,
    health TEXT NOT NULL DEFAULT '',
    designed_capacity INTEGER,
    max_capacity INTEGER,
    current_capacity INTEGER,
    percent_remaining DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (host_id, serial_number)
);

CREATE INDEX host_batteries_host_idx ON host_batteries (host_id);

CREATE TABLE labels (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    query TEXT,
    kind TEXT NOT NULL CHECK (kind IN ('builtin', 'custom')),
    membership_type TEXT NOT NULL CHECK (membership_type IN ('dynamic', 'static', 'identity')),
    platform TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (membership_type = 'dynamic' AND NULLIF(btrim(query), '') IS NOT NULL)
        OR (membership_type IN ('static', 'identity') AND query IS NULL)
    )
);

CREATE INDEX labels_kind_idx ON labels (kind);
CREATE INDEX labels_membership_type_idx ON labels (membership_type);
CREATE INDEX labels_platform_idx ON labels (platform);

CREATE TABLE label_membership (
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (label_id, host_id)
);

CREATE INDEX label_membership_host_idx ON label_membership (host_id);

INSERT INTO labels (name, description, query, kind, membership_type, platform)
VALUES
    ('All Hosts', 'Every enrolled host.', 'select 1;', 'builtin', 'dynamic', NULL),
    ('macOS', 'Hosts reporting macOS.', 'select 1 from os_version where platform = ''darwin'';', 'builtin', 'dynamic', 'darwin'),
    ('macOS 14+', 'Hosts reporting macOS 14 or newer.', 'select 1 from os_version where platform = ''darwin'' and major >= 14;', 'builtin', 'dynamic', 'darwin');

-- +goose Down
DELETE FROM labels WHERE kind = 'builtin' AND name IN ('All Hosts', 'macOS', 'macOS 14+');
DROP TABLE label_membership;
DROP TABLE labels;
DROP TABLE host_batteries;
DROP TABLE host_users;
DROP INDEX hosts_platform_idx;
ALTER TABLE hosts
    DROP COLUMN software_updated_at,
    DROP COLUMN label_updated_at,
    DROP COLUMN detail_query_hash,
    DROP COLUMN config_tls_refresh,
    DROP COLUMN distributed_interval,
    DROP COLUMN primary_mac,
    DROP COLUMN primary_ip,
    DROP COLUMN public_ip,
    DROP COLUMN disk_space_total_bytes,
    DROP COLUMN disk_space_available_bytes,
    DROP COLUMN last_restarted_at,
    DROP COLUMN uptime_seconds,
    DROP COLUMN cpu_subtype,
    DROP COLUMN cpu_type,
    DROP COLUMN hardware_version,
    DROP COLUMN os_build,
    DROP COLUMN os_name;
