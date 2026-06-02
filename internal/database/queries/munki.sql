-- name: CreateMunkiSoftwareTitle :one
INSERT INTO munki_software_titles (
    name,
    display_name,
    description,
    category,
    developer,
    icon_name,
    icon_hash,
    icon_artifact_id
)
VALUES (
    @name,
    @display_name,
    @description,
    @category,
    @developer,
    @icon_name,
    @icon_hash,
    sqlc.narg(icon_artifact_id)::bigint
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
    icon_name = @icon_name,
    icon_hash = @icon_hash,
    icon_artifact_id = sqlc.narg(icon_artifact_id)::bigint,
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
    s.icon_name AS software_icon_name,
    s.icon_hash AS software_icon_hash,
    s.icon_artifact_id AS software_icon_artifact_id,
    art.location AS installer_artifact_location,
    icon.location AS icon_artifact_location,
    software_icon.location AS software_icon_artifact_location
FROM munki_packages p
JOIN munki_software_titles s ON s.id = p.software_id
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
LEFT JOIN munki_artifacts icon ON icon.id = p.icon_artifact_id
LEFT JOIN munki_artifacts software_icon ON software_icon.id = s.icon_artifact_id
WHERE p.id = @id;

-- name: CreateMunkiAssignment :one
INSERT INTO munki_assignments (
    software_id,
    priority,
    label_id,
    effect,
    action,
    optional_install,
    featured_item,
    package_selection,
    pinned_package_id
)
VALUES (
    @software_id,
    CASE
        WHEN @priority::integer > 0 THEN @priority::integer
        ELSE (
            SELECT COALESCE(MAX(a.priority) + 1, 1)
            FROM munki_assignments a
            WHERE a.software_id = @software_id
        )
    END,
    @label_id,
    @effect::munki_assignment_effect,
    sqlc.narg(action)::munki_assignment_action,
    @optional_install,
    @featured_item,
    sqlc.narg(package_selection)::munki_package_selection,
    sqlc.narg(pinned_package_id)::bigint
)
RETURNING *;

-- name: UpdateMunkiAssignment :one
UPDATE munki_assignments
SET
    priority = @priority::integer,
    label_id = @label_id,
    effect = @effect::munki_assignment_effect,
    action = sqlc.narg(action)::munki_assignment_action,
    optional_install = @optional_install,
    featured_item = @featured_item,
    package_selection = sqlc.narg(package_selection)::munki_package_selection,
    pinned_package_id = sqlc.narg(pinned_package_id)::bigint,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: ListMunkiAssignmentIDsBySoftware :many
SELECT a.id
FROM munki_assignments a
WHERE a.software_id = @software_id
ORDER BY a.priority, a.id;

-- name: SetMunkiAssignmentPriorities :exec
UPDATE munki_assignments a
SET priority = -ordered.priority
FROM unnest(@ordered_ids::bigint[]) WITH ORDINALITY AS ordered(id, priority)
WHERE a.id = ordered.id
  AND a.software_id = @software_id;

-- name: NormalizeMunkiAssignmentPriorities :exec
UPDATE munki_assignments a
SET priority = -priority
WHERE a.software_id = @software_id;

-- name: ListEffectiveMunkiPackagesForHost :many
SELECT
    a.id AS assignment_id,
    a.software_id AS assignment_software_id,
    a.effect AS assignment_effect,
    a.action,
    a.optional_install,
    a.featured_item,
    a.package_selection,
    a.pinned_package_id,
    a.priority,
    COALESCE(p.id, 0)::bigint AS package_id,
    COALESCE(p.software_id, a.software_id)::bigint AS software_id,
    s.name AS software_name,
    COALESCE(NULLIF(s.display_name, ''), s.name) AS software_display_name,
    s.icon_name AS software_icon_name,
    s.icon_hash AS software_icon_hash,
    s.icon_artifact_id AS software_icon_artifact_id,
    COALESCE(p.name, '') AS name,
    COALESCE(p.version, '') AS version,
    COALESCE(p.display_name, '') AS display_name,
    COALESCE(p.description, '') AS description,
    COALESCE(p.category, '') AS category,
    COALESCE(p.developer, '') AS developer,
    COALESCE(p.installer_type, 'pkg') AS installer_type,
    COALESCE(p.uninstall_method, '') AS uninstall_method,
    COALESCE(p.restart_action, '') AS restart_action,
    COALESCE(p.minimum_munki_version, '') AS minimum_munki_version,
    COALESCE(p.minimum_os_version, '') AS minimum_os_version,
    COALESCE(p.maximum_os_version, '') AS maximum_os_version,
    COALESCE(p.supported_architectures, ARRAY[]::text[]) AS supported_architectures,
    COALESCE(p.blocking_applications, ARRAY[]::text[]) AS blocking_applications,
    COALESCE(p.requires, ARRAY[]::text[]) AS requires,
    COALESCE(p.update_for, ARRAY[]::text[]) AS update_for,
    COALESCE(p.unattended_install, false) AS unattended_install,
    COALESCE(p.unattended_uninstall, false) AS unattended_uninstall,
    COALESCE(p.uninstallable, false) AS uninstallable,
    COALESCE(p.on_demand, false) AS on_demand,
    COALESCE(p.precache, false) AS precache,
    COALESCE(p.icon_name, '') AS icon_name,
    COALESCE(p.icon_hash, '') AS icon_hash,
    COALESCE(p.extra_pkginfo, '{}'::jsonb) AS extra_pkginfo,
    p.installer_artifact_id,
    art.location AS installer_artifact_location,
    p.icon_artifact_id,
    icon.location AS icon_artifact_location,
    software_icon.location AS software_icon_artifact_location
FROM munki_assignments a
JOIN label_membership lm ON lm.label_id = a.label_id AND lm.host_id = @host_id
JOIN munki_software_titles s ON s.id = a.software_id
LEFT JOIN munki_packages p ON p.software_id = a.software_id
    AND a.effect = 'include'
    AND (
        (a.package_selection = 'latest_eligible' AND a.pinned_package_id IS NULL)
        OR (a.package_selection = 'specific_package' AND p.id = a.pinned_package_id)
    )
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
LEFT JOIN munki_artifacts icon ON icon.id = p.icon_artifact_id
LEFT JOIN munki_artifacts software_icon ON software_icon.id = s.icon_artifact_id
WHERE a.effect = 'exclude'
   OR (a.effect = 'include' AND p.eligible)
ORDER BY a.software_id, a.priority, a.id, lower(p.name), p.id;

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
