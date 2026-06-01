-- +goose Up

CREATE TYPE munki_deployment_intent AS ENUM (
    'ensure_installed',
    'ensure_absent',
    'update_if_present',
    'optional',
    'featured'
);

CREATE TABLE munki_software_titles (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    developer TEXT NOT NULL DEFAULT '',
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
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
    eligible BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (software_id, name, version)
);

CREATE TABLE munki_deployments (
    id BIGSERIAL PRIMARY KEY,
    package_id BIGINT NOT NULL REFERENCES munki_packages (id) ON DELETE CASCADE,
    intent munki_deployment_intent NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    all_hosts BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
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

CREATE INDEX munki_packages_software_idx
    ON munki_packages (software_id);
CREATE INDEX munki_deployments_package_idx
    ON munki_deployments (package_id);
CREATE INDEX munki_deployments_position_idx
    ON munki_deployments (position, id);

-- +goose Down

DROP TABLE IF EXISTS munki_deployment_exclude_hosts;
DROP TABLE IF EXISTS munki_deployment_include_hosts;
DROP TABLE IF EXISTS munki_deployment_exclude_labels;
DROP TABLE IF EXISTS munki_deployment_include_labels;
DROP TABLE IF EXISTS munki_deployments;
DROP TABLE IF EXISTS munki_packages;
DROP TABLE IF EXISTS munki_software_titles;
DROP TYPE IF EXISTS munki_deployment_intent;
