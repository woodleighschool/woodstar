-- +goose Up

CREATE TYPE munki_manifest_action AS ENUM (
    'managed_installs',
    'managed_uninstalls',
    'managed_updates',
    'optional_installs',
    'featured_items',
    'default_installs'
);
CREATE TYPE munki_package_selection AS ENUM ('latest', 'specific');
CREATE TYPE munki_package_relation_kind AS ENUM ('requires', 'update_for');

-- Munki reported host state --------------------------------------------------

CREATE TABLE munki_host_status (
    host_id BIGINT PRIMARY KEY REFERENCES hosts (id) ON DELETE CASCADE,
    version TEXT NOT NULL DEFAULT '',
    manifest_name TEXT NOT NULL DEFAULT '',
    errors TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    warnings TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    problem_installs TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    run_started_at TIMESTAMPTZ,
    run_ended_at TIMESTAMPTZ
);

CREATE TABLE munki_host_items (
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    installed BOOLEAN NOT NULL DEFAULT FALSE,
    installed_version TEXT NOT NULL DEFAULT '',
    target_version TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (host_id, name)
);

CREATE INDEX munki_host_items_host_idx
    ON munki_host_items (host_id);

-- Catalog --------------------------------------------------------------------

CREATE TABLE munki_software (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    developer TEXT NOT NULL DEFAULT '',
    icon_object_id BIGINT REFERENCES storage_objects (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT munki_software_name_unique UNIQUE (name)
);

CREATE INDEX munki_software_icon_object_idx
    ON munki_software (icon_object_id);

CREATE TABLE munki_packages (
    id BIGSERIAL PRIMARY KEY,
    software_id BIGINT NOT NULL REFERENCES munki_software (id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    installer_type TEXT NOT NULL DEFAULT 'pkg',
    installer_object_id BIGINT REFERENCES storage_objects (id) ON DELETE RESTRICT,
    uninstallable BOOLEAN NOT NULL DEFAULT FALSE,
    uninstall_method TEXT NOT NULL DEFAULT '',
    restart_action TEXT NOT NULL DEFAULT '',
    minimum_munki_version TEXT NOT NULL DEFAULT '',
    minimum_os_version TEXT NOT NULL DEFAULT '',
    maximum_os_version TEXT NOT NULL DEFAULT '',
    supported_architectures TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    blocking_applications TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    blocking_applications_none BOOLEAN NOT NULL DEFAULT FALSE,
    blocking_applications_manual_quit_only BOOLEAN NOT NULL DEFAULT FALSE,
    blocking_applications_quit_script TEXT NOT NULL DEFAULT '',
    installable_condition TEXT NOT NULL DEFAULT '',
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
    installer_choices_xml JSONB NOT NULL DEFAULT '[]'::JSONB
        CHECK (jsonb_typeof(installer_choices_xml) = 'array'),
    installer_environment JSONB NOT NULL DEFAULT '[]'::JSONB
        CHECK (jsonb_typeof(installer_environment) = 'array'),
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (software_id, version),
    UNIQUE (software_id, id),
    CONSTRAINT munki_packages_uninstall_method_check CHECK (
        uninstall_method IN ('', 'removepackages', 'remove_copied_items', 'uninstall_script')
    ),
    CONSTRAINT munki_packages_restart_action_check CHECK (
        restart_action IN ('', 'RequireLogout', 'RecommendRestart', 'RequireRestart', 'RequireShutdown')
    ),
    CONSTRAINT munki_packages_blocking_applications_none_check CHECK (
        NOT blocking_applications_none OR cardinality(blocking_applications) = 0
    ),
    CONSTRAINT munki_packages_installer_object_check CHECK (
        (installer_type = 'nopkg' AND installer_object_id IS NULL)
        OR (installer_type IN ('pkg', 'copy_from_dmg') AND installer_object_id IS NOT NULL)
    )
);

CREATE INDEX munki_packages_software_idx
    ON munki_packages (software_id);
CREATE UNIQUE INDEX munki_packages_installer_object_idx
    ON munki_packages (installer_object_id)
    WHERE installer_object_id IS NOT NULL;

CREATE TABLE munki_package_relations (
    id BIGSERIAL PRIMARY KEY,
    package_id BIGINT NOT NULL REFERENCES munki_packages (id) ON DELETE CASCADE,
    relation_kind munki_package_relation_kind NOT NULL,
    target_software_id BIGINT NOT NULL,
    target_package_id BIGINT REFERENCES munki_packages (id) ON DELETE RESTRICT,
    position INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT munki_package_relations_target_software_id_fkey
        FOREIGN KEY (target_software_id)
        REFERENCES munki_software (id) ON DELETE RESTRICT,
    CONSTRAINT munki_package_relations_target_software_package_fkey
        FOREIGN KEY (target_software_id, target_package_id)
        REFERENCES munki_packages (software_id, id) ON DELETE RESTRICT
);

CREATE INDEX munki_package_relations_package_idx
    ON munki_package_relations (package_id, relation_kind, position, id);
CREATE INDEX munki_package_relations_target_package_idx
    ON munki_package_relations (target_package_id);
CREATE INDEX munki_package_relations_target_software_idx
    ON munki_package_relations (target_software_id);

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
        OR (package_selection = 'latest' AND pinned_package_id IS NULL)
        OR (package_selection = 'specific' AND pinned_package_id IS NOT NULL)
    ),
    CONSTRAINT munki_software_targets_pinned_package_software_fkey
        FOREIGN KEY (software_id, pinned_package_id)
        REFERENCES munki_packages (software_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX munki_software_targets_label_idx
    ON munki_software_targets (label_id);
CREATE INDEX munki_software_targets_pinned_package_idx
    ON munki_software_targets (pinned_package_id);

CREATE FUNCTION munki_resolved_software_for_host(_host_id BIGINT)
RETURNS TABLE (
    software_id BIGINT,
    name TEXT,
    actions TEXT[],
    package_selection TEXT,
    pinned_package_id BIGINT
)
LANGUAGE sql
STABLE
AS $$
WITH host_labels AS (
    SELECT label_id
    FROM label_membership
    WHERE host_id = _host_id
),
matching_includes AS (
    SELECT
        s.id AS software_id,
        s.name,
        include_target.actions,
        include_target.package_selection,
        include_target.pinned_package_id,
        row_number() OVER (
            PARTITION BY s.id
            ORDER BY include_target.position
        ) AS include_rank
    FROM munki_software s
    JOIN munki_software_targets include_target
        ON include_target.software_id = s.id
       AND include_target.direction = 'include'
    JOIN host_labels include_label
        ON include_label.label_id = include_target.label_id
    WHERE NOT EXISTS (
        SELECT 1
        FROM munki_software_targets exclude_target
        JOIN host_labels exclude_label
            ON exclude_label.label_id = exclude_target.label_id
        WHERE exclude_target.software_id = s.id
          AND exclude_target.direction = 'exclude'
    )
      AND EXISTS (
        SELECT 1
        FROM munki_packages package
        WHERE package.software_id = s.id
          AND (
              include_target.package_selection = 'latest'
              OR package.id = include_target.pinned_package_id
          )
    )
)
SELECT
    software_id,
    name,
    actions::TEXT[],
    package_selection::TEXT,
    pinned_package_id
FROM matching_includes
WHERE include_rank = 1
$$;

-- Distribution ---------------------------------------------------------------

CREATE TABLE munki_distribution_points (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    position INT NOT NULL UNIQUE,
    client_cidrs TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    client_base_url TEXT NOT NULL DEFAULT '',
    key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE munki_distribution_package_states (
    distribution_point_id BIGINT NOT NULL
        REFERENCES munki_distribution_points (id) ON DELETE CASCADE,
    package_id BIGINT NOT NULL
        REFERENCES munki_packages (id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'syncing', 'current', 'error')),
    reported_sha256 TEXT,
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (distribution_point_id, package_id)
);

CREATE INDEX munki_distribution_package_states_package_idx
    ON munki_distribution_package_states (package_id);

-- Client resources -----------------------------------------------------------

CREATE TABLE munki_client_resources (
    -- Client resources are limited to ID 1 until multiple targetable resources are supported.
    id BIGINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    archive_object_id BIGINT NOT NULL
        REFERENCES storage_objects (id) ON DELETE RESTRICT,
    custom BOOLEAN NOT NULL DEFAULT FALSE,
    banner_object_id BIGINT
        REFERENCES storage_objects (id) ON DELETE RESTRICT,
    banner_fit TEXT NOT NULL DEFAULT 'height' CHECK (banner_fit IN ('height', 'cover')),
    banner_focal_x SMALLINT NOT NULL DEFAULT 0 CHECK (banner_focal_x BETWEEN 0 AND 100),
    links JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(links) = 'array'),
    footer_text TEXT NOT NULL DEFAULT '',
    footer_links JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(footer_links) = 'array'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (custom OR banner_object_id IS NOT NULL)
);
