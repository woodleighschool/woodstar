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
    COALESCE(du.mail_nickname, '') AS username,
    COALESCE(du.display_name, '') AS name,
    COALESCE(du.department, '') AS department,
    COALESCE(
        array_agg(dg.display_name ORDER BY lower(dg.display_name)) FILTER (WHERE dg.id IS NOT NULL),
        ARRAY[]::text[]
    )::text[] AS groups
FROM primary_mapping pm
LEFT JOIN host_directory_user hdu ON hdu.host_id = pm.host_id
LEFT JOIN directory_users du ON du.id = hdu.directory_user_id AND du.active
LEFT JOIN directory_user_groups dug ON dug.directory_user_id = du.id
LEFT JOIN directory_groups dg ON dg.id = dug.directory_group_id
GROUP BY pm.email, pm.source, du.mail_nickname, du.display_name, du.department;
