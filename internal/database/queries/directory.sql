-- name: UpsertDirectoryUser :one
INSERT INTO directory_users (
    external_id,
    user_principal_name,
    mail,
    mail_nickname,
    display_name,
    given_name,
    family_name,
    department,
    active,
    last_synced_at
)
VALUES (
    @external_id,
    @user_principal_name,
    sqlc.narg(mail),
    sqlc.narg(mail_nickname),
    @display_name,
    sqlc.narg(given_name),
    sqlc.narg(family_name),
    sqlc.narg(department),
    @active,
    @last_synced_at
)
ON CONFLICT (external_id) DO UPDATE SET
    user_principal_name = EXCLUDED.user_principal_name,
    mail = EXCLUDED.mail,
    mail_nickname = EXCLUDED.mail_nickname,
    display_name = EXCLUDED.display_name,
    given_name = EXCLUDED.given_name,
    family_name = EXCLUDED.family_name,
    department = EXCLUDED.department,
    active = EXCLUDED.active,
    last_synced_at = EXCLUDED.last_synced_at,
    updated_at = now()
RETURNING *;

-- name: UpsertDirectoryGroup :one
INSERT INTO directory_groups (
    external_id,
    display_name,
    mail_nickname,
    last_synced_at
)
VALUES (
    @external_id,
    @display_name,
    sqlc.narg(mail_nickname),
    @last_synced_at
)
ON CONFLICT (external_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    mail_nickname = EXCLUDED.mail_nickname,
    last_synced_at = EXCLUDED.last_synced_at,
    updated_at = now()
RETURNING *;

-- name: DeleteDirectoryUsersNotIn :exec
DELETE FROM directory_users
WHERE external_id <> ALL(@external_ids::text[]);

-- name: DeleteDirectoryGroupsNotIn :exec
DELETE FROM directory_groups
WHERE external_id <> ALL(@external_ids::text[]);

-- name: ReplaceDirectoryUserGroups :exec
WITH cleared AS (
    DELETE FROM directory_user_groups
    WHERE directory_user_id = @directory_user_id
    RETURNING 1
)
INSERT INTO directory_user_groups (directory_user_id, directory_group_id)
SELECT @directory_user_id, g.id
FROM directory_groups g
WHERE g.external_id = ANY(@group_external_ids::text[])
ON CONFLICT DO NOTHING;

-- name: LoadHostUserAffinity :one
WITH primary_mapping AS (
    SELECT he.host_id, he.email, he.source::text AS source
    FROM host_emails he
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
