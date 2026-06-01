-- name: CreateMunkiSoftwareTitle :one
INSERT INTO munki_software_titles (
    name,
    display_name,
    description,
    category,
    developer
)
VALUES (
    @name,
    @display_name,
    @description,
    @category,
    @developer
)
RETURNING *;

-- name: ListMunkiSoftwareTitles :many
SELECT *
FROM munki_software_titles
ORDER BY lower(COALESCE(NULLIF(display_name, ''), name)), lower(name), id
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountMunkiSoftwareTitles :one
SELECT COUNT(*)::integer
FROM munki_software_titles;

-- name: GetMunkiSoftwareTitleByID :one
SELECT *
FROM munki_software_titles
WHERE id = @id;

-- name: UpdateMunkiSoftwareTitle :one
UPDATE munki_software_titles
SET
    name = @name,
    display_name = @display_name,
    description = @description,
    category = @category,
    developer = @developer,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: CreateMunkiArtifact :one
INSERT INTO munki_artifacts (
    kind,
    display_name,
    location,
    content_type,
    size_bytes,
    sha256,
    storage_key
)
VALUES (
    @kind::munki_artifact_kind,
    @display_name,
    @location,
    @content_type,
    @size_bytes,
    @sha256,
    @storage_key
)
RETURNING *;

-- name: ListMunkiArtifacts :many
SELECT *
FROM munki_artifacts
ORDER BY lower(COALESCE(NULLIF(display_name, ''), location)), lower(location), id
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountMunkiArtifacts :one
SELECT COUNT(*)::integer
FROM munki_artifacts;

-- name: GetMunkiArtifactByID :one
SELECT *
FROM munki_artifacts
WHERE id = @id;

-- name: GetMunkiArtifactByKindAndLocation :one
SELECT *
FROM munki_artifacts
WHERE kind = @kind::munki_artifact_kind
  AND location = @location;

-- name: CreateMunkiPackage :one
INSERT INTO munki_packages (
    software_id,
    name,
    version,
    display_name,
    description,
    category,
    developer,
    metadata,
    installer_artifact_id,
    eligible
)
VALUES (
    @software_id,
    @name,
    @version,
    @display_name,
    @description,
    @category,
    @developer,
    @metadata::jsonb,
    sqlc.narg(installer_artifact_id)::bigint,
    @eligible
)
RETURNING *;

-- name: GetMunkiPackageByID :one
SELECT
    p.*,
    s.name AS software_name,
    s.display_name AS software_display_name,
    art.location AS installer_artifact_location
FROM munki_packages p
JOIN munki_software_titles s ON s.id = p.software_id
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
WHERE p.id = @id;

-- name: CreateMunkiDeployment :one
INSERT INTO munki_deployments (
    package_id,
    intent,
    position,
    all_hosts
)
VALUES (
    @package_id,
    @intent::munki_deployment_intent,
    (
        SELECT COALESCE(MAX(d.position) + 1, 0)
        FROM munki_deployments d
        JOIN munki_packages p ON p.id = d.package_id
        WHERE p.software_id = (
            SELECT software_id
            FROM munki_packages
            WHERE id = @package_id
        )
    ),
    @all_hosts
)
RETURNING *;

-- name: InsertMunkiDeploymentIncludeLabels :exec
INSERT INTO munki_deployment_include_labels (
    deployment_id,
    label_id
)
SELECT @deployment_id, unnest(@label_ids::bigint[]);

-- name: InsertMunkiDeploymentExcludeLabels :exec
INSERT INTO munki_deployment_exclude_labels (
    deployment_id,
    label_id
)
SELECT @deployment_id, unnest(@label_ids::bigint[]);

-- name: InsertMunkiDeploymentIncludeHosts :exec
INSERT INTO munki_deployment_include_hosts (
    deployment_id,
    host_id
)
SELECT @deployment_id, unnest(@host_ids::bigint[]);

-- name: InsertMunkiDeploymentExcludeHosts :exec
INSERT INTO munki_deployment_exclude_hosts (
    deployment_id,
    host_id
)
SELECT @deployment_id, unnest(@host_ids::bigint[]);

-- name: ListMunkiDeploymentIncludeLabelIDs :many
SELECT label_id
FROM munki_deployment_include_labels
WHERE deployment_id = @deployment_id
ORDER BY label_id;

-- name: ListMunkiDeploymentExcludeLabelIDs :many
SELECT label_id
FROM munki_deployment_exclude_labels
WHERE deployment_id = @deployment_id
ORDER BY label_id;

-- name: ListMunkiDeploymentIncludeHostIDs :many
SELECT host_id
FROM munki_deployment_include_hosts
WHERE deployment_id = @deployment_id
ORDER BY host_id;

-- name: ListMunkiDeploymentExcludeHostIDs :many
SELECT host_id
FROM munki_deployment_exclude_hosts
WHERE deployment_id = @deployment_id
ORDER BY host_id;

-- name: ListMunkiDeploymentScopeIDs :many
SELECT deployment_id, 'include_label' AS scope, label_id AS id
FROM munki_deployment_include_labels
WHERE deployment_id = ANY(@deployment_ids::bigint[])
UNION ALL
SELECT deployment_id, 'exclude_label' AS scope, label_id AS id
FROM munki_deployment_exclude_labels
WHERE deployment_id = ANY(@deployment_ids::bigint[])
UNION ALL
SELECT deployment_id, 'include_host' AS scope, host_id AS id
FROM munki_deployment_include_hosts
WHERE deployment_id = ANY(@deployment_ids::bigint[])
UNION ALL
SELECT deployment_id, 'exclude_host' AS scope, host_id AS id
FROM munki_deployment_exclude_hosts
WHERE deployment_id = ANY(@deployment_ids::bigint[])
ORDER BY deployment_id, scope, id;

-- name: ListMunkiDeploymentIDsBySoftware :many
SELECT d.id
FROM munki_deployments d
JOIN munki_packages p ON p.id = d.package_id
WHERE p.software_id = @software_id
ORDER BY d.position, d.id;

-- name: SetMunkiDeploymentPositions :exec
UPDATE munki_deployments d
SET position = -ordered.position
FROM unnest(@ordered_ids::bigint[]) WITH ORDINALITY AS ordered(id, position)
JOIN munki_packages p ON TRUE
WHERE d.id = ordered.id
  AND p.id = d.package_id
  AND p.software_id = @software_id;

-- name: NormalizeMunkiDeploymentPositions :exec
UPDATE munki_deployments d
SET position = -position - 1
FROM munki_packages p
WHERE p.id = d.package_id AND p.software_id = @software_id;

-- name: ListEffectiveMunkiPackagesForHost :many
SELECT
    d.id AS deployment_id,
    d.intent,
    d.position,
    p.id AS package_id,
    p.software_id,
    p.name,
    p.version,
    p.display_name,
    p.description,
    p.category,
    p.developer,
    p.metadata,
    p.installer_artifact_id,
    art.location AS installer_artifact_location,
    CASE
        WHEN EXISTS (
            SELECT 1
            FROM munki_deployment_include_hosts ih
            WHERE ih.deployment_id = d.id AND ih.host_id = @host_id
        ) THEN 30
        WHEN EXISTS (
            SELECT 1
            FROM munki_deployment_include_labels il
            JOIN label_membership lm ON lm.label_id = il.label_id
            WHERE il.deployment_id = d.id AND lm.host_id = @host_id
        ) THEN 20
        WHEN d.all_hosts THEN 10
        ELSE 0
    END AS scope_rank
FROM munki_deployments d
JOIN munki_packages p ON p.id = d.package_id
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
WHERE p.eligible
  AND (
    d.all_hosts
    OR EXISTS (
      SELECT 1
      FROM munki_deployment_include_hosts ih
      WHERE ih.deployment_id = d.id AND ih.host_id = @host_id
    )
    OR EXISTS (
      SELECT 1
      FROM munki_deployment_include_labels il
      JOIN label_membership lm ON lm.label_id = il.label_id
      WHERE il.deployment_id = d.id AND lm.host_id = @host_id
    )
  )
  AND NOT EXISTS (
    SELECT 1
    FROM munki_deployment_exclude_hosts eh
    WHERE eh.deployment_id = d.id AND eh.host_id = @host_id
  )
  AND NOT EXISTS (
    SELECT 1
    FROM munki_deployment_exclude_labels el
    JOIN label_membership lm ON lm.label_id = el.label_id
    WHERE el.deployment_id = d.id AND lm.host_id = @host_id
  )
ORDER BY lower(p.name), d.position, d.id;

-- name: UpsertMunkiHostStatus :exec
INSERT INTO munki_host_status (
    host_id,
    version,
    manifest_name,
    success,
    errors,
    warnings,
    problem_installs,
    run_started_at,
    run_ended_at,
    last_seen_at
)
VALUES (
    @host_id,
    @version,
    @manifest_name,
    sqlc.narg(success)::boolean,
    @errors,
    @warnings,
    @problem_installs,
    @run_started_at,
    @run_ended_at,
    now()
)
ON CONFLICT (host_id) DO UPDATE SET
    version = EXCLUDED.version,
    manifest_name = EXCLUDED.manifest_name,
    success = EXCLUDED.success,
    errors = EXCLUDED.errors,
    warnings = EXCLUDED.warnings,
    problem_installs = EXCLUDED.problem_installs,
    run_started_at = EXCLUDED.run_started_at,
    run_ended_at = EXCLUDED.run_ended_at,
    last_seen_at = now(),
    updated_at = now();

-- name: ClearMunkiHostStatus :exec
DELETE FROM munki_host_status
WHERE host_id = @host_id;

-- name: DeleteMunkiHostItems :exec
DELETE FROM munki_host_items
WHERE host_id = @host_id;

-- name: InsertMunkiHostItem :exec
INSERT INTO munki_host_items (
    host_id,
    name,
    installed,
    installed_version,
    run_ended_at,
    last_seen_at
)
VALUES (
    @host_id,
    @name,
    @installed,
    @installed_version,
    @run_ended_at,
    now()
)
ON CONFLICT (host_id, name) DO UPDATE SET
    installed = EXCLUDED.installed,
    installed_version = EXCLUDED.installed_version,
    run_ended_at = EXCLUDED.run_ended_at,
    last_seen_at = now(),
    updated_at = now();

-- name: ListMunkiHostItems :many
SELECT *
FROM munki_host_items
WHERE host_id = @host_id
ORDER BY lower(name), name;

-- name: GetMunkiHostStatus :one
SELECT *
FROM munki_host_status
WHERE host_id = @host_id;
