-- +goose Up

CREATE TYPE santa_file_access_action AS ENUM ('none', 'audit_only', 'disable');

ALTER TYPE santa_policy ADD VALUE 'silent_gui_blocklist' BEFORE 'cel';
ALTER TYPE santa_policy ADD VALUE 'silent_tty_blocklist' BEFORE 'cel';
ALTER TYPE santa_execution_decision ADD VALUE 'block_binary_mismatch';
ALTER TYPE santa_execution_decision ADD VALUE 'allow_platform';

ALTER TABLE santa_hosts
    DROP COLUMN os_build,
    DROP COLUMN model_identifier;

ALTER TABLE santa_sync_state
    DROP COLUMN client_rules_hash,
    ADD COLUMN desired_compiler_rule_count INT NOT NULL DEFAULT 0
        CHECK (desired_compiler_rule_count >= 0),
    ADD COLUMN compiler_rule_count INT NOT NULL DEFAULT 0
        CHECK (compiler_rule_count >= 0),
    ADD COLUMN transitive_rule_count INT NOT NULL DEFAULT 0
        CHECK (transitive_rule_count >= 0);

DROP TABLE santa_sync_pending_rules;

ALTER TABLE santa_configurations
    ADD COLUMN disable_unknown_event_upload BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN override_file_access_action santa_file_access_action NOT NULL DEFAULT 'none';

ALTER TABLE santa_configurations
    ALTER COLUMN disable_unknown_event_upload DROP DEFAULT,
    ALTER COLUMN override_file_access_action DROP DEFAULT;

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
