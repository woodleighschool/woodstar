-- name: CreateMunkiSoftware :one
INSERT INTO munki_software (
    name,
    description,
    category,
    developer,
    icon_name,
    icon_hash,
    icon_object_id
)
VALUES (
    @name,
    @description,
    @category,
    @developer,
    @icon_name,
    @icon_hash,
    sqlc.narg(icon_object_id)::bigint
)
RETURNING *;

-- name: ListMunkiSoftware :many
SELECT *
FROM munki_software
ORDER BY lower(name), id
LIMIT @limit_rows OFFSET @offset_rows;

-- name: CountMunkiSoftware :one
SELECT COUNT(*)::integer
FROM munki_software;

-- name: GetMunkiSoftwareByID :one
SELECT *
FROM munki_software
WHERE id = @id;

-- name: UpdateMunkiSoftware :one
UPDATE munki_software
SET
    name = @name,
    description = @description,
    category = @category,
    developer = @developer,
    icon_name = @icon_name,
    icon_hash = @icon_hash,
    icon_object_id = sqlc.narg(icon_object_id)::bigint,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteMunkiSoftwareTargetsBySoftware :exec
DELETE FROM munki_software_targets
WHERE software_id = @software_id;

-- name: DeleteMunkiSoftwareTargetsBySoftwareIDs :exec
DELETE FROM munki_software_targets
WHERE software_id = ANY(@ids::bigint[]);

-- name: DeleteMunkiSoftwareByID :one
DELETE FROM munki_software
WHERE id = @id
RETURNING id;

-- name: DeleteMunkiSoftwareByIDs :many
DELETE FROM munki_software
WHERE id = ANY(@ids::bigint[])
RETURNING id;

-- name: CreateMunkiSoftwareInclude :exec
INSERT INTO munki_software_targets (
    software_id,
    direction,
    position,
    label_id,
    actions,
    package_selection,
    pinned_package_id
)
VALUES (
    @software_id,
    'include',
    (@priority)::integer - 1,
    @label_id,
    (@actions)::text[]::munki_manifest_action[],
    @package_selection::munki_package_selection,
    sqlc.narg(pinned_package_id)::bigint
);

-- name: DeleteMunkiSoftwareExcludeLabels :exec
DELETE FROM munki_software_targets
WHERE software_id = @software_id
  AND direction = 'exclude';

-- name: InsertMunkiSoftwareExcludeLabels :exec
INSERT INTO munki_software_targets (software_id, direction, position, label_id)
SELECT @software_id, 'exclude', labels.position - 1, labels.label_id
FROM unnest(@label_ids::bigint[]) WITH ORDINALITY AS labels(label_id, position);

-- name: ListMunkiSoftwareExcludeLabels :many
SELECT software_id, label_id
FROM munki_software_targets
WHERE software_id = ANY(@software_ids::bigint[])
  AND direction = 'exclude'
ORDER BY software_id, position;

-- name: ListEffectiveMunkiPackagesForHost :many
SELECT
    (a.position + 1)::bigint AS target_id,
    a.software_id AS target_software_id,
    a.actions::text[] AS actions,
    a.package_selection::munki_package_selection AS package_selection,
    a.pinned_package_id,
    (a.position + 1)::integer AS priority,
    COALESCE(p.id, 0)::bigint AS package_id,
    COALESCE(p.software_id, a.software_id)::bigint AS software_id,
    s.name AS software_name,
    s.description AS software_description,
    s.category AS software_category,
    s.developer AS software_developer,
    s.icon_name AS software_icon_name,
    s.icon_hash AS software_icon_hash,
    s.icon_object_id AS software_icon_object_id,
    COALESCE(p.version, '') AS version,
    COALESCE(p.installer_type, 'pkg') AS installer_type,
    COALESCE(p.uninstall_method, '') AS uninstall_method,
    COALESCE(p.restart_action, '') AS restart_action,
    COALESCE(p.minimum_munki_version, '') AS minimum_munki_version,
    COALESCE(p.minimum_os_version, '') AS minimum_os_version,
    COALESCE(p.maximum_os_version, '') AS maximum_os_version,
    COALESCE(p.supported_architectures, ARRAY[]::text[]) AS supported_architectures,
    p.blocking_applications AS blocking_applications,
    COALESCE(p.installable_condition, '') AS installable_condition,
    COALESCE(p.blocking_applications_manual_quit_only, false) AS blocking_applications_manual_quit_only,
    COALESCE(p.blocking_applications_quit_script, '') AS blocking_applications_quit_script,
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
    COALESCE(p.installer_choices_xml, '[]'::jsonb) AS installer_choices_xml,
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
    p.installer_object_id,
    installer_obj.prefix AS installer_object_prefix,
    installer_obj.filename AS installer_object_filename,
    p.uninstaller_object_id,
    uninstaller_obj.prefix AS uninstaller_object_prefix,
    uninstaller_obj.filename AS uninstaller_object_filename,
    icon_obj.prefix AS software_icon_object_prefix,
    icon_obj.filename AS software_icon_object_filename
FROM munki_software_targets a
JOIN label_membership lm ON lm.label_id = a.label_id AND lm.host_id = @host_id
JOIN munki_software s ON s.id = a.software_id
JOIN munki_packages p ON p.software_id = a.software_id
    AND (
        (a.package_selection = 'latest_eligible' AND a.pinned_package_id IS NULL)
        OR (a.package_selection = 'specific_package' AND p.id = a.pinned_package_id)
    )
LEFT JOIN storage_objects installer_obj ON installer_obj.id = p.installer_object_id
LEFT JOIN storage_objects uninstaller_obj ON uninstaller_obj.id = p.uninstaller_object_id
LEFT JOIN storage_objects icon_obj ON icon_obj.id = s.icon_object_id
WHERE a.direction = 'include'
  AND p.eligible
  AND installer_obj.available_at IS NOT NULL
  AND NOT EXISTS (
      SELECT 1
      FROM munki_software_targets excluded
      JOIN label_membership excluded_lm
        ON excluded_lm.label_id = excluded.label_id
       AND excluded_lm.host_id = @host_id
      WHERE excluded.software_id = a.software_id
        AND excluded.direction = 'exclude'
  )
ORDER BY a.software_id, a.position, p.id;

-- name: DeleteUnreferencedStorageObjects :many
DELETE FROM storage_objects o
WHERE o.id = ANY(@ids::bigint[])
  AND NOT EXISTS (SELECT 1 FROM munki_software s WHERE s.icon_object_id = o.id)
  AND NOT EXISTS (
      SELECT 1 FROM munki_packages p
      WHERE p.installer_object_id = o.id OR p.uninstaller_object_id = o.id
  )
RETURNING o.prefix, o.id, o.filename;

-- name: ListUnreferencedStorageObjects :many
SELECT o.prefix, o.id, o.filename
FROM storage_objects o
WHERE o.id = ANY(@ids::bigint[])
  AND NOT EXISTS (SELECT 1 FROM munki_software s WHERE s.icon_object_id = o.id)
  AND NOT EXISTS (
      SELECT 1 FROM munki_packages p
      WHERE p.installer_object_id = o.id OR p.uninstaller_object_id = o.id
  );

-- name: SetMunkiSoftwareIconObject :execrows
UPDATE munki_software
SET icon_object_id = @object_id,
    updated_at = now()
WHERE id = @id;
