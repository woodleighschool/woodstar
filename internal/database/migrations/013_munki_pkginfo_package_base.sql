-- +goose Up

ALTER TABLE munki_packages
ADD COLUMN installer_type TEXT NOT NULL DEFAULT 'pkg',
ADD COLUMN uninstall_method TEXT NOT NULL DEFAULT '',
ADD COLUMN restart_action TEXT NOT NULL DEFAULT '',
ADD COLUMN minimum_munki_version TEXT NOT NULL DEFAULT '',
ADD COLUMN minimum_os_version TEXT NOT NULL DEFAULT '',
ADD COLUMN maximum_os_version TEXT NOT NULL DEFAULT '',
ADD COLUMN supported_architectures TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
ADD COLUMN blocking_applications TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
ADD COLUMN requires TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
ADD COLUMN update_for TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
ADD COLUMN unattended_install BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN unattended_uninstall BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN uninstallable BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN on_demand BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN precache BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN icon_name TEXT NOT NULL DEFAULT '',
ADD COLUMN icon_hash TEXT NOT NULL DEFAULT '',
ADD COLUMN icon_artifact_id BIGINT REFERENCES munki_artifacts (id) ON DELETE SET NULL,
ADD COLUMN extra_pkginfo JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(extra_pkginfo) = 'object');

UPDATE munki_packages
SET
    installer_type = COALESCE(NULLIF(metadata->>'installer_type', ''), 'pkg'),
    uninstall_method = COALESCE(metadata->>'uninstall_method', ''),
    restart_action = COALESCE(metadata->>'restart_action', ''),
    minimum_munki_version = COALESCE(metadata->>'minimum_munki_version', ''),
    minimum_os_version = COALESCE(metadata->>'minimum_os_version', ''),
    maximum_os_version = COALESCE(metadata->>'maximum_os_version', ''),
    supported_architectures = COALESCE(
        ARRAY(SELECT jsonb_array_elements_text(metadata->'supported_architectures')),
        ARRAY[]::TEXT[]
    ),
    blocking_applications = COALESCE(
        ARRAY(SELECT jsonb_array_elements_text(metadata->'blocking_applications')),
        ARRAY[]::TEXT[]
    ),
    requires = COALESCE(
        ARRAY(SELECT jsonb_array_elements_text(metadata->'requires')),
        ARRAY[]::TEXT[]
    ),
    update_for = COALESCE(
        ARRAY(SELECT jsonb_array_elements_text(metadata->'update_for')),
        ARRAY[]::TEXT[]
    ),
    unattended_install = COALESCE((metadata->>'unattended_install')::BOOLEAN, FALSE),
    unattended_uninstall = COALESCE((metadata->>'unattended_uninstall')::BOOLEAN, FALSE),
    uninstallable = COALESCE((metadata->>'uninstallable')::BOOLEAN, FALSE),
    on_demand = COALESCE((metadata->>'on_demand')::BOOLEAN, FALSE),
    precache = COALESCE((metadata->>'precache')::BOOLEAN, FALSE),
    extra_pkginfo = metadata
        - 'installer_type'
        - 'uninstall_method'
        - 'restart_action'
        - 'minimum_munki_version'
        - 'minimum_os_version'
        - 'maximum_os_version'
        - 'supported_architectures'
        - 'blocking_applications'
        - 'requires'
        - 'update_for'
        - 'unattended_install'
        - 'unattended_uninstall'
        - 'uninstallable'
        - 'on_demand'
        - 'precache';

ALTER TABLE munki_packages
DROP COLUMN metadata;

CREATE INDEX munki_packages_icon_artifact_idx
    ON munki_packages (icon_artifact_id);

-- +goose Down

DROP INDEX IF EXISTS munki_packages_icon_artifact_idx;

ALTER TABLE munki_packages
ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object');

UPDATE munki_packages
SET metadata = extra_pkginfo
    || jsonb_strip_nulls(jsonb_build_object(
        'installer_type', NULLIF(installer_type, 'pkg'),
        'uninstall_method', NULLIF(uninstall_method, ''),
        'restart_action', NULLIF(restart_action, ''),
        'minimum_munki_version', NULLIF(minimum_munki_version, ''),
        'minimum_os_version', NULLIF(minimum_os_version, ''),
        'maximum_os_version', NULLIF(maximum_os_version, ''),
        'supported_architectures', NULLIF(to_jsonb(supported_architectures), '[]'::jsonb),
        'blocking_applications', NULLIF(to_jsonb(blocking_applications), '[]'::jsonb),
        'requires', NULLIF(to_jsonb(requires), '[]'::jsonb),
        'update_for', NULLIF(to_jsonb(update_for), '[]'::jsonb),
        'unattended_install', CASE WHEN unattended_install THEN TRUE ELSE NULL END,
        'unattended_uninstall', CASE WHEN unattended_uninstall THEN TRUE ELSE NULL END,
        'uninstallable', CASE WHEN uninstallable THEN TRUE ELSE NULL END,
        'on_demand', CASE WHEN on_demand THEN TRUE ELSE NULL END,
        'precache', CASE WHEN precache THEN TRUE ELSE NULL END
    ));

ALTER TABLE munki_packages
DROP COLUMN extra_pkginfo,
DROP COLUMN icon_artifact_id,
DROP COLUMN icon_hash,
DROP COLUMN icon_name,
DROP COLUMN precache,
DROP COLUMN on_demand,
DROP COLUMN uninstallable,
DROP COLUMN unattended_uninstall,
DROP COLUMN unattended_install,
DROP COLUMN update_for,
DROP COLUMN requires,
DROP COLUMN blocking_applications,
DROP COLUMN supported_architectures,
DROP COLUMN maximum_os_version,
DROP COLUMN minimum_os_version,
DROP COLUMN minimum_munki_version,
DROP COLUMN restart_action,
DROP COLUMN uninstall_method,
DROP COLUMN installer_type;
