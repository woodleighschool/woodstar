-- name: UpsertEntraGroup :one
INSERT INTO entra_groups (
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

-- name: DeleteEntraGroupsNotIn :exec
DELETE FROM entra_groups
WHERE external_id <> ALL(@external_ids::text[]);

-- name: DeleteEntraGroupMembershipsForUser :exec
DELETE FROM entra_group_memberships
WHERE user_id = @user_id;

-- name: InsertEntraGroupMemberships :exec
INSERT INTO entra_group_memberships (user_id, group_id)
SELECT @user_id, g.id
FROM entra_groups g
WHERE g.external_id = ANY(@group_external_ids::text[])
ON CONFLICT DO NOTHING;
