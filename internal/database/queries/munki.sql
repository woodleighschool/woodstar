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

-- name: GetMunkiSoftwareTitleByName :one
SELECT *
FROM munki_software_titles
WHERE name = @name;

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

-- name: UpsertMunkiArtifact :one
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
ON CONFLICT (kind, location) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    content_type = EXCLUDED.content_type,
    size_bytes = EXCLUDED.size_bytes,
    sha256 = EXCLUDED.sha256,
    storage_key = EXCLUDED.storage_key,
    updated_at = now()
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
    installer_type,
    uninstall_method,
    restart_action,
    minimum_munki_version,
    minimum_os_version,
    maximum_os_version,
    supported_architectures,
    blocking_applications,
    requires,
    update_for,
    unattended_install,
    unattended_uninstall,
    uninstallable,
    on_demand,
    precache,
    icon_name,
    icon_hash,
    extra_pkginfo,
    installer_artifact_id,
    icon_artifact_id,
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
    @installer_type,
    @uninstall_method,
    @restart_action,
    @minimum_munki_version,
    @minimum_os_version,
    @maximum_os_version,
    @supported_architectures::text[],
    @blocking_applications::text[],
    @requires::text[],
    @update_for::text[],
    @unattended_install,
    @unattended_uninstall,
    @uninstallable,
    @on_demand,
    @precache,
    @icon_name,
    @icon_hash,
    @extra_pkginfo::jsonb,
    sqlc.narg(installer_artifact_id)::bigint,
    sqlc.narg(icon_artifact_id)::bigint,
    @eligible
)
RETURNING *;

-- name: UpsertMunkiPackage :one
INSERT INTO munki_packages (
    software_id,
    name,
    version,
    display_name,
    description,
    category,
    developer,
    installer_type,
    uninstall_method,
    restart_action,
    minimum_munki_version,
    minimum_os_version,
    maximum_os_version,
    supported_architectures,
    blocking_applications,
    requires,
    update_for,
    unattended_install,
    unattended_uninstall,
    uninstallable,
    on_demand,
    precache,
    icon_name,
    icon_hash,
    extra_pkginfo,
    installer_artifact_id,
    icon_artifact_id,
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
    @installer_type,
    @uninstall_method,
    @restart_action,
    @minimum_munki_version,
    @minimum_os_version,
    @maximum_os_version,
    @supported_architectures::text[],
    @blocking_applications::text[],
    @requires::text[],
    @update_for::text[],
    @unattended_install,
    @unattended_uninstall,
    @uninstallable,
    @on_demand,
    @precache,
    @icon_name,
    @icon_hash,
    @extra_pkginfo::jsonb,
    sqlc.narg(installer_artifact_id)::bigint,
    sqlc.narg(icon_artifact_id)::bigint,
    @eligible
)
ON CONFLICT (software_id, name, version) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    developer = EXCLUDED.developer,
    installer_type = EXCLUDED.installer_type,
    uninstall_method = EXCLUDED.uninstall_method,
    restart_action = EXCLUDED.restart_action,
    minimum_munki_version = EXCLUDED.minimum_munki_version,
    minimum_os_version = EXCLUDED.minimum_os_version,
    maximum_os_version = EXCLUDED.maximum_os_version,
    supported_architectures = EXCLUDED.supported_architectures,
    blocking_applications = EXCLUDED.blocking_applications,
    requires = EXCLUDED.requires,
    update_for = EXCLUDED.update_for,
    unattended_install = EXCLUDED.unattended_install,
    unattended_uninstall = EXCLUDED.unattended_uninstall,
    uninstallable = EXCLUDED.uninstallable,
    on_demand = EXCLUDED.on_demand,
    precache = EXCLUDED.precache,
    icon_name = EXCLUDED.icon_name,
    icon_hash = EXCLUDED.icon_hash,
    extra_pkginfo = EXCLUDED.extra_pkginfo,
    installer_artifact_id = EXCLUDED.installer_artifact_id,
    icon_artifact_id = EXCLUDED.icon_artifact_id,
    eligible = EXCLUDED.eligible,
    updated_at = now()
RETURNING *;

-- name: UpdateMunkiPackage :one
UPDATE munki_packages
SET
    name = @name,
    version = @version,
    display_name = @display_name,
    description = @description,
    category = @category,
    developer = @developer,
    installer_type = @installer_type,
    uninstall_method = @uninstall_method,
    restart_action = @restart_action,
    minimum_munki_version = @minimum_munki_version,
    minimum_os_version = @minimum_os_version,
    maximum_os_version = @maximum_os_version,
    supported_architectures = @supported_architectures::text[],
    blocking_applications = @blocking_applications::text[],
    requires = @requires::text[],
    update_for = @update_for::text[],
    unattended_install = @unattended_install,
    unattended_uninstall = @unattended_uninstall,
    uninstallable = @uninstallable,
    on_demand = @on_demand,
    precache = @precache,
    icon_name = @icon_name,
    icon_hash = @icon_hash,
    extra_pkginfo = @extra_pkginfo::jsonb,
    installer_artifact_id = sqlc.narg(installer_artifact_id)::bigint,
    icon_artifact_id = sqlc.narg(icon_artifact_id)::bigint,
    eligible = @eligible,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: GetMunkiPackageByID :one
SELECT
    p.*,
    s.name AS software_name,
    s.display_name AS software_display_name,
    art.location AS installer_artifact_location,
    icon.location AS icon_artifact_location
FROM munki_packages p
JOIN munki_software_titles s ON s.id = p.software_id
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
LEFT JOIN munki_artifacts icon ON icon.id = p.icon_artifact_id
WHERE p.id = @id;

-- name: CreateMunkiDeployment :one
INSERT INTO munki_deployments (
    software_id,
    action,
    self_service,
    package_selection,
    pinned_package_id,
    position,
    all_hosts
)
VALUES (
    @software_id,
    @action::munki_deployment_action,
    @self_service::munki_self_service_mode,
    @package_selection::munki_package_selection,
    sqlc.narg(pinned_package_id)::bigint,
    (
        SELECT COALESCE(MAX(d.position) + 1, 0)
        FROM munki_deployments d
        WHERE d.software_id = @software_id
    ),
    @all_hosts
)
RETURNING *;

-- name: UpdateMunkiDeployment :one
UPDATE munki_deployments
SET
    action = @action::munki_deployment_action,
    self_service = @self_service::munki_self_service_mode,
    package_selection = @package_selection::munki_package_selection,
    pinned_package_id = sqlc.narg(pinned_package_id)::bigint,
    all_hosts = @all_hosts,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteMunkiDeploymentIncludeLabels :exec
DELETE FROM munki_deployment_include_labels
WHERE deployment_id = @deployment_id;

-- name: DeleteMunkiDeploymentExcludeLabels :exec
DELETE FROM munki_deployment_exclude_labels
WHERE deployment_id = @deployment_id;

-- name: DeleteMunkiDeploymentIncludeHosts :exec
DELETE FROM munki_deployment_include_hosts
WHERE deployment_id = @deployment_id;

-- name: DeleteMunkiDeploymentExcludeHosts :exec
DELETE FROM munki_deployment_exclude_hosts
WHERE deployment_id = @deployment_id;

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
WHERE d.software_id = @software_id
ORDER BY d.position, d.id;

-- name: SetMunkiDeploymentPositions :exec
UPDATE munki_deployments d
SET position = -ordered.position
FROM unnest(@ordered_ids::bigint[]) WITH ORDINALITY AS ordered(id, position)
WHERE d.id = ordered.id
  AND d.software_id = @software_id;

-- name: NormalizeMunkiDeploymentPositions :exec
UPDATE munki_deployments d
SET position = -position - 1
WHERE d.software_id = @software_id;

-- name: ListEffectiveMunkiPackagesForHost :many
SELECT
    d.id AS deployment_id,
    d.software_id AS deployment_software_id,
    d.action,
    d.self_service,
    d.package_selection,
    d.pinned_package_id,
    d.position,
    p.id AS package_id,
    p.software_id,
    s.name AS software_name,
    COALESCE(NULLIF(s.display_name, ''), s.name) AS software_display_name,
    p.name,
    p.version,
    p.display_name,
    p.description,
    p.category,
    p.developer,
    p.installer_type,
    p.uninstall_method,
    p.restart_action,
    p.minimum_munki_version,
    p.minimum_os_version,
    p.maximum_os_version,
    p.supported_architectures,
    p.blocking_applications,
    p.requires,
    p.update_for,
    p.unattended_install,
    p.unattended_uninstall,
    p.uninstallable,
    p.on_demand,
    p.precache,
    p.icon_name,
    p.icon_hash,
    p.extra_pkginfo,
    p.installer_artifact_id,
    art.location AS installer_artifact_location,
    p.icon_artifact_id,
    icon.location AS icon_artifact_location,
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
JOIN munki_software_titles s ON s.id = d.software_id
JOIN munki_packages p ON p.software_id = d.software_id
    AND (
        (d.package_selection = 'latest_eligible' AND d.pinned_package_id IS NULL)
        OR (d.package_selection = 'specific_package' AND p.id = d.pinned_package_id)
    )
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
LEFT JOIN munki_artifacts icon ON icon.id = p.icon_artifact_id
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
ORDER BY d.position, d.id, lower(p.name), p.id;

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
