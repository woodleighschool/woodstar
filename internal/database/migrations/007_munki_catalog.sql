-- +goose Up

CREATE TYPE munki_manifest_action AS ENUM (
    'managed_installs',
    'managed_uninstalls',
    'managed_updates',
    'optional_installs',
    'featured_items',
    'default_installs'
);

CREATE TYPE munki_package_selection AS ENUM (
    'latest',
    'specific'
);

CREATE TYPE munki_package_relation_kind AS ENUM (
    'requires',
    'update_for'
);

CREATE TABLE munki_software (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    developer TEXT NOT NULL DEFAULT '',
    icon_object_id BIGINT REFERENCES storage_objects (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE munki_packages (
    id BIGSERIAL PRIMARY KEY,
    software_id BIGINT NOT NULL REFERENCES munki_software (id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    installer_type TEXT NOT NULL DEFAULT 'pkg',
    uninstall_method TEXT NOT NULL DEFAULT '' CHECK (
        uninstall_method IN ('', 'removepackages', 'remove_copied_items', 'uninstall_script')
    ),
    restart_action TEXT NOT NULL DEFAULT '' CHECK (
        restart_action IN ('', 'RequireLogout', 'RecommendRestart', 'RequireRestart', 'RequireShutdown')
    ),
    minimum_munki_version TEXT NOT NULL DEFAULT '',
    minimum_os_version TEXT NOT NULL DEFAULT '',
    maximum_os_version TEXT NOT NULL DEFAULT '',
    supported_architectures TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    blocking_applications TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    blocking_applications_none BOOLEAN NOT NULL DEFAULT FALSE,
    installable_condition TEXT NOT NULL DEFAULT '',
    blocking_applications_manual_quit_only BOOLEAN NOT NULL DEFAULT FALSE,
    blocking_applications_quit_script TEXT NOT NULL DEFAULT '',
    unattended_install BOOLEAN NOT NULL DEFAULT FALSE,
    unattended_uninstall BOOLEAN NOT NULL DEFAULT FALSE,
    on_demand BOOLEAN NOT NULL DEFAULT FALSE,
    precache BOOLEAN NOT NULL DEFAULT FALSE,
    autoremove BOOLEAN NOT NULL DEFAULT FALSE,
    apple_item BOOLEAN NOT NULL DEFAULT FALSE,
    suppress_bundle_relocation BOOLEAN NOT NULL DEFAULT FALSE,
    force_install_after_date TIMESTAMPTZ,
    installed_size BIGINT NOT NULL DEFAULT 0 CHECK (installed_size >= 0),
    package_path TEXT NOT NULL DEFAULT '',
    installer_choices_xml JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(installer_choices_xml) = 'array'),
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
    installer_object_id BIGINT REFERENCES storage_objects (id) ON DELETE RESTRICT,
    eligible BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NOT blocking_applications_none OR cardinality(blocking_applications) = 0),
    UNIQUE (software_id, version),
    UNIQUE (software_id, id)
);

CREATE TABLE munki_package_relations (
    id BIGSERIAL PRIMARY KEY,
    package_id BIGINT NOT NULL REFERENCES munki_packages (id) ON DELETE CASCADE,
    relation_kind munki_package_relation_kind NOT NULL,
    target_software_id BIGINT NOT NULL REFERENCES munki_software (id) ON DELETE RESTRICT,
    target_package_id BIGINT REFERENCES munki_packages (id) ON DELETE RESTRICT,
    position INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (target_software_id, target_package_id)
        REFERENCES munki_packages (software_id, id)
        ON DELETE RESTRICT
);

CREATE TABLE munki_software_targets (
    software_id BIGINT NOT NULL REFERENCES munki_software (id) ON DELETE CASCADE,
    direction target_direction NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE RESTRICT,
    actions munki_manifest_action[],
    package_selection munki_package_selection,
    pinned_package_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (software_id, direction, position),
    UNIQUE (software_id, label_id),
    CONSTRAINT munki_software_targets_direction_metadata_check CHECK (
        (
            direction = 'include'
            AND COALESCE(array_length(actions, 1), 0) > 0
            AND package_selection IS NOT NULL
        )
        OR (
            direction = 'exclude'
            AND actions IS NULL
            AND package_selection IS NULL
            AND pinned_package_id IS NULL
        )
    ),
    CONSTRAINT munki_software_targets_package_selection_check CHECK (
        direction <> 'include'
        OR
        (package_selection = 'latest' AND pinned_package_id IS NULL)
        OR (package_selection = 'specific' AND pinned_package_id IS NOT NULL)
    ),
    CONSTRAINT munki_software_targets_pinned_package_software_fkey FOREIGN KEY (software_id, pinned_package_id)
        REFERENCES munki_packages (software_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX munki_software_icon_object_idx
    ON munki_software (icon_object_id);
CREATE INDEX munki_packages_software_idx
    ON munki_packages (software_id);
CREATE INDEX munki_packages_installer_object_idx
    ON munki_packages (installer_object_id);
CREATE INDEX munki_package_relations_package_idx
    ON munki_package_relations (package_id, relation_kind, position, id);
CREATE INDEX munki_package_relations_target_software_idx
    ON munki_package_relations (target_software_id);
CREATE INDEX munki_package_relations_target_package_idx
    ON munki_package_relations (target_package_id);
CREATE INDEX munki_software_targets_label_idx
    ON munki_software_targets (label_id);
CREATE INDEX munki_software_targets_pinned_package_idx
    ON munki_software_targets (pinned_package_id);
