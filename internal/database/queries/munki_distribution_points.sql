-- name: CreateMunkiDistributionPoint :one
INSERT INTO munki_distribution_points (
    name,
    enabled,
    position,
    client_cidrs,
    client_base_url,
    "key"
)
VALUES (
    @name,
    @enabled,
    (SELECT COALESCE(MAX(position) + 1, 0) FROM munki_distribution_points),
    @client_cidrs::text[],
    @client_base_url,
    @key
)
RETURNING *;

-- name: GetMunkiDistributionPointByID :one
SELECT *
FROM munki_distribution_points
WHERE id = @id;

-- name: GetMunkiDistributionPointByKey :one
SELECT *
FROM munki_distribution_points
WHERE "key" = @key;

-- name: UpdateMunkiDistributionPoint :one
UPDATE munki_distribution_points
SET
    name = @name,
    enabled = @enabled,
    client_cidrs = @client_cidrs::text[],
    client_base_url = @client_base_url,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: RotateMunkiDistributionPointKey :one
UPDATE munki_distribution_points
SET "key" = @key,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteMunkiDistributionPoint :one
DELETE FROM munki_distribution_points
WHERE id = @id
RETURNING id;

-- name: ListMunkiDistributionPointIDsByPosition :many
SELECT id
FROM munki_distribution_points
ORDER BY position, id;

-- name: SetMunkiDistributionPointPositions :exec
UPDATE munki_distribution_points c
SET position = -ordered.position
FROM unnest(@ordered_ids::bigint[]) WITH ORDINALITY AS ordered(id, position)
WHERE c.id = ordered.id;

-- name: NormalizeMunkiDistributionPointPositions :exec
UPDATE munki_distribution_points
SET position = -position - 1;

-- name: ListEligibleMunkiDistributionPointsForClient :many
SELECT c.*
FROM munki_distribution_points c
WHERE c.enabled
  AND c.client_base_url <> ''
  AND @client_ip::inet <<= ANY (c.client_cidrs::inet[])
  AND EXISTS (
      SELECT 1
      FROM munki_distribution_package_states s
      JOIN munki_packages p ON p.id = s.package_id
      JOIN storage_objects o ON o.id = p.installer_object_id
      WHERE s.distribution_point_id = c.id
        AND s.package_id = @package_id
        AND s.status = 'current'
        AND o.available_at IS NOT NULL
        AND o.sha256 = s.reported_sha256
  )
ORDER BY c.position, c.id;

-- name: UpsertMunkiDistributionPackageState :exec
INSERT INTO munki_distribution_package_states (
    distribution_point_id,
    package_id,
    status,
    reported_sha256,
    error
)
VALUES (
    @distribution_point_id,
    @package_id,
    @status,
    @reported_sha256,
    @error
)
ON CONFLICT (distribution_point_id, package_id) DO UPDATE
SET status = EXCLUDED.status,
    reported_sha256 = EXCLUDED.reported_sha256,
    error = EXCLUDED.error,
    updated_at = now();

-- name: DeleteMunkiDistributionPackageStatesNotIn :exec
DELETE FROM munki_distribution_package_states
WHERE distribution_point_id = @distribution_point_id
  AND package_id <> ALL (@package_ids::bigint[]);

-- name: ListMunkiDistributionPackageStates :many
SELECT
    p.id AS package_id,
    sw.name AS display_name,
    p.version,
    sw.icon_object_id,
    CASE
        WHEN s.package_id IS NULL THEN 'pending'
        WHEN s.status = 'error' THEN 'error'
        WHEN o.sha256 = s.reported_sha256 THEN 'current'
        ELSE 'syncing'
    END::text AS status,
    COALESCE(s.error, '') AS error
FROM munki_packages p
JOIN munki_software sw ON sw.id = p.software_id
JOIN storage_objects o ON o.id = p.installer_object_id
LEFT JOIN munki_distribution_package_states s
    ON s.package_id = p.id
    AND s.distribution_point_id = @distribution_point_id
WHERE o.available_at IS NOT NULL
  AND o.sha256 IS NOT NULL
  AND o.size_bytes IS NOT NULL
ORDER BY sw.name, p.version;

-- name: GetMunkiPackageInstallerObject :one
SELECT
    sqlc.embed(o)
FROM munki_packages p
JOIN storage_objects o ON o.id = p.installer_object_id
WHERE p.id = @package_id
  AND o.available_at IS NOT NULL;

-- name: ListDesiredMunkiPackages :many
SELECT
    p.id AS package_id,
    s.name AS display_name,
    p.version,
    o.filename,
    o.sha256,
    o.size_bytes
FROM munki_packages p
JOIN munki_software s ON s.id = p.software_id
JOIN storage_objects o ON o.id = p.installer_object_id
WHERE o.available_at IS NOT NULL
  AND o.sha256 IS NOT NULL
  AND o.size_bytes IS NOT NULL
ORDER BY p.id;
