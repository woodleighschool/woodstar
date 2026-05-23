-- +goose Up

CREATE TYPE santa_client_mode AS ENUM ('unknown', 'monitor', 'lockdown', 'standalone');
CREATE TYPE santa_removable_media_action AS ENUM ('allow', 'block', 'remount');
CREATE TYPE santa_rule_type AS ENUM ('binary', 'certificate', 'teamid', 'signingid', 'cdhash');
CREATE TYPE santa_policy AS ENUM ('allowlist', 'allowlist_compiler', 'blocklist', 'silent_blocklist', 'cel');
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
    'bundle_binary'
);

-- Santa host observation -----------------------------------------------------

CREATE TABLE santa_hosts (
    host_id BIGINT PRIMARY KEY REFERENCES hosts (id) ON DELETE CASCADE,
    machine_id TEXT NOT NULL,
    serial_number TEXT NOT NULL,
    santa_version TEXT NOT NULL DEFAULT '',
    client_mode_reported santa_client_mode NOT NULL DEFAULT 'unknown',
    primary_user TEXT NOT NULL DEFAULT '',
    primary_user_groups TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    sip_status SMALLINT,
    os_build TEXT NOT NULL DEFAULT '',
    model_identifier TEXT NOT NULL DEFAULT '',
    last_seen_at TIMESTAMPTZ,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX santa_hosts_machine_id_idx ON santa_hosts (machine_id);
CREATE INDEX santa_hosts_serial_number_idx ON santa_hosts (serial_number);

CREATE TABLE santa_sync_state (
    host_id BIGINT PRIMARY KEY REFERENCES hosts (id) ON DELETE CASCADE,
    client_rules_hash TEXT NOT NULL DEFAULT '',
    desired_targets JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(desired_targets) = 'array'),
    applied_targets JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(applied_targets) = 'array'),
    pending_payload JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(pending_payload) = 'array'),
    pending_payload_rule_count BIGINT NOT NULL DEFAULT 0 CHECK (pending_payload_rule_count >= 0),
    pending_full_sync BOOLEAN NOT NULL DEFAULT FALSE,
    pending_preflight_at TIMESTAMPTZ,
    last_rule_sync_attempt_at TIMESTAMPTZ,
    last_rule_sync_success_at TIMESTAMPTZ,
    last_clean_sync_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Santa configurations -------------------------------------------------------

CREATE TABLE santa_configurations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    position INT NOT NULL UNIQUE,
    client_mode santa_client_mode NOT NULL DEFAULT 'monitor',
    enable_bundles BOOLEAN,
    enable_transitive_rules BOOLEAN,
    enable_all_event_upload BOOLEAN,
    full_sync_interval_seconds INT CHECK (
        full_sync_interval_seconds IS NULL
        OR full_sync_interval_seconds >= 60
    ),
    batch_size INT,
    allowed_path_regex TEXT,
    blocked_path_regex TEXT,
    removable_media_action santa_removable_media_action,
    removable_media_remount_flags TEXT[],
    encrypted_removable_media_action santa_removable_media_action,
    encrypted_removable_media_remount_flags TEXT[],
    event_detail_url TEXT,
    event_detail_text TEXT,
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

CREATE TABLE santa_configuration_labels (
    label_id BIGINT PRIMARY KEY REFERENCES labels (id) ON DELETE CASCADE,
    configuration_id BIGINT NOT NULL REFERENCES santa_configurations (id) ON DELETE CASCADE
);

CREATE INDEX santa_configuration_labels_configuration_idx
    ON santa_configuration_labels (configuration_id);

-- Santa rules ----------------------------------------------------------------

CREATE TABLE santa_rules (
    id BIGSERIAL PRIMARY KEY,
    rule_type santa_rule_type NOT NULL,
    identifier TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    custom_message TEXT NOT NULL DEFAULT '',
    custom_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (rule_type, identifier),
    CHECK (NULLIF(btrim(identifier), '') IS NOT NULL)
);

CREATE TABLE santa_rule_includes (
    id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT NOT NULL REFERENCES santa_rules (id) ON DELETE CASCADE,
    position INT NOT NULL,
    policy santa_policy NOT NULL,
    cel_expression TEXT,
    UNIQUE (rule_id, position),
    CHECK (
        (policy = 'cel' AND NULLIF(btrim(COALESCE(cel_expression, '')), '') IS NOT NULL)
        OR (policy <> 'cel' AND NULLIF(btrim(COALESCE(cel_expression, '')), '') IS NULL)
    )
);

CREATE TABLE santa_rule_include_labels (
    include_id BIGINT NOT NULL REFERENCES santa_rule_includes (id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (include_id, label_id)
);

CREATE INDEX santa_rule_include_labels_label_idx
    ON santa_rule_include_labels (label_id);

CREATE TABLE santa_rule_exclude_labels (
    rule_id BIGINT NOT NULL REFERENCES santa_rules (id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (rule_id, label_id)
);

CREATE INDEX santa_rule_exclude_labels_label_idx
    ON santa_rule_exclude_labels (label_id);

-- Santa execution events -----------------------------------------------------

CREATE TABLE santa_executables (
    id BIGSERIAL PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
    file_name TEXT NOT NULL DEFAULT '',
    file_bundle_id TEXT NOT NULL DEFAULT '',
    file_bundle_path TEXT NOT NULL DEFAULT '',
    signing_id TEXT NOT NULL DEFAULT '',
    team_id TEXT NOT NULL DEFAULT '',
    cdhash TEXT NOT NULL DEFAULT '',
    entitlements JSONB NOT NULL DEFAULT '{}'::JSONB CHECK (jsonb_typeof(entitlements) = 'object'),
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

-- Santa sync tokens ----------------------------------------------------------

CREATE TABLE santa_sync_tokens (
    id BIGSERIAL PRIMARY KEY,
    value_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    CHECK (NULLIF(btrim(value_hash), '') IS NOT NULL)
);

-- +goose Down

DROP TABLE santa_sync_tokens;
DROP TABLE santa_execution_events;
DROP TABLE santa_executable_signing_chains;
DROP TABLE santa_signing_chains;
DROP TABLE santa_executables;
DROP TABLE santa_rule_exclude_labels;
DROP TABLE santa_rule_include_labels;
DROP TABLE santa_rule_includes;
DROP TABLE santa_rules;
DROP TABLE santa_configuration_labels;
DROP TABLE santa_configurations;
DROP TABLE santa_sync_state;
DROP TABLE santa_hosts;

DROP TYPE santa_execution_decision;
DROP TYPE santa_policy;
DROP TYPE santa_rule_type;
DROP TYPE santa_removable_media_action;
DROP TYPE santa_client_mode;
