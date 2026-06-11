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
    installable_condition,
    blocking_applications_manual_quit_only,
    blocking_applications_quit_script,
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
    installer_artifact_id,
    uninstaller_artifact_id,
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
    @installable_condition,
    @blocking_applications_manual_quit_only,
    @blocking_applications_quit_script,
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
    @installer_choices_xml::jsonb,
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
    sqlc.narg(installer_artifact_id)::bigint,
    sqlc.narg(uninstaller_artifact_id)::bigint,
    @eligible
)
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
    installable_condition = @installable_condition,
    blocking_applications_manual_quit_only = @blocking_applications_manual_quit_only,
    blocking_applications_quit_script = @blocking_applications_quit_script,
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
    installer_choices_xml = @installer_choices_xml::jsonb,
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
    installer_artifact_id = sqlc.narg(installer_artifact_id)::bigint,
    uninstaller_artifact_id = sqlc.narg(uninstaller_artifact_id)::bigint,
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
    software_icon.location AS software_icon_artifact_location
FROM munki_packages p
JOIN munki_software s ON s.id = p.software_id
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
LEFT JOIN munki_artifacts uninstaller ON uninstaller.id = p.uninstaller_artifact_id
LEFT JOIN munki_artifacts software_icon ON software_icon.id = s.icon_artifact_id
WHERE p.id = @id;

-- name: DeleteMunkiPackage :execrows
DELETE FROM munki_packages
WHERE id = @id;

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
