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
CREATE TYPE santa_file_access_decision AS ENUM (
    'unknown',
    'denied',
    'denied_invalid_signature',
    'audit_only'
);
CREATE TYPE santa_signing_status AS ENUM (
    'unspecified',
    'unsigned',
    'invalid',
    'adhoc',
    'development',
    'production'
);
CREATE TYPE santa_sync_target_phase AS ENUM ('desired', 'applied');
CREATE TYPE santa_file_access_action AS ENUM ('none', 'audit_only', 'disable');

-- Santa host state and synchronization --------------------------------------

CREATE TABLE santa_hosts (
    host_id BIGINT PRIMARY KEY REFERENCES hosts (id) ON DELETE CASCADE,
    machine_id TEXT NOT NULL,
    serial_number TEXT NOT NULL,
    santa_version TEXT NOT NULL DEFAULT '',
    client_mode_reported santa_client_mode NOT NULL DEFAULT 'unknown',
    primary_user TEXT NOT NULL DEFAULT '',
    primary_user_groups TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    sip_status BIGINT,
    last_seen_at TIMESTAMPTZ,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX santa_hosts_machine_id_idx ON santa_hosts (machine_id);
CREATE INDEX santa_hosts_serial_number_idx ON santa_hosts (serial_number);

CREATE TABLE santa_sync_state (
    host_id BIGINT PRIMARY KEY REFERENCES hosts (id) ON DELETE CASCADE,
    pending_full_sync BOOLEAN NOT NULL DEFAULT FALSE,
    pending_payload_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (pending_payload_rule_count >= 0),
    pending_preflight_at TIMESTAMPTZ,
    desired_binary_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (desired_binary_rule_count >= 0),
    desired_certificate_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (desired_certificate_rule_count >= 0),
    desired_teamid_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (desired_teamid_rule_count >= 0),
    desired_signingid_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (desired_signingid_rule_count >= 0),
    desired_cdhash_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (desired_cdhash_rule_count >= 0),
    desired_compiler_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (desired_compiler_rule_count >= 0),
    binary_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (binary_rule_count >= 0),
    certificate_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (certificate_rule_count >= 0),
    teamid_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (teamid_rule_count >= 0),
    signingid_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (signingid_rule_count >= 0),
    cdhash_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (cdhash_rule_count >= 0),
    compiler_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (compiler_rule_count >= 0),
    transitive_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (transitive_rule_count >= 0),
    rules_received BIGINT NOT NULL DEFAULT 0 CHECK (rules_received >= 0),
    rules_processed BIGINT NOT NULL DEFAULT 0 CHECK (rules_processed >= 0),
    last_rule_sync_attempt_at TIMESTAMPTZ,
    last_rule_sync_success_at TIMESTAMPTZ,
    last_clean_sync_at TIMESTAMPTZ,
    last_reported_counts_match_at TIMESTAMPTZ,
    preflight_rules_hash TEXT NOT NULL DEFAULT '',
    confirmed_rules_hash TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT santa_sync_state_preflight_rules_hash_check CHECK (
        preflight_rules_hash = '' OR preflight_rules_hash ~ '^[0-9a-f]{32}$'
    ),
    CONSTRAINT santa_sync_state_confirmed_rules_hash_check CHECK (
        confirmed_rules_hash = '' OR confirmed_rules_hash ~ '^[0-9a-f]{32}$'
    )
);

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
    notification_app_name TEXT NOT NULL DEFAULT '',
    payload_hash TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (host_id, phase, position),
    CONSTRAINT santa_sync_targets_host_phase_identity_key
        UNIQUE (host_id, phase, rule_type, identifier),
    CHECK (NULLIF(btrim(identifier), '') IS NOT NULL),
    CHECK (NULLIF(btrim(payload_hash), '') IS NOT NULL)
);

CREATE INDEX santa_sync_targets_host_phase_idx
    ON santa_sync_targets (host_id, phase);

-- Configuration and rules ---------------------------------------------------

CREATE TABLE santa_configurations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    position INT NOT NULL UNIQUE,
    client_mode santa_client_mode NOT NULL,
    enable_bundles BOOLEAN NOT NULL,
    enable_transitive_rules BOOLEAN NOT NULL,
    enable_all_event_upload BOOLEAN NOT NULL,
    disable_unknown_event_upload BOOLEAN NOT NULL,
    full_sync_interval_seconds INT NOT NULL CHECK (full_sync_interval_seconds >= 60),
    batch_size INT NOT NULL CHECK (batch_size BETWEEN 5 AND 100),
    allowed_path_regex TEXT NOT NULL,
    blocked_path_regex TEXT NOT NULL,
    override_file_access_action santa_file_access_action NOT NULL,
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
        OR (policy = 'cel' AND NULLIF(btrim(COALESCE(cel_expression, '')), '') IS NOT NULL)
        OR (policy <> 'cel' AND NULLIF(btrim(COALESCE(cel_expression, '')), '') IS NULL)
    )
);

CREATE INDEX santa_rule_targets_label_idx
    ON santa_rule_targets (label_id);

-- Reference data -------------------------------------------------------------

CREATE TABLE santa_executables (
    id BIGSERIAL PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
    file_name TEXT NOT NULL DEFAULT '',
    file_bundle_id TEXT NOT NULL DEFAULT '',
    file_bundle_path TEXT NOT NULL DEFAULT '',
    file_bundle_executable_rel_path TEXT NOT NULL DEFAULT '',
    file_bundle_name TEXT NOT NULL DEFAULT '',
    file_bundle_version TEXT NOT NULL DEFAULT '',
    file_bundle_version_string TEXT NOT NULL DEFAULT '',
    file_bundle_hash TEXT NOT NULL DEFAULT '',
    file_bundle_hash_millis BIGINT NOT NULL DEFAULT 0 CHECK (file_bundle_hash_millis >= 0),
    file_bundle_binary_count BIGINT NOT NULL DEFAULT 0 CHECK (file_bundle_binary_count >= 0),
    signing_id TEXT NOT NULL DEFAULT '',
    team_id TEXT NOT NULL DEFAULT '',
    cdhash TEXT NOT NULL DEFAULT '',
    codesigning_flags BIGINT NOT NULL DEFAULT 0,
    signing_status santa_signing_status NOT NULL DEFAULT 'unspecified',
    secure_signing_time TIMESTAMPTZ,
    signing_time TIMESTAMPTZ,
    entitlements JSONB CHECK (entitlements IS NULL OR jsonb_typeof(entitlements) = 'object'),
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NULLIF(btrim(sha256), '') IS NOT NULL)
);

CREATE INDEX santa_executables_file_name_idx ON santa_executables (file_name);
CREATE INDEX santa_executables_signing_id_idx ON santa_executables (signing_id);
CREATE INDEX santa_executables_team_id_idx ON santa_executables (team_id);
CREATE INDEX santa_executables_cdhash_idx ON santa_executables (cdhash);

CREATE TABLE santa_signing_chains (
    id BIGSERIAL PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
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

CREATE TABLE santa_bundles (
    id BIGSERIAL PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
    bundle_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    executable_rel_path TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    version_string TEXT NOT NULL DEFAULT '',
    binary_count BIGINT NOT NULL DEFAULT 0 CHECK (binary_count >= 0),
    hash_millis BIGINT NOT NULL DEFAULT 0 CHECK (hash_millis >= 0),
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

-- Events ---------------------------------------------------------------------

CREATE TABLE santa_execution_events (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    executable_id BIGINT NOT NULL REFERENCES santa_executables (id) ON DELETE RESTRICT,
    file_path TEXT NOT NULL DEFAULT '',
    executing_user TEXT NOT NULL DEFAULT '',
    logged_in_users TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    current_sessions TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    decision santa_execution_decision NOT NULL,
    static_rule BOOLEAN NOT NULL DEFAULT FALSE,
    pid INT NOT NULL DEFAULT 0,
    ppid INT NOT NULL DEFAULT 0,
    parent_name TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX santa_execution_events_host_time_idx
    ON santa_execution_events (host_id, occurred_at DESC);
CREATE INDEX santa_execution_events_executable_time_idx
    ON santa_execution_events (executable_id, occurred_at DESC);
CREATE INDEX santa_execution_events_user_time_idx
    ON santa_execution_events (executing_user, occurred_at DESC);
CREATE INDEX santa_execution_events_decision_ingested_idx
    ON santa_execution_events (decision, ingested_at DESC);
CREATE INDEX santa_execution_events_occurred_at_idx
    ON santa_execution_events (occurred_at DESC, id DESC);

CREATE TABLE santa_file_access_events (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    rule_version TEXT NOT NULL DEFAULT '',
    rule_name TEXT NOT NULL DEFAULT '',
    target TEXT NOT NULL DEFAULT '',
    decision santa_file_access_decision NOT NULL,
    process_chain JSONB NOT NULL DEFAULT '[]'::JSONB,
    primary_process_sha256 TEXT NOT NULL DEFAULT '',
    primary_process_path TEXT NOT NULL DEFAULT '',
    primary_process_signing_id TEXT NOT NULL DEFAULT '',
    primary_process_team_id TEXT NOT NULL DEFAULT '',
    primary_process_cdhash TEXT NOT NULL DEFAULT '',
    primary_process_pid INT NOT NULL DEFAULT 0,
    occurred_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (jsonb_typeof(process_chain) = 'array')
);

CREATE INDEX santa_file_access_events_host_time_idx
    ON santa_file_access_events (host_id, occurred_at DESC);
CREATE INDEX santa_file_access_events_decision_ingested_idx
    ON santa_file_access_events (decision, ingested_at DESC);
CREATE INDEX santa_file_access_events_primary_process_sha_idx
    ON santa_file_access_events (primary_process_sha256)
    WHERE primary_process_sha256 <> '';
CREATE INDEX santa_file_access_events_occurred_at_idx
    ON santa_file_access_events (occurred_at DESC, id DESC);

CREATE TABLE santa_standalone_rule_creation_events (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    identifier TEXT NOT NULL,
    decision santa_execution_decision NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NULLIF(btrim(identifier), '') IS NOT NULL)
);

CREATE INDEX santa_standalone_rule_creation_events_host_time_idx
    ON santa_standalone_rule_creation_events (host_id, occurred_at DESC);
CREATE INDEX santa_standalone_rule_creation_events_occurred_at_idx
    ON santa_standalone_rule_creation_events (occurred_at);

-- Rule resolution ------------------------------------------------------------

CREATE FUNCTION santa_rule_type_sort(_rule_type santa_rule_type)
RETURNS INTEGER
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT CASE _rule_type
        WHEN 'cdhash' THEN 1
        WHEN 'binary' THEN 2
        WHEN 'signingid' THEN 3
        WHEN 'certificate' THEN 4
        WHEN 'teamid' THEN 5
        WHEN 'bundle' THEN 6
        ELSE 7
    END
$$;

CREATE FUNCTION santa_resolved_rules_for_host(_host_id BIGINT)
RETURNS TABLE (
    rule_id BIGINT,
    rule_type TEXT,
    identifier TEXT,
    name TEXT,
    description TEXT,
    policy TEXT,
    cel_expression TEXT,
    custom_message TEXT,
    custom_url TEXT,
    notification_app_name TEXT,
    rule_type_sort INTEGER
)
LANGUAGE sql
STABLE
AS $$
WITH host_labels AS (
    SELECT label_id
    FROM label_membership
    WHERE host_id = _host_id
),
matching_includes AS (
    SELECT
        r.id AS rule_id,
        r.rule_type,
        r.identifier,
        r.name,
        r.description,
        i.policy,
        COALESCE(i.cel_expression, '') AS cel_expression,
        r.custom_message,
        r.custom_url,
        i.position::bigint AS matched_include_id,
        santa_rule_type_sort(r.rule_type) AS rule_type_sort,
        row_number() OVER (PARTITION BY r.id ORDER BY i.position) AS include_rank
    FROM santa_rules r
    JOIN santa_rule_targets i ON i.rule_id = r.id AND i.direction = 'include'
    JOIN host_labels include_hl ON include_hl.label_id = i.label_id
    WHERE NOT EXISTS (
        SELECT 1
        FROM santa_rule_targets el
        JOIN host_labels hl ON hl.label_id = el.label_id
        WHERE el.rule_id = r.id
          AND el.direction = 'exclude'
    )
),
selected_includes AS (
    SELECT
        rule_id,
        rule_type,
        identifier,
        name,
        description,
        policy,
        cel_expression,
        custom_message,
        custom_url,
        rule_type_sort
    FROM matching_includes
    WHERE include_rank = 1
),
expanded_rules AS (
    SELECT
        rule_id,
        rule_type,
        identifier,
        name,
        description,
        policy,
        cel_expression,
        custom_message,
        custom_url,
        ''::text AS notification_app_name,
        rule_type_sort
    FROM selected_includes
    WHERE rule_type <> 'bundle'
    UNION ALL
    SELECT
        si.rule_id,
        'binary'::santa_rule_type AS rule_type,
        e.sha256 AS identifier,
        si.name,
        si.description,
        si.policy,
        si.cel_expression,
        si.custom_message,
        si.custom_url,
        COALESCE(NULLIF(b.name, ''), '') AS notification_app_name,
        santa_rule_type_sort('binary') AS rule_type_sort
    FROM selected_includes si
    JOIN santa_bundles b ON b.sha256 = si.identifier AND b.uploaded_at IS NOT NULL
    JOIN santa_bundle_executables be ON be.bundle_id = b.id
    JOIN santa_executables e ON e.id = be.executable_id
    WHERE si.rule_type = 'bundle'
      AND e.sha256 <> ''
)
SELECT
    rule_id,
    rule_type::text,
    identifier,
    name,
    description,
    policy::text,
    cel_expression,
    custom_message,
    custom_url,
    notification_app_name,
    rule_type_sort
FROM expanded_rules
$$;
