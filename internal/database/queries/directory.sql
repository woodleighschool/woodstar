-- name: UpsertDirectoryGroup :one
INSERT INTO directory_groups (
    source,
    external_id,
    display_name,
    mail_nickname
)
VALUES (
    @source::directory_source,
    @external_id,
    @display_name,
    sqlc.narg(mail_nickname)
)
ON CONFLICT (source, external_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    mail_nickname = EXCLUDED.mail_nickname,
    updated_at = now()
RETURNING *;

-- name: DeleteDirectoryGroupsNotIn :exec
DELETE FROM directory_groups
WHERE source = @source::directory_source
  AND external_id <> ALL(@external_ids::text[]);

-- name: DeleteDirectoryGroupMembershipsForUser :exec
DELETE FROM directory_group_memberships
WHERE user_id = @user_id;

-- name: InsertDirectoryGroupMemberships :exec
INSERT INTO directory_group_memberships (user_id, group_id)
SELECT @user_id, g.id
FROM directory_groups g
WHERE g.source = @source::directory_source
  AND g.external_id = ANY(@group_external_ids::text[])
ON CONFLICT DO NOTHING;

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
LEFT JOIN users u ON u.id = hul.user_id AND u.deleted_at IS NULL
LEFT JOIN directory_group_memberships egm ON egm.user_id = u.id
LEFT JOIN directory_groups eg ON eg.id = egm.group_id
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
LEFT JOIN users u ON u.id = hul.user_id AND u.deleted_at IS NULL
LEFT JOIN directory_group_memberships egm ON egm.user_id = u.id
LEFT JOIN directory_groups eg ON eg.id = egm.group_id
GROUP BY pm.host_id, pm.email, pm.source, u.mail_nickname, u.name, u.department
ORDER BY pm.host_id;

-- name: ReconcileHostUserLinks :exec
INSERT INTO host_user_links (host_id, user_id, source)
SELECT host_id, user_id, 'reported_user_affinity'::host_user_link_source
FROM (
    SELECT DISTINCT ON (he.host_id)
        he.host_id,
        u.id AS user_id
    FROM host_user_affinity_mappings he
    INNER JOIN users u
        ON lower(u.email) = lower(he.email)
        OR lower(COALESCE(u.user_principal_name, '')) = lower(he.email)
    WHERE he.source::text = ANY(@affinity_sources::text[])
      AND u.deleted_at IS NULL
    ORDER BY he.host_id, CASE he.source
        WHEN 'orbit_profile' THEN 0
        WHEN 'santa_primary_user' THEN 0
        ELSE 10
    END, he.source
) preferred
ON CONFLICT (host_id) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    source = 'reported_user_affinity',
    updated_at = now()
WHERE host_user_links.source <> 'manual';
