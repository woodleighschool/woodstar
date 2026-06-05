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
