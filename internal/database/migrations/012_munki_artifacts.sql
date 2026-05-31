-- +goose Up

CREATE TYPE munki_artifact_kind AS ENUM (
    'package',
    'icon'
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

ALTER TABLE munki_releases
ADD COLUMN installer_artifact_id BIGINT REFERENCES munki_artifacts (id) ON DELETE SET NULL;

CREATE INDEX munki_artifacts_kind_idx
    ON munki_artifacts (kind, lower(location), id);
CREATE INDEX munki_releases_installer_artifact_idx
    ON munki_releases (installer_artifact_id);

-- +goose Down

DROP INDEX IF EXISTS munki_releases_installer_artifact_idx;
DROP INDEX IF EXISTS munki_artifacts_kind_idx;
ALTER TABLE munki_releases
DROP COLUMN IF EXISTS installer_artifact_id;
DROP TABLE IF EXISTS munki_artifacts;
DROP TYPE IF EXISTS munki_artifact_kind;
