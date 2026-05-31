-- +goose Up

CREATE TYPE munki_assignment_intent AS ENUM (
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

CREATE TABLE munki_releases (
    id BIGSERIAL PRIMARY KEY,
    software_id BIGINT NOT NULL REFERENCES munki_software_titles (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    pkginfo JSONB NOT NULL CHECK (jsonb_typeof(pkginfo) = 'object'),
    eligible BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (software_id, name, version)
);

CREATE TABLE munki_assignments (
    id BIGSERIAL PRIMARY KEY,
    release_id BIGINT NOT NULL REFERENCES munki_releases (id) ON DELETE CASCADE,
    intent munki_assignment_intent NOT NULL,
    all_hosts BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE munki_assignment_include_labels (
    assignment_id BIGINT NOT NULL REFERENCES munki_assignments (id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (assignment_id, label_id)
);

CREATE TABLE munki_assignment_exclude_labels (
    assignment_id BIGINT NOT NULL REFERENCES munki_assignments (id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    PRIMARY KEY (assignment_id, label_id)
);

CREATE TABLE munki_assignment_include_hosts (
    assignment_id BIGINT NOT NULL REFERENCES munki_assignments (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    PRIMARY KEY (assignment_id, host_id)
);

CREATE TABLE munki_assignment_exclude_hosts (
    assignment_id BIGINT NOT NULL REFERENCES munki_assignments (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    PRIMARY KEY (assignment_id, host_id)
);

CREATE INDEX munki_releases_software_idx
    ON munki_releases (software_id);
CREATE INDEX munki_assignments_release_idx
    ON munki_assignments (release_id);

-- +goose Down

DROP TABLE IF EXISTS munki_assignment_exclude_hosts;
DROP TABLE IF EXISTS munki_assignment_include_hosts;
DROP TABLE IF EXISTS munki_assignment_exclude_labels;
DROP TABLE IF EXISTS munki_assignment_include_labels;
DROP TABLE IF EXISTS munki_assignments;
DROP TABLE IF EXISTS munki_releases;
DROP TABLE IF EXISTS munki_software_titles;
DROP TYPE IF EXISTS munki_assignment_intent;
