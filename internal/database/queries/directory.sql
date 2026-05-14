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

-- name: ListDirectoryUsers :many
SELECT * FROM directory_users ORDER BY user_principal_name;

-- name: ListDirectoryGroups :many
SELECT * FROM directory_groups ORDER BY display_name;

-- name: GetDirectoryUserByUPN :one
SELECT * FROM directory_users WHERE user_principal_name = @user_principal_name;
