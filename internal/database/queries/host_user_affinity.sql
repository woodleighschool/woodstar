-- name: UpsertHostUserAffinityMapping :exec
INSERT INTO host_user_affinity_mappings (
    host_id,
    email,
    source
)
VALUES (
    @host_id,
    @email,
    @source
)
ON CONFLICT (host_id, source) DO UPDATE SET
    email = EXCLUDED.email,
    updated_at = now();

-- name: DeleteHostUserAffinityMapping :exec
DELETE FROM host_user_affinity_mappings
WHERE host_id = @host_id
  AND source = @source;

-- name: ListHostUserAffinityMappings :many
SELECT *
FROM host_user_affinity_mappings
WHERE host_id = @host_id
ORDER BY CASE source
    WHEN 'manual' THEN 0
    WHEN 'orbit_profile' THEN 1
    WHEN 'santa_primary_user' THEN 1
    ELSE 10
END, source;

-- name: ListHostUserAffinityMappingsForHosts :many
SELECT *
FROM host_user_affinity_mappings
WHERE host_id = ANY(@host_ids::bigint[])
ORDER BY host_id, CASE source
    WHEN 'manual' THEN 0
    WHEN 'orbit_profile' THEN 1
    WHEN 'santa_primary_user' THEN 1
    ELSE 10
END, source;

-- name: LoadHostUserAffinityPrimary :one
WITH primary_mapping AS (
    SELECT he.host_id, he.email, he.source::text AS source
    FROM host_user_affinity_mappings he
    WHERE he.host_id = @affinity_host_id
    ORDER BY CASE he.source
        WHEN 'manual' THEN 0
        WHEN 'orbit_profile' THEN 1
        WHEN 'santa_primary_user' THEN 1
        ELSE 10
    END, he.source
    LIMIT 1
)
SELECT
    pm.email,
    pm.source,
    COALESCE(u.mail_nickname, '') AS username,
    COALESCE(u.name, '') AS name,
    COALESCE(u.department, '') AS department,
    COALESCE(
        array_agg(eg.display_name ORDER BY lower(eg.display_name)) FILTER (WHERE eg.id IS NOT NULL),
        ARRAY[]::text[]
    )::text[] AS groups
FROM primary_mapping pm
LEFT JOIN host_user_links hul ON hul.host_id = pm.host_id
LEFT JOIN users u ON u.id = hul.user_id AND u.active
LEFT JOIN entra_group_memberships egm ON egm.user_id = u.id
LEFT JOIN entra_groups eg ON eg.id = egm.group_id
GROUP BY pm.email, pm.source, u.mail_nickname, u.name, u.department;

-- name: ListHostUserAffinityPrimaries :many
WITH primary_mapping AS (
    SELECT DISTINCT ON (he.host_id) he.host_id, he.email, he.source::text AS source
    FROM host_user_affinity_mappings he
    WHERE he.host_id = ANY(@host_ids::bigint[])
    ORDER BY he.host_id, CASE he.source
        WHEN 'manual' THEN 0
        WHEN 'orbit_profile' THEN 1
        WHEN 'santa_primary_user' THEN 1
        ELSE 10
    END, he.source
)
SELECT
    pm.host_id,
    pm.email,
    pm.source,
    COALESCE(u.mail_nickname, '') AS username,
    COALESCE(u.name, '') AS name,
    COALESCE(u.department, '') AS department,
    COALESCE(
        array_agg(eg.display_name ORDER BY lower(eg.display_name)) FILTER (WHERE eg.id IS NOT NULL),
        ARRAY[]::text[]
    )::text[] AS groups
FROM primary_mapping pm
LEFT JOIN host_user_links hul ON hul.host_id = pm.host_id
LEFT JOIN users u ON u.id = hul.user_id AND u.active
LEFT JOIN entra_group_memberships egm ON egm.user_id = u.id
LEFT JOIN entra_groups eg ON eg.id = egm.group_id
GROUP BY pm.host_id, pm.email, pm.source, u.mail_nickname, u.name, u.department
ORDER BY pm.host_id;
