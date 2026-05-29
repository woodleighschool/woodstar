-- name: UpsertSantaHostObservation :exec
INSERT INTO santa_hosts (
    host_id,
    machine_id,
    serial_number,
    santa_version,
    client_mode_reported,
    primary_user,
    primary_user_groups,
    sip_status,
    os_build,
    model_identifier,
    last_seen_at
)
VALUES (
    @host_id,
    @machine_id,
    @serial_number,
    @santa_version,
    @client_mode_reported::santa_client_mode,
    @primary_user,
    @primary_user_groups,
    @sip_status,
    @os_build,
    @model_identifier,
    COALESCE(sqlc.narg(last_seen_at)::timestamptz, now())
)
ON CONFLICT (host_id) DO UPDATE SET
    machine_id = EXCLUDED.machine_id,
    serial_number = EXCLUDED.serial_number,
    santa_version = EXCLUDED.santa_version,
    client_mode_reported = EXCLUDED.client_mode_reported,
    primary_user = EXCLUDED.primary_user,
    primary_user_groups = EXCLUDED.primary_user_groups,
    sip_status = EXCLUDED.sip_status,
    os_build = EXCLUDED.os_build,
    model_identifier = EXCLUDED.model_identifier,
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = now();

-- name: CreateSantaConfiguration :one
INSERT INTO santa_configurations (
    name,
    description,
    position,
    client_mode,
    enable_bundles,
    enable_transitive_rules,
    enable_all_event_upload,
    full_sync_interval_seconds,
    batch_size,
    allowed_path_regex,
    blocked_path_regex,
    removable_media_action,
    removable_media_remount_flags,
    encrypted_removable_media_action,
    encrypted_removable_media_remount_flags,
    event_detail_url,
    event_detail_text
)
VALUES (
    @name,
    @description,
    (SELECT COALESCE(MAX(position) + 1, 0) FROM santa_configurations),
    @client_mode::santa_client_mode,
    @enable_bundles,
    @enable_transitive_rules,
    @enable_all_event_upload,
    @full_sync_interval_seconds::integer,
    @batch_size::integer,
    @allowed_path_regex,
    @blocked_path_regex,
    sqlc.narg(removable_media_action)::santa_removable_media_action,
    sqlc.narg(removable_media_remount_flags)::text[],
    sqlc.narg(encrypted_removable_media_action)::santa_removable_media_action,
    sqlc.narg(encrypted_removable_media_remount_flags)::text[],
    @event_detail_url,
    @event_detail_text
)
RETURNING *;

-- name: GetSantaConfigurationByID :one
SELECT *
FROM santa_configurations
WHERE id = @id;

-- name: UpdateSantaConfiguration :one
UPDATE santa_configurations
SET
    name = @name,
    description = @description,
    client_mode = @client_mode::santa_client_mode,
    enable_bundles = @enable_bundles,
    enable_transitive_rules = @enable_transitive_rules,
    enable_all_event_upload = @enable_all_event_upload,
    full_sync_interval_seconds = @full_sync_interval_seconds::integer,
    batch_size = @batch_size::integer,
    allowed_path_regex = @allowed_path_regex,
    blocked_path_regex = @blocked_path_regex,
    removable_media_action = sqlc.narg(removable_media_action)::santa_removable_media_action,
    removable_media_remount_flags = sqlc.narg(removable_media_remount_flags)::text[],
    encrypted_removable_media_action = sqlc.narg(encrypted_removable_media_action)::santa_removable_media_action,
    encrypted_removable_media_remount_flags = sqlc.narg(encrypted_removable_media_remount_flags)::text[],
    event_detail_url = @event_detail_url,
    event_detail_text = @event_detail_text,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteSantaConfiguration :one
DELETE FROM santa_configurations
WHERE id = @id
RETURNING id;

-- name: DeleteSantaConfigurations :many
DELETE FROM santa_configurations
WHERE id = ANY(@ids::bigint[])
RETURNING id;

-- name: CreateSantaRule :one
INSERT INTO santa_rules (
    rule_type,
    identifier,
    name,
    description,
    custom_message,
    custom_url
)
VALUES (
    @rule_type::santa_rule_type,
    @identifier,
    @name,
    @description,
    @custom_message,
    @custom_url
)
RETURNING *;

-- name: GetSantaRuleByID :one
SELECT *
FROM santa_rules
WHERE id = @id;

-- name: UpdateSantaRule :one
UPDATE santa_rules
SET
    rule_type = @rule_type::santa_rule_type,
    identifier = @identifier,
    name = @name,
    description = @description,
    custom_message = @custom_message,
    custom_url = @custom_url,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteSantaRule :one
DELETE FROM santa_rules
WHERE id = @id
RETURNING id;

-- name: DeleteSantaRules :many
DELETE FROM santa_rules
WHERE id = ANY(@ids::bigint[])
RETURNING id;

-- name: GetHostIDByMachineID :one
SELECT id
FROM hosts
WHERE hardware_uuid = @machine_id;

-- name: GetObservedSantaHostState :one
SELECT
    sh.santa_version,
    sh.client_mode_reported,
    sh.last_seen_at,
    ss.last_clean_sync_at
FROM santa_hosts sh
LEFT JOIN santa_sync_state ss ON ss.host_id = sh.host_id
WHERE sh.host_id = @host_id;

-- name: GetSantaSyncSummary :one
SELECT
    (SELECT count(*)::integer FROM santa_sync_targets st WHERE st.host_id = @summary_host_id AND st.phase = 'desired') AS desired_count,
    (SELECT count(*)::integer FROM santa_sync_targets st WHERE st.host_id = @summary_host_id AND st.phase = 'applied') AS applied_count,
    (SELECT count(*)::integer FROM santa_sync_pending_rules pr WHERE pr.host_id = @summary_host_id) AS pending_count;

-- name: ListSantaConfigurationIDsByPosition :many
SELECT id
FROM santa_configurations
ORDER BY position, id;

-- name: SetSantaConfigurationPositions :exec
UPDATE santa_configurations c
SET position = -ordered.position
FROM unnest(@ordered_ids::bigint[]) WITH ORDINALITY AS ordered(id, position)
WHERE c.id = ordered.id;

-- name: NormalizeSantaConfigurationPositions :exec
UPDATE santa_configurations
SET position = -position - 1;

-- name: ResolveSantaConfigurationForHost :one
SELECT
    sqlc.embed(c),
    l.id AS label_id,
    l.name AS label_name
FROM santa_configurations c
JOIN santa_configuration_labels cl ON cl.configuration_id = c.id
JOIN label_membership lm ON lm.label_id = cl.label_id AND lm.host_id = @host_id
JOIN labels l ON l.id = cl.label_id
ORDER BY c.position, c.id, l.name, l.id
LIMIT 1;

-- name: DeleteSantaConfigurationLabels :exec
DELETE FROM santa_configuration_labels
WHERE configuration_id = @configuration_id;

-- name: InsertSantaConfigurationLabels :exec
INSERT INTO santa_configuration_labels (label_id, configuration_id)
SELECT label_id, @configuration_id
FROM unnest(@label_ids::bigint[]) AS label_id;

-- name: FindSantaConfigurationLabelConflict :one
SELECT
    cl.label_id,
    c.id AS configuration_id,
    c.name AS configuration_name
FROM santa_configuration_labels cl
JOIN santa_configurations c ON c.id = cl.configuration_id
WHERE cl.label_id = ANY(@label_ids::bigint[])
  AND c.id <> @configuration_id
ORDER BY cl.label_id
LIMIT 1;

-- name: ListSantaConfigurationLabels :many
SELECT configuration_id, label_id
FROM santa_configuration_labels
WHERE configuration_id = ANY(@configuration_ids::bigint[])
ORDER BY configuration_id, label_id;

-- name: SantaRuleExists :one
SELECT EXISTS (
    SELECT 1
    FROM santa_rules
    WHERE id = @id
);

-- name: ListSantaRuleIncludeIDs :many
SELECT id
FROM santa_rule_includes
WHERE rule_id = @rule_id
ORDER BY position, id;

-- name: SetSantaRuleIncludePositions :exec
UPDATE santa_rule_includes i
SET position = -ordered.position
FROM unnest(@ordered_ids::bigint[]) WITH ORDINALITY AS ordered(id, position)
WHERE i.id = ordered.id AND i.rule_id = @rule_id;

-- name: NormalizeSantaRuleIncludePositions :exec
UPDATE santa_rule_includes
SET position = -position - 1
WHERE rule_id = @rule_id;

-- name: CountSantaRulesForHost :one
WITH host_labels AS (
    SELECT label_id
    FROM label_membership
    WHERE host_id = @host_id
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
        i.id AS matched_include_id,
        CASE r.rule_type
            WHEN 'cdhash' THEN 1
            WHEN 'binary' THEN 2
            WHEN 'signingid' THEN 3
            WHEN 'certificate' THEN 4
            WHEN 'teamid' THEN 5
            WHEN 'bundle' THEN 6
            ELSE 7
        END AS rule_type_sort,
        row_number() OVER (PARTITION BY r.id ORDER BY i.position, i.id) AS include_rank
    FROM santa_rules r
    JOIN santa_rule_includes i ON i.rule_id = r.id
    JOIN host_labels include_hl ON include_hl.label_id = i.label_id
    WHERE NOT EXISTS (
        SELECT 1
        FROM santa_rule_exclude_labels el
        JOIN host_labels hl ON hl.label_id = el.label_id
        WHERE el.rule_id = r.id
    )
),
selected_includes AS (
    SELECT *
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
        matched_include_id,
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
        si.matched_include_id,
        2 AS rule_type_sort
    FROM selected_includes si
    JOIN santa_bundles b ON b.sha256 = si.identifier AND b.uploaded_at IS NOT NULL
    JOIN santa_bundle_executables be ON be.bundle_id = b.id
    JOIN santa_executables e ON e.id = be.executable_id
    WHERE si.rule_type = 'bundle'
      AND e.sha256 <> ''
)
SELECT count(*)::integer
FROM expanded_rules;

-- name: ListSantaRulesForHost :many
WITH host_labels AS (
    SELECT label_id
    FROM label_membership
    WHERE host_id = @host_id
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
        i.id AS matched_include_id,
        CASE r.rule_type
            WHEN 'cdhash' THEN 1
            WHEN 'binary' THEN 2
            WHEN 'signingid' THEN 3
            WHEN 'certificate' THEN 4
            WHEN 'teamid' THEN 5
            WHEN 'bundle' THEN 6
            ELSE 7
        END AS rule_type_sort,
        row_number() OVER (PARTITION BY r.id ORDER BY i.position, i.id) AS include_rank
    FROM santa_rules r
    JOIN santa_rule_includes i ON i.rule_id = r.id
    JOIN host_labels include_hl ON include_hl.label_id = i.label_id
    WHERE NOT EXISTS (
        SELECT 1
        FROM santa_rule_exclude_labels el
        JOIN host_labels hl ON hl.label_id = el.label_id
        WHERE el.rule_id = r.id
    )
),
selected_includes AS (
    SELECT *
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
        matched_include_id,
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
        si.matched_include_id,
        2 AS rule_type_sort
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
    matched_include_id,
    rule_type_sort
FROM expanded_rules
ORDER BY rule_type_sort, identifier, rule_id;

-- name: ListSantaRulesForHostPage :many
WITH host_labels AS (
    SELECT label_id
    FROM label_membership
    WHERE host_id = @host_id
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
        i.id AS matched_include_id,
        CASE r.rule_type
            WHEN 'cdhash' THEN 1
            WHEN 'binary' THEN 2
            WHEN 'signingid' THEN 3
            WHEN 'certificate' THEN 4
            WHEN 'teamid' THEN 5
            WHEN 'bundle' THEN 6
            ELSE 7
        END AS rule_type_sort,
        row_number() OVER (PARTITION BY r.id ORDER BY i.position, i.id) AS include_rank
    FROM santa_rules r
    JOIN santa_rule_includes i ON i.rule_id = r.id
    JOIN host_labels include_hl ON include_hl.label_id = i.label_id
    WHERE NOT EXISTS (
        SELECT 1
        FROM santa_rule_exclude_labels el
        JOIN host_labels hl ON hl.label_id = el.label_id
        WHERE el.rule_id = r.id
    )
),
selected_includes AS (
    SELECT *
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
        matched_include_id,
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
        si.matched_include_id,
        2 AS rule_type_sort
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
    matched_include_id,
    rule_type_sort
FROM expanded_rules
ORDER BY rule_type_sort, identifier, rule_id
LIMIT @limit_count OFFSET @offset_count;

-- name: ListAppliedSantaSyncTargets :many
SELECT
    rule_type::text,
    identifier,
    policy::text,
    cel_expression,
    custom_message,
    custom_url,
    notification_app_name,
    payload_hash
FROM santa_sync_targets
WHERE host_id = @host_id AND phase = 'applied';

-- name: DeleteSantaRuleExcludeLabels :exec
DELETE FROM santa_rule_exclude_labels
WHERE rule_id = @rule_id;

-- name: DeleteSantaRuleIncludes :exec
DELETE FROM santa_rule_includes
WHERE rule_id = @rule_id;

-- name: InsertSantaRuleIncludes :exec
WITH input AS (
    SELECT
        p.policy,
        ce.cel_expression,
        l.label_id,
        p.position
    FROM unnest(@policies::text[]) WITH ORDINALITY AS p(policy, position)
    JOIN unnest(@cel_expressions::text[]) WITH ORDINALITY AS ce(cel_expression, position) USING (position)
    JOIN unnest(@label_ids::bigint[]) WITH ORDINALITY AS l(label_id, position) USING (position)
)
INSERT INTO santa_rule_includes (rule_id, position, policy, cel_expression, label_id)
SELECT
    @rule_id,
    position - 1,
    policy::santa_policy,
    NULLIF(cel_expression, ''),
    label_id
FROM input
ORDER BY position;

-- name: InsertSantaRuleExcludeLabels :exec
INSERT INTO santa_rule_exclude_labels (rule_id, label_id)
SELECT @rule_id, label_id
FROM unnest(@label_ids::bigint[]) AS label_id;

-- name: ListSantaRuleIncludes :many
SELECT
    rule_id,
    id,
    position,
    policy::text,
    COALESCE(cel_expression, '') AS cel_expression,
    label_id
FROM santa_rule_includes
WHERE rule_id = ANY(@rule_ids::bigint[])
ORDER BY rule_id, position, id;

-- name: ListSantaRuleExcludeLabels :many
SELECT rule_id, label_id
FROM santa_rule_exclude_labels
WHERE rule_id = ANY(@rule_ids::bigint[])
ORDER BY rule_id, label_id;

-- name: ListBuiltinLabelIDs :many
SELECT id
FROM labels
WHERE id = ANY(@label_ids::bigint[]) AND label_type = 'builtin';

-- name: IsSantaBundleComplete :one
SELECT (uploaded_at IS NOT NULL)::boolean AS complete
FROM santa_bundles
WHERE sha256 = @sha256;

-- name: ListSantaRuleTargets :many
WITH candidate_sources AS (
    SELECT
        'binary'::text AS target_type,
        e.sha256 AS identifier,
        COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.sha256) AS name,
        NULLIF(e.file_name, '') AS detail,
        e.file_bundle_id AS bundle_id,
        COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version) AS version,
        0::integer AS binary_count,
        0::integer AS collected_binary_count,
        true AS complete
    FROM santa_executables e
    WHERE e.sha256 <> ''
    UNION ALL
    SELECT
        'binary'::text,
        p.executable_sha256,
        COALESCE(NULLIF(st.display_name, ''), st.name, p.executable_sha256),
        COALESCE(NULLIF(p.executable_path, ''), p.installed_path),
        s.bundle_identifier,
        s.version,
        0::integer,
        0::integer,
        true
    FROM host_software_installed_paths p
    JOIN software s ON s.id = p.software_id
    JOIN software_titles st ON st.id = s.title_id
    WHERE p.executable_sha256 IS NOT NULL AND p.executable_sha256 <> ''
    UNION ALL
    SELECT
        'certificate'::text,
        c.sha256,
        COALESCE(NULLIF(c.common_name, ''), c.sha256),
        c.organizational_unit,
        ''::text,
        ''::text,
        0::integer,
        0::integer,
        true
    FROM santa_certificates c
    WHERE c.sha256 <> ''
    UNION ALL
    SELECT
        'teamid'::text,
        e.team_id,
        e.team_id,
        COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, '')),
        e.file_bundle_id,
        COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version),
        0::integer,
        0::integer,
        true
    FROM santa_executables e
    WHERE e.team_id <> ''
    UNION ALL
    SELECT
        'teamid'::text,
        p.team_identifier,
        p.team_identifier,
        COALESCE(NULLIF(st.display_name, ''), st.name),
        s.bundle_identifier,
        s.version,
        0::integer,
        0::integer,
        true
    FROM host_software_installed_paths p
    JOIN software s ON s.id = p.software_id
    JOIN software_titles st ON st.id = s.title_id
    WHERE p.team_identifier <> ''
    UNION ALL
    SELECT
        'signingid'::text,
        e.signing_id,
        e.signing_id,
        COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, '')),
        e.file_bundle_id,
        COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version),
        0::integer,
        0::integer,
        true
    FROM santa_executables e
    WHERE e.signing_id <> ''
    UNION ALL
    SELECT
        'signingid'::text,
        p.team_identifier || ':' || s.bundle_identifier,
        p.team_identifier || ':' || s.bundle_identifier,
        COALESCE(NULLIF(st.display_name, ''), st.name),
        s.bundle_identifier,
        s.version,
        0::integer,
        0::integer,
        true
    FROM host_software_installed_paths p
    JOIN software s ON s.id = p.software_id
    JOIN software_titles st ON st.id = s.title_id
    WHERE p.team_identifier <> '' AND s.bundle_identifier <> ''
    UNION ALL
    SELECT
        'cdhash'::text,
        e.cdhash,
        e.cdhash,
        COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, '')),
        e.file_bundle_id,
        COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version),
        0::integer,
        0::integer,
        true
    FROM santa_executables e
    WHERE e.cdhash <> ''
    UNION ALL
    SELECT
        'cdhash'::text,
        p.cdhash_sha256,
        p.cdhash_sha256,
        COALESCE(NULLIF(st.display_name, ''), st.name),
        s.bundle_identifier,
        s.version,
        0::integer,
        0::integer,
        true
    FROM host_software_installed_paths p
    JOIN software s ON s.id = p.software_id
    JOIN software_titles st ON st.id = s.title_id
    WHERE p.cdhash_sha256 IS NOT NULL AND p.cdhash_sha256 <> ''
    UNION ALL
    SELECT
        'bundle'::text,
        b.sha256,
        COALESCE(NULLIF(b.name, ''), NULLIF(b.bundle_id, ''), b.sha256),
        b.path,
        b.bundle_id,
        COALESCE(NULLIF(b.version_string, ''), b.version),
        b.binary_count,
        COUNT(be.executable_id)::integer,
        b.uploaded_at IS NOT NULL
    FROM santa_bundles b
    LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
    WHERE b.sha256 <> ''
    GROUP BY b.id
),
targets AS (
    SELECT
        target_type,
        identifier,
        COALESCE(NULLIF(max(name), ''), identifier) AS name,
        COALESCE(max(detail), '')::text AS detail,
        COALESCE(max(bundle_id), '')::text AS bundle_id,
        COALESCE(max(version), '')::text AS version,
        max(binary_count)::integer AS binary_count,
        max(collected_binary_count)::integer AS collected_binary_count,
        bool_or(complete) AS complete
    FROM candidate_sources
    WHERE identifier <> ''
    GROUP BY target_type, identifier
)
SELECT
    t.target_type,
    t.identifier,
    t.name,
    t.detail,
    t.bundle_id,
    t.version,
    t.binary_count,
    t.collected_binary_count,
    COUNT(r.id)::integer AS rule_count,
    t.complete
FROM targets t
LEFT JOIN santa_rules r
    ON r.rule_type::text = t.target_type AND r.identifier = t.identifier
WHERE
    (@q::text = '' OR t.identifier ILIKE '%' || @q::text || '%' OR t.name ILIKE '%' || @q::text || '%'
        OR t.detail ILIKE '%' || @q::text || '%' OR t.bundle_id ILIKE '%' || @q::text || '%')
    AND (@target_type::text = '' OR t.target_type = @target_type::text)
GROUP BY
    t.target_type,
    t.identifier,
    t.name,
    t.detail,
    t.bundle_id,
    t.version,
    t.binary_count,
    t.collected_binary_count,
    t.complete
ORDER BY
    CASE t.target_type
        WHEN 'bundle' THEN 1
        WHEN 'signingid' THEN 2
        WHEN 'teamid' THEN 3
        WHEN 'certificate' THEN 4
        WHEN 'binary' THEN 5
        WHEN 'cdhash' THEN 6
        ELSE 7
    END,
    lower(t.name),
    t.identifier
LIMIT @limit_count;
