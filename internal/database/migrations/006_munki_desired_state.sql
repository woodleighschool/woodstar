-- +goose Up

CREATE TYPE munki_artifact_kind AS ENUM (
    'package',
    'icon'
);

CREATE TYPE munki_assignment_action AS ENUM (
    'install',
    'remove',
    'update_if_present',
    'none'
);

CREATE TYPE munki_package_selection AS ENUM (
    'latest_eligible',
    'specific_package'
);

CREATE TYPE munki_assignment_effect AS ENUM (
    'include',
    'exclude'
);

CREATE TYPE munki_package_relation_kind AS ENUM (
    'requires',
    'update_for'
);

CREATE TABLE munki_artifacts (
    id BIGSERIAL PRIMARY KEY,
    kind munki_artifact_kind NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    sha256 TEXT NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    storage_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (kind, location)
);

CREATE TABLE munki_software_titles (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    developer TEXT NOT NULL DEFAULT '',
    icon_name TEXT NOT NULL DEFAULT '',
    icon_hash TEXT NOT NULL DEFAULT '',
    icon_artifact_id BIGINT REFERENCES munki_artifacts (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE munki_packages (
    id BIGSERIAL PRIMARY KEY,
    software_id BIGINT NOT NULL REFERENCES munki_software_titles (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    developer TEXT NOT NULL DEFAULT '',
    installer_type TEXT NOT NULL DEFAULT 'pkg',
    uninstall_method TEXT NOT NULL DEFAULT 'none',
    custom_uninstall_method TEXT NOT NULL DEFAULT '',
    restart_action TEXT NOT NULL DEFAULT '',
    minimum_munki_version TEXT NOT NULL DEFAULT '',
    minimum_os_version TEXT NOT NULL DEFAULT '',
    maximum_os_version TEXT NOT NULL DEFAULT '',
    supported_architectures TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    blocking_applications TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    unattended_install BOOLEAN NOT NULL DEFAULT FALSE,
    unattended_uninstall BOOLEAN NOT NULL DEFAULT FALSE,
    uninstallable BOOLEAN NOT NULL DEFAULT FALSE,
    on_demand BOOLEAN NOT NULL DEFAULT FALSE,
    precache BOOLEAN NOT NULL DEFAULT FALSE,
    autoremove BOOLEAN NOT NULL DEFAULT FALSE,
    apple_item BOOLEAN NOT NULL DEFAULT FALSE,
    suppress_bundle_relocation BOOLEAN NOT NULL DEFAULT FALSE,
    force_install_after_date TIMESTAMPTZ,
    installed_size BIGINT NOT NULL DEFAULT 0 CHECK (installed_size >= 0),
    payload_identifier TEXT NOT NULL DEFAULT '',
    package_path TEXT NOT NULL DEFAULT '',
    installer_choices_xml TEXT NOT NULL DEFAULT '',
    installer_environment JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(installer_environment) = 'array'),
    installs JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(installs) = 'array'),
    receipts JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(receipts) = 'array'),
    items_to_copy JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(items_to_copy) = 'array'),
    notes TEXT NOT NULL DEFAULT '',
    installcheck_script TEXT NOT NULL DEFAULT '',
    uninstallcheck_script TEXT NOT NULL DEFAULT '',
    preinstall_script TEXT NOT NULL DEFAULT '',
    postinstall_script TEXT NOT NULL DEFAULT '',
    preuninstall_script TEXT NOT NULL DEFAULT '',
    postuninstall_script TEXT NOT NULL DEFAULT '',
    uninstall_script TEXT NOT NULL DEFAULT '',
    version_script TEXT NOT NULL DEFAULT '',
    preinstall_alert_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    preinstall_alert_title TEXT NOT NULL DEFAULT '',
    preinstall_alert_detail TEXT NOT NULL DEFAULT '',
    preinstall_alert_ok_label TEXT NOT NULL DEFAULT '',
    preinstall_alert_cancel_label TEXT NOT NULL DEFAULT '',
    preuninstall_alert_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    preuninstall_alert_title TEXT NOT NULL DEFAULT '',
    preuninstall_alert_detail TEXT NOT NULL DEFAULT '',
    preuninstall_alert_ok_label TEXT NOT NULL DEFAULT '',
    preuninstall_alert_cancel_label TEXT NOT NULL DEFAULT '',
    icon_name TEXT NOT NULL DEFAULT '',
    icon_hash TEXT NOT NULL DEFAULT '',
    installer_artifact_id BIGINT REFERENCES munki_artifacts (id) ON DELETE SET NULL,
    uninstaller_artifact_id BIGINT REFERENCES munki_artifacts (id) ON DELETE SET NULL,
    icon_artifact_id BIGINT REFERENCES munki_artifacts (id) ON DELETE SET NULL,
    eligible BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (software_id, name, version),
    UNIQUE (software_id, id)
);

CREATE TABLE munki_package_relations (
    id BIGSERIAL PRIMARY KEY,
    package_id BIGINT NOT NULL REFERENCES munki_packages (id) ON DELETE CASCADE,
    relation_kind munki_package_relation_kind NOT NULL,
    target_package_id BIGINT REFERENCES munki_packages (id) ON DELETE RESTRICT,
    name TEXT NOT NULL DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT munki_package_relations_target_check CHECK (
        (target_package_id IS NOT NULL AND btrim(name) = '')
        OR (target_package_id IS NULL AND btrim(name) <> '')
    )
);

CREATE TABLE munki_assignments (
    id BIGSERIAL PRIMARY KEY,
    software_id BIGINT NOT NULL REFERENCES munki_software_titles (id) ON DELETE CASCADE,
    priority INTEGER NOT NULL DEFAULT 1,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE RESTRICT,
    effect munki_assignment_effect NOT NULL DEFAULT 'include',
    action munki_assignment_action,
    optional_install BOOLEAN NOT NULL DEFAULT FALSE,
    featured_item BOOLEAN NOT NULL DEFAULT FALSE,
    package_selection munki_package_selection,
    pinned_package_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT munki_assignments_priority_check CHECK (priority >= 1),
    CONSTRAINT munki_assignments_include_payload_check CHECK (
        (
            effect = 'include'
            AND action IS NOT NULL
            AND package_selection IS NOT NULL
            AND (
                (package_selection = 'latest_eligible' AND pinned_package_id IS NULL)
                OR (package_selection = 'specific_package' AND pinned_package_id IS NOT NULL)
            )
        )
        OR (
            effect = 'exclude'
            AND action IS NULL
            AND optional_install IS FALSE
            AND featured_item IS FALSE
            AND package_selection IS NULL
            AND pinned_package_id IS NULL
        )
    ),
    CONSTRAINT munki_assignments_pinned_package_software_fkey FOREIGN KEY (software_id, pinned_package_id)
        REFERENCES munki_packages (software_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX munki_artifacts_kind_idx
    ON munki_artifacts (kind, lower(location), id);
CREATE INDEX munki_software_titles_icon_artifact_idx
    ON munki_software_titles (icon_artifact_id);
CREATE INDEX munki_packages_software_idx
    ON munki_packages (software_id);
CREATE INDEX munki_packages_installer_artifact_idx
    ON munki_packages (installer_artifact_id);
CREATE INDEX munki_packages_uninstaller_artifact_idx
    ON munki_packages (uninstaller_artifact_id);
CREATE INDEX munki_packages_icon_artifact_idx
    ON munki_packages (icon_artifact_id);
CREATE INDEX munki_package_relations_package_idx
    ON munki_package_relations (package_id, relation_kind, position, id);
CREATE INDEX munki_package_relations_target_package_idx
    ON munki_package_relations (target_package_id);
CREATE INDEX munki_assignments_software_idx
    ON munki_assignments (software_id);
CREATE INDEX munki_assignments_pinned_package_idx
    ON munki_assignments (pinned_package_id);
CREATE INDEX munki_assignments_priority_idx
    ON munki_assignments (software_id, priority, id);
CREATE INDEX munki_assignments_label_idx
    ON munki_assignments (label_id);
