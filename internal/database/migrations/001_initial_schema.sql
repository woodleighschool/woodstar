-- +goose Up

CREATE TYPE user_role AS ENUM ('admin', 'viewer');
CREATE TYPE directory_source AS ENUM ('local', 'entra');
CREATE TYPE agent AS ENUM ('orbit', 'santa');
CREATE TYPE host_primary_user_source AS ENUM ('manual', 'orbit_profile');
CREATE TYPE target_direction AS ENUM ('include', 'exclude');

-- Users, sessions, Enrollment ------------------------------------------

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
CREATE INDEX users_lower_upn_idx ON users (lower(user_principal_name))
    WHERE user_principal_name IS NOT NULL;

-- Owned by alexedwards/scs/pgxstore.
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BYTEA NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

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

-- Directory groups -----------------------------------------------------------

CREATE TABLE directory_groups (
    id BIGSERIAL PRIMARY KEY,
    source directory_source NOT NULL,
    external_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    mail_nickname TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CHECK (source <> 'local')
);

CREATE TABLE directory_group_memberships (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES directory_groups (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX directory_group_memberships_group_idx ON directory_group_memberships (group_id);

-- Hosts ----------------------------------------------------------------------

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
    -- Empty string means "no key issued yet"; a partial unique index enforces
    -- uniqueness only on real keys so multiple unenrolled rows can coexist.
    orbit_node_key TEXT NOT NULL DEFAULT '',
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX hosts_orbit_node_key_idx
    ON hosts (orbit_node_key)
    WHERE orbit_node_key <> '';
CREATE UNIQUE INDEX hosts_osquery_node_key_idx
    ON hosts (osquery_node_key)
    WHERE osquery_node_key <> '';
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

CREATE TABLE host_certificates (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    sha1 TEXT NOT NULL,
    common_name TEXT NOT NULL DEFAULT '',
    subject_country TEXT NOT NULL DEFAULT '',
    subject_organization TEXT NOT NULL DEFAULT '',
    subject_organizational_unit TEXT NOT NULL DEFAULT '',
    subject_common_name TEXT NOT NULL DEFAULT '',
    issuer_country TEXT NOT NULL DEFAULT '',
    issuer_organization TEXT NOT NULL DEFAULT '',
    issuer_organizational_unit TEXT NOT NULL DEFAULT '',
    issuer_common_name TEXT NOT NULL DEFAULT '',
    key_algorithm TEXT NOT NULL DEFAULT '',
    key_strength INTEGER,
    key_usage TEXT NOT NULL DEFAULT '',
    signing_algorithm TEXT NOT NULL DEFAULT '',
    not_valid_after TIMESTAMPTZ,
    not_valid_before TIMESTAMPTZ,
    serial TEXT NOT NULL DEFAULT '',
    certificate_authority BOOLEAN NOT NULL DEFAULT FALSE,
    source TEXT NOT NULL DEFAULT '',
    username TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (host_id, sha1, source, username)
);

CREATE INDEX host_certificates_host_idx ON host_certificates (host_id);
CREATE INDEX host_certificates_sha1_idx ON host_certificates (sha1);

-- Software -------------------------------------------------------------------

CREATE TABLE software_titles (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    extension_for TEXT NOT NULL DEFAULT '',
    bundle_identifier TEXT NOT NULL DEFAULT '',
    vendor TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (name, source, extension_for, bundle_identifier)
);

CREATE UNIQUE INDEX software_titles_bundle_idx
    ON software_titles (bundle_identifier, source, extension_for)
    WHERE bundle_identifier <> '';

CREATE TABLE software (
    id BIGSERIAL PRIMARY KEY,
    title_id BIGINT NOT NULL REFERENCES software_titles (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    bundle_identifier TEXT NOT NULL DEFAULT '',
    extension_id TEXT NOT NULL DEFAULT '',
    extension_for TEXT NOT NULL DEFAULT '',
    vendor TEXT NOT NULL DEFAULT '',
    arch TEXT NOT NULL DEFAULT '',
    release TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (
        title_id,
        version,
        source,
        bundle_identifier,
        extension_id,
        extension_for,
        vendor,
        arch,
        release
    )
);

CREATE INDEX software_name_idx ON software (name);
CREATE INDEX software_title_idx ON software (title_id);

CREATE TABLE host_software (
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    software_id BIGINT NOT NULL REFERENCES software (id) ON DELETE CASCADE,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_opened_at TIMESTAMPTZ,
    PRIMARY KEY (host_id, software_id)
);

CREATE INDEX host_software_software_idx ON host_software (software_id);

CREATE TABLE host_software_installed_paths (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    software_id BIGINT NOT NULL REFERENCES software (id) ON DELETE CASCADE,
    installed_path TEXT NOT NULL,
    team_identifier TEXT NOT NULL DEFAULT '',
    cdhash_sha256 TEXT,
    executable_sha256 TEXT,
    executable_path TEXT,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (host_id, software_id, installed_path)
);

CREATE INDEX host_software_installed_paths_host_software_idx
    ON host_software_installed_paths (host_id, software_id);

-- Labels ---------------------------------------------------------------------
-- Labels are first-class targeting primitives. label_type separates system
-- labels from admin-created labels; label_membership_type is how membership is produced:
--   dynamic - osquery query result drives membership
--   manual  - membership is written by the server (e.g. All Hosts on enroll)
--   derived - membership is computed from non-osquery host attributes (criteria JSON)

CREATE TABLE labels (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    builtin_key TEXT,
    description TEXT NOT NULL DEFAULT '',
    query TEXT,
    criteria JSONB,
    label_type TEXT NOT NULL CHECK (label_type IN ('builtin', 'regular')),
    label_membership_type TEXT NOT NULL CHECK (label_membership_type IN ('dynamic', 'manual', 'derived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (label_membership_type = 'dynamic' AND NULLIF(btrim(query), '') IS NOT NULL AND criteria IS NULL)
        OR (label_membership_type = 'manual' AND query IS NULL AND criteria IS NULL)
        OR (label_membership_type = 'derived' AND query IS NULL AND criteria IS NOT NULL)
    ),
    CHECK (
        (label_type = 'builtin' AND builtin_key IS NOT NULL AND builtin_key IN ('all-hosts'))
        OR (label_type = 'regular' AND builtin_key IS NULL)
    )
);

CREATE INDEX labels_label_type_idx ON labels (label_type);
CREATE INDEX labels_label_membership_type_idx ON labels (label_membership_type);
CREATE UNIQUE INDEX labels_builtin_key_unique_idx ON labels (builtin_key) WHERE builtin_key IS NOT NULL;

CREATE TABLE label_membership (
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (label_id, host_id)
);

CREATE INDEX label_membership_host_idx ON label_membership (host_id);

INSERT INTO labels (name, builtin_key, description, query, label_type, label_membership_type)
VALUES
    ('All Hosts', 'all-hosts', 'Every enrolled host.', NULL, 'builtin', 'manual');

-- Reports / Checks -----------------------------------------------------------

CREATE TABLE reports (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL,
    min_osquery_version TEXT,
    schedule_interval INTEGER NOT NULL DEFAULT 0,
    created_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (schedule_interval >= 0)
);

CREATE INDEX reports_schedule_idx
    ON reports (schedule_interval)
    WHERE schedule_interval > 0;

CREATE TABLE report_results (
    id BIGSERIAL PRIMARY KEY,
    report_id BIGINT NOT NULL REFERENCES reports (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    data JSONB,
    last_fetched TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX report_results_report_last_fetched_idx
    ON report_results (report_id, last_fetched);

CREATE INDEX report_results_report_host_last_fetched_idx
    ON report_results (report_id, host_id, last_fetched);

CREATE TABLE checks (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL,
    created_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE check_membership (
    check_id BIGINT NOT NULL REFERENCES checks (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    passes BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (check_id, host_id)
);

CREATE INDEX check_membership_passes_idx
    ON check_membership (check_id, passes);

CREATE TABLE osquery_report_targets (
    report_id BIGINT NOT NULL REFERENCES reports (id) ON DELETE CASCADE,
    direction target_direction NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE RESTRICT,
    PRIMARY KEY (report_id, direction, position),
    UNIQUE (report_id, label_id)
);

CREATE INDEX osquery_report_targets_label_idx ON osquery_report_targets (label_id);

CREATE TABLE osquery_check_targets (
    check_id BIGINT NOT NULL REFERENCES checks (id) ON DELETE CASCADE,
    direction target_direction NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE RESTRICT,
    PRIMARY KEY (check_id, direction, position),
    UNIQUE (check_id, label_id)
);

CREATE INDEX osquery_check_targets_label_idx ON osquery_check_targets (label_id);
