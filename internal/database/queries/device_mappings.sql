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

-- name: ListHostDeviceMappings :many
SELECT *
FROM host_emails
WHERE host_id = @host_id
ORDER BY source;

-- name: ListHostDeviceMappingsForHosts :many
SELECT *
FROM host_emails
WHERE host_id = ANY(@host_ids::bigint[])
ORDER BY host_id, source;
