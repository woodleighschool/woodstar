-- +goose Up

CREATE TYPE santa_signing_status AS ENUM (
    'unspecified',
    'unsigned',
    'invalid',
    'adhoc',
    'development',
    'production'
);

ALTER TABLE santa_sync_targets
    ADD COLUMN notification_app_name TEXT NOT NULL DEFAULT '';

ALTER TABLE santa_sync_pending_rules
    ADD COLUMN notification_app_name TEXT NOT NULL DEFAULT '';

ALTER TABLE santa_executables
    ADD COLUMN file_bundle_executable_rel_path TEXT NOT NULL DEFAULT '',
    ADD COLUMN file_bundle_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN file_bundle_version TEXT NOT NULL DEFAULT '',
    ADD COLUMN file_bundle_version_string TEXT NOT NULL DEFAULT '',
    ADD COLUMN file_bundle_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN file_bundle_hash_millis INT NOT NULL DEFAULT 0 CHECK (file_bundle_hash_millis >= 0),
    ADD COLUMN file_bundle_binary_count INT NOT NULL DEFAULT 0 CHECK (file_bundle_binary_count >= 0),
    ADD COLUMN codesigning_flags BIGINT NOT NULL DEFAULT 0 CHECK (codesigning_flags >= 0),
    ADD COLUMN signing_status santa_signing_status NOT NULL DEFAULT 'unspecified',
    ADD COLUMN secure_signing_time TIMESTAMPTZ,
    ADD COLUMN signing_time TIMESTAMPTZ;

CREATE TABLE santa_certificates (
    id BIGSERIAL PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
    common_name TEXT NOT NULL DEFAULT '',
    organization TEXT NOT NULL DEFAULT '',
    organizational_unit TEXT NOT NULL DEFAULT '',
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NULLIF(btrim(sha256), '') IS NOT NULL)
);

CREATE TABLE santa_signing_chain_entries (
    signing_chain_id BIGINT NOT NULL REFERENCES santa_signing_chains (id) ON DELETE CASCADE,
    position INT NOT NULL CHECK (position >= 0),
    certificate_id BIGINT NOT NULL REFERENCES santa_certificates (id) ON DELETE RESTRICT,
    PRIMARY KEY (signing_chain_id, position)
);

CREATE INDEX santa_signing_chain_entries_certificate_idx
    ON santa_signing_chain_entries (certificate_id);

ALTER TABLE santa_signing_chains
    DROP COLUMN entries;

ALTER TABLE santa_execution_events
    ADD COLUMN pid INT NOT NULL DEFAULT 0,
    ADD COLUMN ppid INT NOT NULL DEFAULT 0,
    ADD COLUMN parent_name TEXT NOT NULL DEFAULT '';

CREATE TABLE santa_bundles (
    id BIGSERIAL PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
    bundle_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    executable_rel_path TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    version_string TEXT NOT NULL DEFAULT '',
    binary_count INT NOT NULL DEFAULT 0 CHECK (binary_count >= 0),
    hash_millis INT NOT NULL DEFAULT 0 CHECK (hash_millis >= 0),
    uploaded_at TIMESTAMPTZ,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NULLIF(btrim(sha256), '') IS NOT NULL)
);

CREATE TABLE santa_bundle_executables (
    bundle_id BIGINT NOT NULL REFERENCES santa_bundles (id) ON DELETE CASCADE,
    executable_id BIGINT NOT NULL REFERENCES santa_executables (id) ON DELETE CASCADE,
    PRIMARY KEY (bundle_id, executable_id)
);

CREATE INDEX santa_bundle_executables_executable_idx
    ON santa_bundle_executables (executable_id);

ALTER TABLE santa_file_access_events
    ADD COLUMN primary_process_sha256 TEXT NOT NULL DEFAULT '',
    ADD COLUMN primary_process_path TEXT NOT NULL DEFAULT '',
    ADD COLUMN primary_process_signing_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN primary_process_team_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN primary_process_cdhash TEXT NOT NULL DEFAULT '',
    ADD COLUMN primary_process_pid INT NOT NULL DEFAULT 0;

CREATE INDEX santa_file_access_events_primary_process_sha_idx
    ON santa_file_access_events (primary_process_sha256)
    WHERE primary_process_sha256 <> '';
