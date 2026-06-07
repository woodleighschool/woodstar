-- name: CreateMunkiSoftwareTitle :one
INSERT INTO munki_software_titles (
    name,
    description,
    category,
    developer,
    icon_name,
    icon_hash,
    icon_artifact_id
)
VALUES (
    @name,
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
ORDER BY lower(name), id
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
    description = @description,
    category = @category,
    developer = @developer,
    icon_name = @icon_name,
    icon_hash = @icon_hash,
    icon_artifact_id = sqlc.narg(icon_artifact_id)::bigint,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteMunkiAssignmentsBySoftware :exec
DELETE FROM munki_assignments
WHERE software_id = @software_id;

-- name: DeleteMunkiAssignmentsBySoftwareIDs :exec
DELETE FROM munki_assignments
WHERE software_id = ANY(@ids::bigint[]);

-- name: DeleteMunkiSoftwareTitle :one
DELETE FROM munki_software_titles
WHERE id = @id
RETURNING id;

-- name: DeleteMunkiSoftwareTitles :many
DELETE FROM munki_software_titles
WHERE id = ANY(@ids::bigint[])
RETURNING id;

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
    version,
    installer_type,
    uninstall_method,
    restart_action,
    minimum_munki_version,
    minimum_os_version,
    maximum_os_version,
    supported_architectures,
    blocking_applications,
    unattended_install,
    unattended_uninstall,
    on_demand,
    precache,
    autoremove,
    apple_item,
    suppress_bundle_relocation,
    force_install_after_date,
    installed_size,
    package_path,
    installer_choices_xml,
    installer_environment,
    installs,
    receipts,
    items_to_copy,
    notes,
    installcheck_script,
    uninstallcheck_script,
    preinstall_script,
    postinstall_script,
    preuninstall_script,
    postuninstall_script,
    uninstall_script,
    version_script,
    preinstall_alert_enabled,
    preinstall_alert_title,
    preinstall_alert_detail,
    preinstall_alert_ok_label,
    preinstall_alert_cancel_label,
    preuninstall_alert_enabled,
    preuninstall_alert_title,
    preuninstall_alert_detail,
    preuninstall_alert_ok_label,
    preuninstall_alert_cancel_label,
    icon_name,
    icon_hash,
    installer_artifact_id,
    uninstaller_artifact_id,
    icon_artifact_id,
    eligible
)
VALUES (
    @software_id,
    @version,
    @installer_type,
    @uninstall_method,
    @restart_action,
    @minimum_munki_version,
    @minimum_os_version,
    @maximum_os_version,
    @supported_architectures::text[],
    @blocking_applications::text[],
    @unattended_install,
    @unattended_uninstall,
    @on_demand,
    @precache,
    @autoremove,
    @apple_item,
    @suppress_bundle_relocation,
    sqlc.narg(force_install_after_date)::timestamptz,
    @installed_size,
    @package_path,
    @installer_choices_xml,
    @installer_environment::jsonb,
    @installs::jsonb,
    @receipts::jsonb,
    @items_to_copy::jsonb,
    @notes,
    @installcheck_script,
    @uninstallcheck_script,
    @preinstall_script,
    @postinstall_script,
    @preuninstall_script,
    @postuninstall_script,
    @uninstall_script,
    @version_script,
    @preinstall_alert_enabled,
    @preinstall_alert_title,
    @preinstall_alert_detail,
    @preinstall_alert_ok_label,
    @preinstall_alert_cancel_label,
    @preuninstall_alert_enabled,
    @preuninstall_alert_title,
    @preuninstall_alert_detail,
    @preuninstall_alert_ok_label,
    @preuninstall_alert_cancel_label,
    @icon_name,
    @icon_hash,
    sqlc.narg(installer_artifact_id)::bigint,
    sqlc.narg(uninstaller_artifact_id)::bigint,
    sqlc.narg(icon_artifact_id)::bigint,
    @eligible
)
RETURNING *;

-- name: UpsertMunkiPackage :one
INSERT INTO munki_packages (
    software_id,
    version,
    installer_type,
    uninstall_method,
    restart_action,
    minimum_munki_version,
    minimum_os_version,
    maximum_os_version,
    supported_architectures,
    blocking_applications,
    unattended_install,
    unattended_uninstall,
    on_demand,
    precache,
    autoremove,
    apple_item,
    suppress_bundle_relocation,
    force_install_after_date,
    installed_size,
    package_path,
    installer_choices_xml,
    installer_environment,
    installs,
    receipts,
    items_to_copy,
    notes,
    installcheck_script,
    uninstallcheck_script,
    preinstall_script,
    postinstall_script,
    preuninstall_script,
    postuninstall_script,
    uninstall_script,
    version_script,
    preinstall_alert_enabled,
    preinstall_alert_title,
    preinstall_alert_detail,
    preinstall_alert_ok_label,
    preinstall_alert_cancel_label,
    preuninstall_alert_enabled,
    preuninstall_alert_title,
    preuninstall_alert_detail,
    preuninstall_alert_ok_label,
    preuninstall_alert_cancel_label,
    icon_name,
    icon_hash,
    installer_artifact_id,
    uninstaller_artifact_id,
    icon_artifact_id,
    eligible
)
VALUES (
    @software_id,
    @version,
    @installer_type,
    @uninstall_method,
    @restart_action,
    @minimum_munki_version,
    @minimum_os_version,
    @maximum_os_version,
    @supported_architectures::text[],
    @blocking_applications::text[],
    @unattended_install,
    @unattended_uninstall,
    @on_demand,
    @precache,
    @autoremove,
    @apple_item,
    @suppress_bundle_relocation,
    sqlc.narg(force_install_after_date)::timestamptz,
    @installed_size,
    @package_path,
    @installer_choices_xml,
    @installer_environment::jsonb,
    @installs::jsonb,
    @receipts::jsonb,
    @items_to_copy::jsonb,
    @notes,
    @installcheck_script,
    @uninstallcheck_script,
    @preinstall_script,
    @postinstall_script,
    @preuninstall_script,
    @postuninstall_script,
    @uninstall_script,
    @version_script,
    @preinstall_alert_enabled,
    @preinstall_alert_title,
    @preinstall_alert_detail,
    @preinstall_alert_ok_label,
    @preinstall_alert_cancel_label,
    @preuninstall_alert_enabled,
    @preuninstall_alert_title,
    @preuninstall_alert_detail,
    @preuninstall_alert_ok_label,
    @preuninstall_alert_cancel_label,
    @icon_name,
    @icon_hash,
    sqlc.narg(installer_artifact_id)::bigint,
    sqlc.narg(uninstaller_artifact_id)::bigint,
    sqlc.narg(icon_artifact_id)::bigint,
    @eligible
)
ON CONFLICT (software_id, version) DO UPDATE SET
    installer_type = EXCLUDED.installer_type,
    uninstall_method = EXCLUDED.uninstall_method,
    restart_action = EXCLUDED.restart_action,
    minimum_munki_version = EXCLUDED.minimum_munki_version,
    minimum_os_version = EXCLUDED.minimum_os_version,
    maximum_os_version = EXCLUDED.maximum_os_version,
    supported_architectures = EXCLUDED.supported_architectures,
    blocking_applications = EXCLUDED.blocking_applications,
    unattended_install = EXCLUDED.unattended_install,
    unattended_uninstall = EXCLUDED.unattended_uninstall,
    on_demand = EXCLUDED.on_demand,
    precache = EXCLUDED.precache,
    autoremove = EXCLUDED.autoremove,
    apple_item = EXCLUDED.apple_item,
    suppress_bundle_relocation = EXCLUDED.suppress_bundle_relocation,
    force_install_after_date = EXCLUDED.force_install_after_date,
    installed_size = EXCLUDED.installed_size,
    package_path = EXCLUDED.package_path,
    installer_choices_xml = EXCLUDED.installer_choices_xml,
    installer_environment = EXCLUDED.installer_environment,
    installs = EXCLUDED.installs,
    receipts = EXCLUDED.receipts,
    items_to_copy = EXCLUDED.items_to_copy,
    notes = EXCLUDED.notes,
    installcheck_script = EXCLUDED.installcheck_script,
    uninstallcheck_script = EXCLUDED.uninstallcheck_script,
    preinstall_script = EXCLUDED.preinstall_script,
    postinstall_script = EXCLUDED.postinstall_script,
    preuninstall_script = EXCLUDED.preuninstall_script,
    postuninstall_script = EXCLUDED.postuninstall_script,
    uninstall_script = EXCLUDED.uninstall_script,
    version_script = EXCLUDED.version_script,
    preinstall_alert_enabled = EXCLUDED.preinstall_alert_enabled,
    preinstall_alert_title = EXCLUDED.preinstall_alert_title,
    preinstall_alert_detail = EXCLUDED.preinstall_alert_detail,
    preinstall_alert_ok_label = EXCLUDED.preinstall_alert_ok_label,
    preinstall_alert_cancel_label = EXCLUDED.preinstall_alert_cancel_label,
    preuninstall_alert_enabled = EXCLUDED.preuninstall_alert_enabled,
    preuninstall_alert_title = EXCLUDED.preuninstall_alert_title,
    preuninstall_alert_detail = EXCLUDED.preuninstall_alert_detail,
    preuninstall_alert_ok_label = EXCLUDED.preuninstall_alert_ok_label,
    preuninstall_alert_cancel_label = EXCLUDED.preuninstall_alert_cancel_label,
    icon_name = EXCLUDED.icon_name,
    icon_hash = EXCLUDED.icon_hash,
    installer_artifact_id = EXCLUDED.installer_artifact_id,
    uninstaller_artifact_id = EXCLUDED.uninstaller_artifact_id,
    icon_artifact_id = EXCLUDED.icon_artifact_id,
    eligible = EXCLUDED.eligible,
    updated_at = now()
RETURNING *;

-- name: UpdateMunkiPackage :one
UPDATE munki_packages
SET
    version = @version,
    installer_type = @installer_type,
    uninstall_method = @uninstall_method,
    restart_action = @restart_action,
    minimum_munki_version = @minimum_munki_version,
    minimum_os_version = @minimum_os_version,
    maximum_os_version = @maximum_os_version,
    supported_architectures = @supported_architectures::text[],
    blocking_applications = @blocking_applications::text[],
    unattended_install = @unattended_install,
    unattended_uninstall = @unattended_uninstall,
    on_demand = @on_demand,
    precache = @precache,
    autoremove = @autoremove,
    apple_item = @apple_item,
    suppress_bundle_relocation = @suppress_bundle_relocation,
    force_install_after_date = sqlc.narg(force_install_after_date)::timestamptz,
    installed_size = @installed_size,
    package_path = @package_path,
    installer_choices_xml = @installer_choices_xml,
    installer_environment = @installer_environment::jsonb,
    installs = @installs::jsonb,
    receipts = @receipts::jsonb,
    items_to_copy = @items_to_copy::jsonb,
    notes = @notes,
    installcheck_script = @installcheck_script,
    uninstallcheck_script = @uninstallcheck_script,
    preinstall_script = @preinstall_script,
    postinstall_script = @postinstall_script,
    preuninstall_script = @preuninstall_script,
    postuninstall_script = @postuninstall_script,
    uninstall_script = @uninstall_script,
    version_script = @version_script,
    preinstall_alert_enabled = @preinstall_alert_enabled,
    preinstall_alert_title = @preinstall_alert_title,
    preinstall_alert_detail = @preinstall_alert_detail,
    preinstall_alert_ok_label = @preinstall_alert_ok_label,
    preinstall_alert_cancel_label = @preinstall_alert_cancel_label,
    preuninstall_alert_enabled = @preuninstall_alert_enabled,
    preuninstall_alert_title = @preuninstall_alert_title,
    preuninstall_alert_detail = @preuninstall_alert_detail,
    preuninstall_alert_ok_label = @preuninstall_alert_ok_label,
    preuninstall_alert_cancel_label = @preuninstall_alert_cancel_label,
    icon_name = @icon_name,
    icon_hash = @icon_hash,
    installer_artifact_id = sqlc.narg(installer_artifact_id)::bigint,
    uninstaller_artifact_id = sqlc.narg(uninstaller_artifact_id)::bigint,
    icon_artifact_id = sqlc.narg(icon_artifact_id)::bigint,
    eligible = @eligible,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: GetMunkiPackageByID :one
SELECT
    p.*,
    s.name AS software_name,
    s.description AS software_description,
    s.category AS software_category,
    s.developer AS software_developer,
    s.icon_name AS software_icon_name,
    s.icon_hash AS software_icon_hash,
    s.icon_artifact_id AS software_icon_artifact_id,
    art.location AS installer_artifact_location,
    uninstaller.location AS uninstaller_artifact_location,
    icon.location AS icon_artifact_location,
    software_icon.location AS software_icon_artifact_location
FROM munki_packages p
JOIN munki_software_titles s ON s.id = p.software_id
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
LEFT JOIN munki_artifacts uninstaller ON uninstaller.id = p.uninstaller_artifact_id
LEFT JOIN munki_artifacts icon ON icon.id = p.icon_artifact_id
LEFT JOIN munki_artifacts software_icon ON software_icon.id = s.icon_artifact_id
WHERE p.id = @id;

-- name: DeleteMunkiPackageRelationsByKind :exec
DELETE FROM munki_package_relations
WHERE package_id = @package_id
  AND relation_kind = @relation_kind::munki_package_relation_kind;

-- name: CreateMunkiPackageRelation :exec
INSERT INTO munki_package_relations (
    package_id,
    relation_kind,
    target_package_id,
    position
)
VALUES (
    @package_id,
    @relation_kind::munki_package_relation_kind,
    @target_package_id,
    @position::integer
);

-- name: CreateMunkiAssignment :one
INSERT INTO munki_assignments (
    software_id,
    priority,
    label_id,
    action,
    optional_install,
    featured_item,
    package_selection,
    pinned_package_id
)
VALUES (
    @software_id,
    @priority::integer,
    @label_id,
    @action::munki_assignment_action,
    @optional_install,
    @featured_item,
    @package_selection::munki_package_selection,
    sqlc.narg(pinned_package_id)::bigint
)
RETURNING *;

-- name: DeleteMunkiAssignmentExcludeLabels :exec
DELETE FROM munki_assignment_exclude_labels
WHERE software_id = @software_id;

-- name: InsertMunkiAssignmentExcludeLabels :exec
INSERT INTO munki_assignment_exclude_labels (software_id, label_id)
SELECT @software_id, label_id
FROM unnest(@label_ids::bigint[]) AS label_id;

-- name: ListMunkiAssignmentExcludeLabels :many
SELECT software_id, label_id
FROM munki_assignment_exclude_labels
WHERE software_id = ANY(@software_ids::bigint[])
ORDER BY software_id, label_id;

-- name: ListEffectiveMunkiPackagesForHost :many
SELECT
    a.id AS assignment_id,
    a.software_id AS assignment_software_id,
    a.action,
    a.optional_install,
    a.featured_item,
    a.package_selection,
    a.pinned_package_id,
    a.priority,
    COALESCE(p.id, 0)::bigint AS package_id,
    COALESCE(p.software_id, a.software_id)::bigint AS software_id,
    s.name AS software_name,
    s.description AS software_description,
    s.category AS software_category,
    s.developer AS software_developer,
    s.icon_name AS software_icon_name,
    s.icon_hash AS software_icon_hash,
    s.icon_artifact_id AS software_icon_artifact_id,
    COALESCE(p.version, '') AS version,
    COALESCE(p.installer_type, 'pkg') AS installer_type,
    COALESCE(p.uninstall_method, '') AS uninstall_method,
    COALESCE(p.restart_action, '') AS restart_action,
    COALESCE(p.minimum_munki_version, '') AS minimum_munki_version,
    COALESCE(p.minimum_os_version, '') AS minimum_os_version,
    COALESCE(p.maximum_os_version, '') AS maximum_os_version,
    COALESCE(p.supported_architectures, ARRAY[]::text[]) AS supported_architectures,
    COALESCE(p.blocking_applications, ARRAY[]::text[]) AS blocking_applications,
    COALESCE(p.unattended_install, false) AS unattended_install,
    COALESCE(p.unattended_uninstall, false) AS unattended_uninstall,
    COALESCE(p.on_demand, false) AS on_demand,
    COALESCE(p.precache, false) AS precache,
    COALESCE(p.autoremove, false) AS autoremove,
    COALESCE(p.apple_item, false) AS apple_item,
    COALESCE(p.suppress_bundle_relocation, false) AS suppress_bundle_relocation,
    p.force_install_after_date,
    COALESCE(p.installed_size, 0)::bigint AS installed_size,
    COALESCE(p.package_path, '') AS package_path,
    COALESCE(p.installer_choices_xml, '') AS installer_choices_xml,
    COALESCE(p.installer_environment, '[]'::jsonb) AS installer_environment,
    COALESCE(p.installs, '[]'::jsonb) AS installs,
    COALESCE(p.receipts, '[]'::jsonb) AS receipts,
    COALESCE(p.items_to_copy, '[]'::jsonb) AS items_to_copy,
    COALESCE(p.notes, '') AS notes,
    COALESCE(p.installcheck_script, '') AS installcheck_script,
    COALESCE(p.uninstallcheck_script, '') AS uninstallcheck_script,
    COALESCE(p.preinstall_script, '') AS preinstall_script,
    COALESCE(p.postinstall_script, '') AS postinstall_script,
    COALESCE(p.preuninstall_script, '') AS preuninstall_script,
    COALESCE(p.postuninstall_script, '') AS postuninstall_script,
    COALESCE(p.uninstall_script, '') AS uninstall_script,
    COALESCE(p.version_script, '') AS version_script,
    COALESCE(p.preinstall_alert_enabled, false) AS preinstall_alert_enabled,
    COALESCE(p.preinstall_alert_title, '') AS preinstall_alert_title,
    COALESCE(p.preinstall_alert_detail, '') AS preinstall_alert_detail,
    COALESCE(p.preinstall_alert_ok_label, '') AS preinstall_alert_ok_label,
    COALESCE(p.preinstall_alert_cancel_label, '') AS preinstall_alert_cancel_label,
    COALESCE(p.preuninstall_alert_enabled, false) AS preuninstall_alert_enabled,
    COALESCE(p.preuninstall_alert_title, '') AS preuninstall_alert_title,
    COALESCE(p.preuninstall_alert_detail, '') AS preuninstall_alert_detail,
    COALESCE(p.preuninstall_alert_ok_label, '') AS preuninstall_alert_ok_label,
    COALESCE(p.preuninstall_alert_cancel_label, '') AS preuninstall_alert_cancel_label,
    COALESCE(p.icon_name, '') AS icon_name,
    COALESCE(p.icon_hash, '') AS icon_hash,
    p.installer_artifact_id,
    art.location AS installer_artifact_location,
    p.uninstaller_artifact_id,
    uninstaller.location AS uninstaller_artifact_location,
    p.icon_artifact_id,
    icon.location AS icon_artifact_location,
    software_icon.location AS software_icon_artifact_location
FROM munki_assignments a
JOIN label_membership lm ON lm.label_id = a.label_id AND lm.host_id = @host_id
JOIN munki_software_titles s ON s.id = a.software_id
JOIN munki_packages p ON p.software_id = a.software_id
    AND (
        (a.package_selection = 'latest_eligible' AND a.pinned_package_id IS NULL)
        OR (a.package_selection = 'specific_package' AND p.id = a.pinned_package_id)
    )
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
LEFT JOIN munki_artifacts uninstaller ON uninstaller.id = p.uninstaller_artifact_id
LEFT JOIN munki_artifacts icon ON icon.id = p.icon_artifact_id
LEFT JOIN munki_artifacts software_icon ON software_icon.id = s.icon_artifact_id
WHERE p.eligible
  AND NOT EXISTS (
      SELECT 1
      FROM munki_assignment_exclude_labels excluded
      JOIN label_membership excluded_lm
        ON excluded_lm.label_id = excluded.label_id
       AND excluded_lm.host_id = @host_id
      WHERE excluded.software_id = a.software_id
  )
ORDER BY a.software_id, a.priority, a.id, p.id;

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
