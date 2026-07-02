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
        COALESCE(NULLIF(b.name, ''), NULLIF(b.bundle_id, ''), b.sha256) AS notification_app_name,
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
