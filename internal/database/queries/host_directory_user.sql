-- name: GetHostDirectoryUser :one
SELECT *
FROM host_directory_user
WHERE host_id = @host_id;

-- ManualSetHostDirectoryUser writes an admin-asserted link; manual wins over
-- any prior source.
-- name: ManualSetHostDirectoryUser :one
INSERT INTO host_directory_user (host_id, directory_user_id, source)
VALUES (@host_id, @directory_user_id, 'manual')
ON CONFLICT (host_id) DO UPDATE SET
    directory_user_id = EXCLUDED.directory_user_id,
    source = 'manual',
    updated_at = now()
RETURNING *;

-- name: DeleteHostDirectoryUser :exec
DELETE FROM host_directory_user
WHERE host_id = @host_id;

-- ReconcileHostDirectoryLinks joins host_emails (Orbit-provided) to
-- directory_users by UPN and inserts mdm_email links. Existing manual links
-- are preserved by the WHERE clause; existing mdm_email links update if the
-- matched directory user has changed.
-- name: ReconcileHostDirectoryLinks :exec
INSERT INTO host_directory_user (host_id, directory_user_id, source)
SELECT he.host_id, du.id, 'mdm_email'
FROM host_emails he
INNER JOIN directory_users du ON du.user_principal_name = he.email
WHERE he.source = @mdm_source
ON CONFLICT (host_id) DO UPDATE SET
    directory_user_id = EXCLUDED.directory_user_id,
    source = 'mdm_email',
    updated_at = now()
WHERE host_directory_user.source <> 'manual';
