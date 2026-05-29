-- name: ListSoftwareReferenceFacts :many
SELECT
    COALESCE(s.bundle_identifier, '') AS bundle_identifier,
    COALESCE(paths.installed_path, '') AS installed_path,
    COALESCE(paths.executable_path, '') AS executable_path,
    COALESCE(paths.executable_sha256, '') AS executable_sha256,
    COALESCE(paths.cdhash_sha256, '') AS cdhash_sha256,
    COALESCE(paths.team_identifier, '') AS team_identifier
FROM software_titles st
LEFT JOIN software s ON s.title_id = st.id
LEFT JOIN host_software_installed_paths paths ON paths.software_id = s.id
WHERE st.id = @software_title_id;

-- name: GetSoftwareReferenceExecutionCounts :one
WITH matched_executables AS (
    SELECT DISTINCT e.id
    FROM santa_executables e
    WHERE
        e.sha256 = ANY(@executable_sha256s::text[])
        OR e.cdhash = ANY(@cdhashes::text[])
        OR e.team_id = ANY(@team_ids::text[])
        OR e.signing_id = ANY(@signing_ids::text[])
        OR e.file_bundle_id = ANY(@bundle_ids::text[])
        OR e.file_bundle_path = ANY(@paths::text[])
        OR EXISTS (
            SELECT 1
            FROM santa_execution_events ee
            WHERE ee.executable_id = e.id AND ee.file_path = ANY(@paths::text[])
        )
)
SELECT
    COUNT(DISTINCT ee.id)::integer AS execution_count,
    (COUNT(DISTINCT ee.id) FILTER (WHERE ee.decision::text LIKE 'block_%'))::integer AS block_count
FROM santa_execution_events ee
LEFT JOIN matched_executables me ON me.id = ee.executable_id
WHERE me.id IS NOT NULL OR ee.file_path = ANY(@paths::text[]);

-- name: ListSoftwareReferenceBundles :many
WITH matched_executables AS (
    SELECT DISTINCT e.id
    FROM santa_executables e
    WHERE
        e.sha256 = ANY(@executable_sha256s::text[])
        OR e.cdhash = ANY(@cdhashes::text[])
        OR e.team_id = ANY(@team_ids::text[])
        OR e.signing_id = ANY(@signing_ids::text[])
        OR e.file_bundle_id = ANY(@bundle_ids::text[])
        OR e.file_bundle_path = ANY(@paths::text[])
        OR EXISTS (
            SELECT 1
            FROM santa_execution_events ee
            WHERE ee.executable_id = e.id AND ee.file_path = ANY(@paths::text[])
        )
),
matched_bundles AS (
    SELECT DISTINCT b.id
    FROM santa_bundles b
    LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
    LEFT JOIN matched_executables me ON me.id = be.executable_id
    WHERE b.bundle_id = ANY(@bundle_ids::text[]) OR me.id IS NOT NULL
)
SELECT
    b.sha256,
    b.bundle_id,
    b.name,
    b.path,
    b.version,
    b.version_string,
    b.binary_count,
    COUNT(be.executable_id)::integer AS collected_binary_count,
    b.hash_millis,
    b.uploaded_at,
    (b.uploaded_at IS NOT NULL)::boolean AS complete
FROM matched_bundles mb
JOIN santa_bundles b ON b.id = mb.id
LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
GROUP BY b.id
ORDER BY lower(COALESCE(NULLIF(b.name, ''), b.bundle_id, b.sha256)), b.sha256;

-- name: ListSoftwareReferenceExecutables :many
WITH matched_executables AS (
    SELECT DISTINCT e.id
    FROM santa_executables e
    WHERE
        e.sha256 = ANY(@executable_sha256s::text[])
        OR e.cdhash = ANY(@cdhashes::text[])
        OR e.team_id = ANY(@team_ids::text[])
        OR e.signing_id = ANY(@signing_ids::text[])
        OR e.file_bundle_id = ANY(@bundle_ids::text[])
        OR e.file_bundle_path = ANY(@paths::text[])
        OR EXISTS (
            SELECT 1
            FROM santa_execution_events ee
            WHERE ee.executable_id = e.id AND ee.file_path = ANY(@paths::text[])
        )
)
SELECT
    e.sha256,
    e.file_name,
    e.file_bundle_id,
    e.file_bundle_name,
    COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version) AS bundle_version,
    e.signing_id,
    e.team_id,
    e.cdhash,
    COUNT(ee.id)::integer AS execution_count,
    (COUNT(ee.id) FILTER (WHERE ee.decision::text LIKE 'block_%'))::integer AS block_count
FROM matched_executables me
JOIN santa_executables e ON e.id = me.id
LEFT JOIN santa_execution_events ee ON ee.executable_id = e.id
GROUP BY e.id
ORDER BY lower(COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.sha256)), e.sha256;

-- name: ListSoftwareReferenceSigningIdentities :many
WITH matched_executables AS (
    SELECT DISTINCT e.id
    FROM santa_executables e
    WHERE
        e.sha256 = ANY(@executable_sha256s::text[])
        OR e.cdhash = ANY(@cdhashes::text[])
        OR e.team_id = ANY(@team_ids::text[])
        OR e.signing_id = ANY(@signing_ids::text[])
        OR e.file_bundle_id = ANY(@bundle_ids::text[])
        OR e.file_bundle_path = ANY(@paths::text[])
        OR EXISTS (
            SELECT 1
            FROM santa_execution_events ee
            WHERE ee.executable_id = e.id AND ee.file_path = ANY(@paths::text[])
        )
),
identities AS (
    SELECT
        'teamid'::text AS target_type,
        e.team_id AS identifier,
        COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.team_id) AS name,
        e.id AS executable_id
    FROM matched_executables me
    JOIN santa_executables e ON e.id = me.id
    WHERE e.team_id <> ''
    UNION ALL
    SELECT
        'signingid'::text,
        e.signing_id,
        COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.signing_id),
        e.id
    FROM matched_executables me
    JOIN santa_executables e ON e.id = me.id
    WHERE e.signing_id <> ''
    UNION ALL
    SELECT
        'cdhash'::text,
        e.cdhash,
        COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.cdhash),
        e.id
    FROM matched_executables me
    JOIN santa_executables e ON e.id = me.id
    WHERE e.cdhash <> ''
)
SELECT
    i.target_type,
    i.identifier,
    COALESCE(NULLIF(MAX(i.name), ''), i.identifier) AS name,
    COUNT(DISTINCT i.executable_id)::integer AS executable_count,
    COUNT(DISTINCT r.id)::integer AS rule_count
FROM identities i
LEFT JOIN santa_rules r ON r.rule_type::text = i.target_type AND r.identifier = i.identifier
GROUP BY i.target_type, i.identifier
ORDER BY i.target_type, lower(COALESCE(NULLIF(MAX(i.name), ''), i.identifier)), i.identifier;

-- name: ListSoftwareReferenceCertificates :many
WITH matched_executables AS (
    SELECT DISTINCT e.id
    FROM santa_executables e
    WHERE
        e.sha256 = ANY(@executable_sha256s::text[])
        OR e.cdhash = ANY(@cdhashes::text[])
        OR e.team_id = ANY(@team_ids::text[])
        OR e.signing_id = ANY(@signing_ids::text[])
        OR e.file_bundle_id = ANY(@bundle_ids::text[])
        OR e.file_bundle_path = ANY(@paths::text[])
        OR EXISTS (
            SELECT 1
            FROM santa_execution_events ee
            WHERE ee.executable_id = e.id AND ee.file_path = ANY(@paths::text[])
        )
),
matched_certificates AS (
    SELECT DISTINCT c.id
    FROM matched_executables me
    JOIN santa_executable_signing_chains esc ON esc.executable_id = me.id
    JOIN santa_signing_chain_entries sce ON sce.signing_chain_id = esc.signing_chain_id
    JOIN santa_certificates c ON c.id = sce.certificate_id
)
SELECT
    c.sha256,
    c.common_name,
    c.organization,
    c.organizational_unit,
    c.valid_from,
    c.valid_until,
    COUNT(r.id)::integer AS rule_count
FROM matched_certificates mc
JOIN santa_certificates c ON c.id = mc.id
LEFT JOIN santa_rules r ON r.rule_type = 'certificate' AND r.identifier = c.sha256
GROUP BY c.id
ORDER BY lower(COALESCE(NULLIF(c.common_name, ''), c.sha256)), c.sha256;

-- name: ListSoftwareReferenceRules :many
WITH matched_executables AS (
    SELECT DISTINCT e.id
    FROM santa_executables e
    WHERE
        e.sha256 = ANY(@executable_sha256s::text[])
        OR e.cdhash = ANY(@cdhashes::text[])
        OR e.team_id = ANY(@team_ids::text[])
        OR e.signing_id = ANY(@signing_ids::text[])
        OR e.file_bundle_id = ANY(@bundle_ids::text[])
        OR e.file_bundle_path = ANY(@paths::text[])
        OR EXISTS (
            SELECT 1
            FROM santa_execution_events ee
            WHERE ee.executable_id = e.id AND ee.file_path = ANY(@paths::text[])
        )
),
matched_bundles AS (
    SELECT DISTINCT b.id, b.sha256
    FROM santa_bundles b
    LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
    LEFT JOIN matched_executables me ON me.id = be.executable_id
    WHERE b.bundle_id = ANY(@bundle_ids::text[]) OR me.id IS NOT NULL
),
matched_certificates AS (
    SELECT DISTINCT c.sha256
    FROM matched_executables me
    JOIN santa_executable_signing_chains esc ON esc.executable_id = me.id
    JOIN santa_signing_chain_entries sce ON sce.signing_chain_id = esc.signing_chain_id
    JOIN santa_certificates c ON c.id = sce.certificate_id
),
matched_targets AS (
    SELECT 'binary'::text AS target_type, unnest(@executable_sha256s::text[]) AS identifier
    UNION
    SELECT 'cdhash'::text, unnest(@cdhashes::text[])
    UNION
    SELECT 'teamid'::text, unnest(@team_ids::text[])
    UNION
    SELECT 'signingid'::text, unnest(@signing_ids::text[])
    UNION
    SELECT 'binary'::text, e.sha256
    FROM matched_executables me
    JOIN santa_executables e ON e.id = me.id
    WHERE e.sha256 <> ''
    UNION
    SELECT 'cdhash'::text, e.cdhash
    FROM matched_executables me
    JOIN santa_executables e ON e.id = me.id
    WHERE e.cdhash <> ''
    UNION
    SELECT 'teamid'::text, e.team_id
    FROM matched_executables me
    JOIN santa_executables e ON e.id = me.id
    WHERE e.team_id <> ''
    UNION
    SELECT 'signingid'::text, e.signing_id
    FROM matched_executables me
    JOIN santa_executables e ON e.id = me.id
    WHERE e.signing_id <> ''
    UNION
    SELECT 'bundle'::text, sha256
    FROM matched_bundles
    UNION
    SELECT 'certificate'::text, sha256
    FROM matched_certificates
)
SELECT
    r.id,
    r.rule_type::text AS rule_type,
    r.identifier,
    r.name,
    r.custom_message,
    r.custom_url
FROM santa_rules r
WHERE EXISTS (
    SELECT 1
    FROM matched_targets mt
    WHERE mt.target_type = r.rule_type::text AND mt.identifier = r.identifier
)
ORDER BY r.rule_type::text, lower(COALESCE(NULLIF(r.name, ''), r.identifier)), r.identifier;
