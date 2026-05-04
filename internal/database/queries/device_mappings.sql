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
SELECT
    id,
    host_id,
    email,
    source,
    created_at,
    updated_at
FROM host_emails
WHERE host_id = @host_id
ORDER BY source;
