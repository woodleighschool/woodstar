-- +goose Up

CREATE TYPE santa_client_mode AS ENUM ('unknown', 'monitor', 'lockdown', 'standalone');
CREATE TYPE santa_removable_media_action AS ENUM ('allow', 'block', 'remount');
CREATE TYPE santa_rule_type AS ENUM ('binary', 'certificate', 'teamid', 'signingid', 'cdhash', 'bundle');
CREATE TYPE santa_policy AS ENUM (
    'allowlist',
    'allowlist_compiler',
    'blocklist',
    'silent_blocklist',
    'silent_gui_blocklist',
    'silent_tty_blocklist',
    'cel'
);
CREATE TYPE santa_execution_decision AS ENUM (
    'unknown',
    'allow_unknown',
    'allow_binary',
    'allow_certificate',
    'allow_scope',
    'allow_teamid',
    'allow_signingid',
    'allow_cdhash',
    'block_unknown',
    'block_binary',
    'block_certificate',
    'block_scope',
    'block_teamid',
    'block_signingid',
    'block_cdhash',
    'bundle_binary',
    'block_binary_mismatch',
    'allow_platform'
);

-- Santa host observation
CREATE TABLE santa_hosts (
    host_id BIGINT PRIMARY KEY REFERENCES hosts (id) ON DELETE CASCADE,
    machine_id TEXT NOT NULL,
    serial_number TEXT NOT NULL,
    santa_version TEXT NOT NULL DEFAULT '',
    client_mode_reported santa_client_mode NOT NULL DEFAULT 'unknown',
    primary_user TEXT NOT NULL DEFAULT '',
    primary_user_groups TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    sip_status SMALLINT,
    last_seen_at TIMESTAMPTZ,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX santa_hosts_machine_id_idx ON santa_hosts (machine_id);
CREATE INDEX santa_hosts_serial_number_idx ON santa_hosts (serial_number);

CREATE TABLE santa_sync_state (
    host_id BIGINT PRIMARY KEY REFERENCES hosts (id) ON DELETE CASCADE,
    pending_full_sync BOOLEAN NOT NULL DEFAULT FALSE,
    pending_payload_rule_count INT NOT NULL DEFAULT 0 CHECK (pending_payload_rule_count >= 0),
    pending_preflight_at TIMESTAMPTZ,
    desired_binary_rule_count INT NOT NULL DEFAULT 0 CHECK (desired_binary_rule_count >= 0),
    desired_certificate_rule_count INT NOT NULL DEFAULT 0 CHECK (desired_certificate_rule_count >= 0),
    desired_teamid_rule_count INT NOT NULL DEFAULT 0 CHECK (desired_teamid_rule_count >= 0),
    desired_signingid_rule_count INT NOT NULL DEFAULT 0 CHECK (desired_signingid_rule_count >= 0),
    desired_cdhash_rule_count INT NOT NULL DEFAULT 0 CHECK (desired_cdhash_rule_count >= 0),
    desired_compiler_rule_count INT NOT NULL DEFAULT 0 CHECK (desired_compiler_rule_count >= 0),
    binary_rule_count INT NOT NULL DEFAULT 0 CHECK (binary_rule_count >= 0),
    certificate_rule_count INT NOT NULL DEFAULT 0 CHECK (certificate_rule_count >= 0),
    teamid_rule_count INT NOT NULL DEFAULT 0 CHECK (teamid_rule_count >= 0),
    signingid_rule_count INT NOT NULL DEFAULT 0 CHECK (signingid_rule_count >= 0),
    cdhash_rule_count INT NOT NULL DEFAULT 0 CHECK (cdhash_rule_count >= 0),
    compiler_rule_count INT NOT NULL DEFAULT 0 CHECK (compiler_rule_count >= 0),
    transitive_rule_count INT NOT NULL DEFAULT 0 CHECK (transitive_rule_count >= 0),
    rules_received INT NOT NULL DEFAULT 0 CHECK (rules_received >= 0),
    rules_processed INT NOT NULL DEFAULT 0 CHECK (rules_processed >= 0),
    last_rule_sync_attempt_at TIMESTAMPTZ,
    last_rule_sync_success_at TIMESTAMPTZ,
    last_clean_sync_at TIMESTAMPTZ,
    last_reported_counts_match_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TYPE santa_sync_target_phase AS ENUM ('desired', 'applied');

CREATE TABLE santa_sync_targets (
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    phase santa_sync_target_phase NOT NULL,
    position INT NOT NULL CHECK (position >= 0),
    rule_type santa_rule_type NOT NULL,
    identifier TEXT NOT NULL,
    policy santa_policy NOT NULL,
    cel_expression TEXT NOT NULL DEFAULT '',
    custom_message TEXT NOT NULL DEFAULT '',
    custom_url TEXT NOT NULL DEFAULT '',
    payload_hash TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (host_id, phase, position),
    UNIQUE (host_id, phase, payload_hash),
    CHECK (NULLIF(btrim(identifier), '') IS NOT NULL),
    CHECK (NULLIF(btrim(payload_hash), '') IS NOT NULL)
);

CREATE INDEX santa_sync_targets_host_phase_idx
    ON santa_sync_targets (host_id, phase);

-- Santa configurations
CREATE TABLE santa_configurations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    position INT NOT NULL UNIQUE,
    client_mode santa_client_mode NOT NULL,
    enable_bundles BOOLEAN NOT NULL,
    enable_transitive_rules BOOLEAN NOT NULL,
    enable_all_event_upload BOOLEAN NOT NULL,
    full_sync_interval_seconds INT NOT NULL CHECK (full_sync_interval_seconds >= 60),
    batch_size INT NOT NULL CHECK (batch_size BETWEEN 5 AND 100),
    allowed_path_regex TEXT NOT NULL,
    blocked_path_regex TEXT NOT NULL,
    removable_media_action santa_removable_media_action,
    removable_media_remount_flags TEXT[],
    encrypted_removable_media_action santa_removable_media_action,
    encrypted_removable_media_remount_flags TEXT[],
    event_detail_url TEXT NOT NULL,
    event_detail_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        removable_media_action <> 'remount'
        OR COALESCE(cardinality(removable_media_remount_flags), 0) > 0
    ),
    CHECK (
        encrypted_removable_media_action <> 'remount'
        OR COALESCE(cardinality(encrypted_removable_media_remount_flags), 0) > 0
    )
);

CREATE TABLE santa_configuration_targets (
    configuration_id BIGINT NOT NULL REFERENCES santa_configurations (id) ON DELETE CASCADE,
    direction target_direction NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE RESTRICT,
    PRIMARY KEY (configuration_id, direction, position),
    UNIQUE (configuration_id, label_id)
);

CREATE INDEX santa_configuration_targets_label_idx
    ON santa_configuration_targets (label_id);

-- Santa rules
CREATE TABLE santa_rules (
    id BIGSERIAL PRIMARY KEY,
    rule_type santa_rule_type NOT NULL,
    identifier TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    custom_message TEXT NOT NULL DEFAULT '',
    custom_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (rule_type, identifier),
    CHECK (NULLIF(btrim(identifier), '') IS NOT NULL)
);

CREATE TABLE santa_rule_targets (
    rule_id BIGINT NOT NULL REFERENCES santa_rules (id) ON DELETE CASCADE,
    direction target_direction NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE RESTRICT,
    policy santa_policy,
    cel_expression TEXT,
    PRIMARY KEY (rule_id, direction, position),
    UNIQUE (rule_id, label_id),
    CHECK (
        (direction = 'include' AND policy IS NOT NULL)
        OR (direction = 'exclude' AND policy IS NULL AND cel_expression IS NULL)
    ),
    CHECK (
        direction <> 'include'
        OR
        (policy = 'cel' AND NULLIF(btrim(COALESCE(cel_expression, '')), '') IS NOT NULL)
        OR (policy <> 'cel' AND NULLIF(btrim(COALESCE(cel_expression, '')), '') IS NULL)
    )
);

CREATE INDEX santa_rule_targets_label_idx
    ON santa_rule_targets (label_id);

-- Santa execution events
CREATE TABLE santa_executables (
    id BIGSERIAL PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
    file_name TEXT NOT NULL DEFAULT '',
    file_bundle_id TEXT NOT NULL DEFAULT '',
    file_bundle_path TEXT NOT NULL DEFAULT '',
    signing_id TEXT NOT NULL DEFAULT '',
    team_id TEXT NOT NULL DEFAULT '',
    cdhash TEXT NOT NULL DEFAULT '',
    entitlements JSONB CHECK (entitlements IS NULL OR jsonb_typeof(entitlements) = 'object'),
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NULLIF(btrim(sha256), '') IS NOT NULL)
);

CREATE TABLE santa_signing_chains (
    id BIGSERIAL PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
    entries JSONB NOT NULL CHECK (jsonb_typeof(entries) = 'array'),
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NULLIF(btrim(sha256), '') IS NOT NULL)
);

CREATE TABLE santa_executable_signing_chains (
    executable_id BIGINT NOT NULL REFERENCES santa_executables (id) ON DELETE CASCADE,
    signing_chain_id BIGINT NOT NULL REFERENCES santa_signing_chains (id) ON DELETE CASCADE,
    PRIMARY KEY (executable_id, signing_chain_id)
);

CREATE INDEX santa_executable_signing_chains_chain_idx
    ON santa_executable_signing_chains (signing_chain_id);

CREATE TABLE santa_execution_events (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    executable_id BIGINT NOT NULL REFERENCES santa_executables (id) ON DELETE RESTRICT,
    file_path TEXT NOT NULL DEFAULT '',
    executing_user TEXT NOT NULL DEFAULT '',
    logged_in_users TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    current_sessions TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    decision santa_execution_decision NOT NULL,
    occurred_at TIMESTAMPTZ,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX santa_execution_events_host_time_idx
    ON santa_execution_events (host_id, COALESCE(occurred_at, ingested_at) DESC);
CREATE INDEX santa_execution_events_decision_ingested_idx
    ON santa_execution_events (decision, ingested_at DESC);
