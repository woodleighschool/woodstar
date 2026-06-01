-- +goose Up

CREATE TYPE munki_artifact_kind AS ENUM (
    'package',
    'icon'
);

CREATE TYPE munki_deployment_action AS ENUM (
    'install',
    'remove',
    'update_if_present',
    'none'
);

CREATE TYPE munki_package_selection AS ENUM (
    'latest_eligible',
    'specific_package'
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
    uninstall_method TEXT NOT NULL DEFAULT '',
    restart_action TEXT NOT NULL DEFAULT '',
    minimum_munki_version TEXT NOT NULL DEFAULT '',
    minimum_os_version TEXT NOT NULL DEFAULT '',
    maximum_os_version TEXT NOT NULL DEFAULT '',
    supported_architectures TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    blocking_applications TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    requires TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    update_for TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    unattended_install BOOLEAN NOT NULL DEFAULT FALSE,
    unattended_uninstall BOOLEAN NOT NULL DEFAULT FALSE,
    uninstallable BOOLEAN NOT NULL DEFAULT FALSE,
    on_demand BOOLEAN NOT NULL DEFAULT FALSE,
    precache BOOLEAN NOT NULL DEFAULT FALSE,
    icon_name TEXT NOT NULL DEFAULT '',
    icon_hash TEXT NOT NULL DEFAULT '',
    extra_pkginfo JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(extra_pkginfo) = 'object'),
    installer_artifact_id BIGINT REFERENCES munki_artifacts (id) ON DELETE SET NULL,
    icon_artifact_id BIGINT REFERENCES munki_artifacts (id) ON DELETE SET NULL,
    eligible BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (software_id, name, version),
    UNIQUE (software_id, id)
);

CREATE TABLE munki_deployments (
    id BIGSERIAL PRIMARY KEY,
    software_id BIGINT NOT NULL REFERENCES munki_software_titles (id) ON DELETE CASCADE,
    action munki_deployment_action NOT NULL DEFAULT 'install',
    optional_install BOOLEAN NOT NULL DEFAULT FALSE,
    featured_item BOOLEAN NOT NULL DEFAULT FALSE,
    package_selection munki_package_selection NOT NULL DEFAULT 'latest_eligible',
    pinned_package_id BIGINT,
    position INTEGER NOT NULL DEFAULT 0,
    all_hosts BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT munki_deployments_package_selection_check CHECK (
        (package_selection = 'latest_eligible' AND pinned_package_id IS NULL)
        OR (package_selection = 'specific_package' AND pinned_package_id IS NOT NULL)
    ),
    CONSTRAINT munki_deployments_pinned_package_software_fkey FOREIGN KEY (software_id, pinned_package_id)
        REFERENCES munki_packages (software_id, id)
        ON DELETE RESTRICT
);

CREATE TABLE munki_deployment_include_labels (
    deployment_id BIGINT NOT NULL REFERENCES munki_deployments (id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (deployment_id, label_id)
);

CREATE TABLE munki_deployment_exclude_labels (
    deployment_id BIGINT NOT NULL REFERENCES munki_deployments (id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (deployment_id, label_id)
);

CREATE TABLE munki_deployment_include_hosts (
    deployment_id BIGINT NOT NULL REFERENCES munki_deployments (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    PRIMARY KEY (deployment_id, host_id)
);

CREATE TABLE munki_deployment_exclude_hosts (
    deployment_id BIGINT NOT NULL REFERENCES munki_deployments (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    PRIMARY KEY (deployment_id, host_id)
);

CREATE INDEX munki_artifacts_kind_idx
    ON munki_artifacts (kind, lower(location), id);
CREATE INDEX munki_software_titles_icon_artifact_idx
    ON munki_software_titles (icon_artifact_id);
CREATE INDEX munki_packages_software_idx
    ON munki_packages (software_id);
CREATE INDEX munki_packages_installer_artifact_idx
    ON munki_packages (installer_artifact_id);
CREATE INDEX munki_packages_icon_artifact_idx
    ON munki_packages (icon_artifact_id);
CREATE INDEX munki_deployments_software_idx
    ON munki_deployments (software_id);
CREATE INDEX munki_deployments_pinned_package_idx
    ON munki_deployments (pinned_package_id);
CREATE INDEX munki_deployments_position_idx
    ON munki_deployments (software_id, position, id);

-- +goose Down

DROP TABLE IF EXISTS munki_deployment_exclude_hosts;
DROP TABLE IF EXISTS munki_deployment_include_hosts;
DROP TABLE IF EXISTS munki_deployment_exclude_labels;
DROP TABLE IF EXISTS munki_deployment_include_labels;
DROP TABLE IF EXISTS munki_deployments;
DROP TABLE IF EXISTS munki_packages;
DROP TABLE IF EXISTS munki_software_titles;
DROP TABLE IF EXISTS munki_artifacts;
DROP TYPE IF EXISTS munki_package_selection;
DROP TYPE IF EXISTS munki_deployment_action;
DROP TYPE IF EXISTS munki_artifact_kind;
