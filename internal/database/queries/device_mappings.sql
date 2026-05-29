-- name: UpsertHostDeviceMapping :exec
INSERT INTO host_emails (
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

-- name: DeleteHostDeviceMapping :exec
DELETE FROM host_emails
WHERE host_id = @host_id
  AND source = @source;

-- name: ListHostDeviceMappings :many
SELECT *
FROM host_emails
WHERE host_id = @host_id
ORDER BY CASE source
    WHEN 'manual' THEN 0
    WHEN 'orbit_profile' THEN 1
    WHEN 'santa_primary_user' THEN 1
    ELSE 10
END, source;

-- name: ListHostDeviceMappingsForHosts :many
SELECT *
FROM host_emails
WHERE host_id = ANY(@host_ids::bigint[])
ORDER BY host_id, CASE source
    WHEN 'manual' THEN 0
    WHEN 'orbit_profile' THEN 1
    WHEN 'santa_primary_user' THEN 1
    ELSE 10
END, source;
