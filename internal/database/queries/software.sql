-- name: DeleteHostSoftware :exec
DELETE FROM host_software
WHERE host_id = @host_id;

-- name: UpsertSoftwareTitle :one
INSERT INTO software (
    name,
    version,
    source,
    bundle_identifier
)
VALUES (
    @name,
    @version,
    @source,
    @bundle_identifier
)
ON CONFLICT (name, version, source, bundle_identifier) DO UPDATE SET
    name = EXCLUDED.name
RETURNING id;

-- name: UpsertHostSoftware :exec
INSERT INTO host_software (
    host_id,
    software_id,
    last_seen_at,
    last_opened_at
)
VALUES (
    @host_id,
    @software_id,
    now(),
    @last_opened_at
)
ON CONFLICT (host_id, software_id) DO UPDATE SET
    last_seen_at = now(),
    last_opened_at = EXCLUDED.last_opened_at;

-- name: ListSoftwareForHost :many
SELECT
    software.id,
    software.name,
    software.version,
    software.source,
    software.bundle_identifier,
    host_software.last_seen_at,
    host_software.last_opened_at
FROM host_software
JOIN software ON software.id = host_software.software_id
WHERE host_software.host_id = @host_id
ORDER BY lower(software.name), software.version, software.source;

-- name: ListSoftwareTitlesWithHostCount :many
SELECT
    software.id,
    software.name,
    software.version,
    software.source,
    software.bundle_identifier,
    count(host_software.host_id)::integer AS host_count,
    software.created_at
FROM software
LEFT JOIN host_software ON host_software.software_id = software.id
GROUP BY software.id
ORDER BY count(host_software.host_id) DESC, lower(software.name), software.version;
