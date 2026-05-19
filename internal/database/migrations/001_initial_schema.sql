-- +goose Up

CREATE TYPE user_role AS ENUM ('admin', 'viewer');
CREATE TYPE secret_kind AS ENUM ('orbit');
CREATE TYPE platform AS ENUM ('darwin', 'windows', 'linux');
CREATE TYPE label_scope_mode AS ENUM ('none', 'include_any', 'include_all', 'exclude_any');

-- Users, sessions, secrets ---------------------------------------------------

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    role user_role NOT NULL,
    api_key TEXT,
    api_key_created_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_api_key_idx
    ON users (api_key)
    WHERE api_key IS NOT NULL;

-- Owned by alexedwards/scs/pgxstore.
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

-- Directory (Entra-only MVP; table shape stays portable) ---------------------

CREATE TABLE directory_users (
    id BIGSERIAL PRIMARY KEY,
    external_id TEXT NOT NULL UNIQUE,
    user_principal_name TEXT NOT NULL UNIQUE,
    mail TEXT,
    mail_nickname TEXT,
    display_name TEXT NOT NULL DEFAULT '',
    given_name TEXT,
    family_name TEXT,
    department TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    last_synced_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX directory_users_upn_idx ON directory_users (user_principal_name);
CREATE INDEX directory_users_department_idx ON directory_users (department);

CREATE TABLE directory_groups (
    id BIGSERIAL PRIMARY KEY,
    external_id TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    mail_nickname TEXT,
    last_synced_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE directory_user_groups (
    directory_user_id BIGINT NOT NULL REFERENCES directory_users (id) ON DELETE CASCADE,
    directory_group_id BIGINT NOT NULL REFERENCES directory_groups (id) ON DELETE CASCADE,
    PRIMARY KEY (directory_user_id, directory_group_id)
);

CREATE INDEX directory_user_groups_group_idx ON directory_user_groups (directory_group_id);

-- Hosts ----------------------------------------------------------------------

-- host_directory_user links one host to one directory user (1:1). The link
-- has a source: 'manual' is set by an admin and is sticky; 'mdm_email' is
-- inferred by the reconciler from an Orbit MCX device-mapping email and is
-- overwritten by future manual links and overwrites future mdm_email
-- inferences when the matched directory user changes.

CREATE TABLE hosts (
    id BIGSERIAL PRIMARY KEY,
    hardware_uuid TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    hostname TEXT NOT NULL DEFAULT '',
    computer_name TEXT NOT NULL DEFAULT '',
    hardware_serial TEXT NOT NULL DEFAULT '',
    hardware_model TEXT NOT NULL DEFAULT '',
    hardware_version TEXT NOT NULL DEFAULT '',
    hardware_vendor TEXT NOT NULL DEFAULT '',
    os_name TEXT NOT NULL DEFAULT '',
    os_version TEXT NOT NULL DEFAULT '',
    os_build TEXT NOT NULL DEFAULT '',
    platform TEXT NOT NULL DEFAULT '',
    platform_like TEXT NOT NULL DEFAULT '',
    osquery_version TEXT NOT NULL DEFAULT '',
    orbit_version TEXT NOT NULL DEFAULT '',
    -- Empty string means "no key issued yet"; a partial unique index enforces
    -- uniqueness only on real keys so multiple unenrolled rows can coexist.
    orbit_node_key TEXT NOT NULL DEFAULT '',
    osquery_node_key TEXT NOT NULL DEFAULT '',
    cpu_type TEXT NOT NULL DEFAULT '',
    cpu_subtype TEXT NOT NULL DEFAULT '',
    cpu_brand TEXT NOT NULL DEFAULT '',
    cpu_logical_cores INTEGER NOT NULL DEFAULT 0,
    cpu_physical_cores INTEGER NOT NULL DEFAULT 0,
    physical_memory BIGINT NOT NULL DEFAULT 0,
    kernel_version TEXT NOT NULL DEFAULT '',
    uptime_seconds BIGINT,
    last_restarted_at TIMESTAMPTZ,
    disk_space_available_bytes BIGINT,
    disk_space_total_bytes BIGINT,
    public_ip INET,
    primary_ip INET,
    primary_mac TEXT NOT NULL DEFAULT '',
    distributed_interval INTEGER,
    config_tls_refresh INTEGER,
    detail_query_hash TEXT NOT NULL DEFAULT '',
    enrolled_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    detail_updated_at TIMESTAMPTZ,
    label_updated_at TIMESTAMPTZ,
    software_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX hosts_orbit_node_key_idx
    ON hosts (orbit_node_key)
    WHERE orbit_node_key <> '';
CREATE UNIQUE INDEX hosts_osquery_node_key_idx
    ON hosts (osquery_node_key)
    WHERE osquery_node_key <> '';
CREATE INDEX hosts_platform_idx
    ON hosts (platform)
    WHERE deleted_at IS NULL;
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

CREATE TABLE host_directory_user (
    host_id BIGINT PRIMARY KEY REFERENCES hosts (id) ON DELETE CASCADE,
    directory_user_id BIGINT NOT NULL REFERENCES directory_users (id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source IN ('manual', 'mdm_email')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX host_directory_user_directory_user_idx
    ON host_directory_user (directory_user_id);

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
    description TEXT NOT NULL DEFAULT '',
    query TEXT,
    criteria JSONB,
    label_type TEXT NOT NULL CHECK (label_type IN ('builtin', 'regular')),
    label_membership_type TEXT NOT NULL CHECK (label_membership_type IN ('dynamic', 'manual', 'derived')),
    platform platform,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (label_membership_type = 'dynamic' AND NULLIF(btrim(query), '') IS NOT NULL AND criteria IS NULL)
        OR (label_membership_type = 'manual' AND query IS NULL AND criteria IS NULL)
        OR (label_membership_type = 'derived' AND query IS NULL AND criteria IS NOT NULL)
    )
);

CREATE INDEX labels_label_type_idx ON labels (label_type);
CREATE INDEX labels_label_membership_type_idx ON labels (label_membership_type);
CREATE INDEX labels_platform_idx ON labels (platform);

CREATE TABLE label_membership (
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (label_id, host_id)
);

CREATE INDEX label_membership_host_idx ON label_membership (host_id);

INSERT INTO labels (name, description, query, label_type, label_membership_type, platform)
VALUES
    ('All Hosts', 'Every enrolled host.', NULL, 'builtin', 'manual', NULL),
    ('macOS', 'All macOS hosts', 'select 1 from os_version where platform = ''darwin'';', 'builtin', 'dynamic', 'darwin'),
    ('Windows', 'All Windows hosts', 'select 1 from os_version where platform = ''windows'';', 'builtin', 'dynamic', 'windows'),
    ('Linux', 'All Linux hosts', 'select 1 from os_version where platform <> '''' and platform not in (''darwin'', ''windows'');', 'builtin', 'dynamic', 'linux');

-- Queries / Checks -----------------------------------------------------------

CREATE TABLE queries (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL,
    platform platform,
    min_osquery_version TEXT,
    schedule_interval INTEGER NOT NULL DEFAULT 0,
    label_scope_mode label_scope_mode NOT NULL DEFAULT 'none',
    created_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (schedule_interval >= 0)
);

CREATE INDEX queries_schedule_idx
    ON queries (schedule_interval)
    WHERE schedule_interval > 0;

CREATE TABLE query_results (
    id BIGSERIAL PRIMARY KEY,
    query_id BIGINT NOT NULL REFERENCES queries (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    data JSONB,
    last_fetched TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX query_results_query_last_fetched_idx
    ON query_results (query_id, last_fetched);

CREATE INDEX query_results_query_host_last_fetched_idx
    ON query_results (query_id, host_id, last_fetched);

CREATE TABLE checks (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL,
    platform platform,
    label_scope_mode label_scope_mode NOT NULL DEFAULT 'none',
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

CREATE TABLE query_labels (
    query_id BIGINT NOT NULL REFERENCES queries (id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (query_id, label_id)
);

CREATE INDEX query_labels_label_idx ON query_labels (label_id);

CREATE TABLE check_labels (
    check_id BIGINT NOT NULL REFERENCES checks (id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (check_id, label_id)
);

CREATE INDEX check_labels_label_idx ON check_labels (label_id);

-- +goose Down

DROP TABLE check_labels;
DROP TABLE query_labels;
DROP TABLE check_membership;
DROP TABLE checks;
DROP TABLE query_results;
DROP TABLE queries;
DROP TABLE label_membership;
DROP TABLE labels;
DROP TABLE host_software_installed_paths;
DROP TABLE host_software;
DROP TABLE software;
DROP TABLE software_titles;
DROP TABLE host_batteries;
DROP TABLE host_users;
DROP TABLE host_directory_user;
DROP TABLE host_emails;
DROP TABLE hosts;
DROP TABLE directory_user_groups;
DROP TABLE directory_groups;
DROP TABLE directory_users;
DROP TABLE secrets;
DROP TABLE sessions;
DROP TABLE users;
DROP TYPE secret_kind;
DROP TYPE platform;
DROP TYPE label_scope_mode;
DROP TYPE user_role;
