-- +goose Up

CREATE TABLE munki_client_resource_builders (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton)
        REFERENCES munki_client_resources (singleton) ON DELETE CASCADE,
    banner_object_id BIGINT NOT NULL
        REFERENCES storage_objects (id) ON DELETE RESTRICT,
    banner_fit TEXT NOT NULL CHECK (banner_fit IN ('height', 'cover')),
    banner_focal_x SMALLINT NOT NULL CHECK (banner_focal_x BETWEEN 0 AND 100),
    links JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(links) = 'array'),
    footer_text TEXT NOT NULL DEFAULT '',
    footer_links JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(footer_links) = 'array')
);

INSERT INTO munki_client_resource_builders (
    singleton,
    banner_object_id,
    banner_fit,
    banner_focal_x,
    links,
    footer_text,
    footer_links
)
SELECT
    singleton,
    banner_object_id,
    'height',
    CASE banner_alignment WHEN 'left' THEN 0 ELSE 50 END,
    links,
    footer_text,
    footer_links
FROM munki_client_resources;

ALTER TABLE munki_client_resources
    ADD COLUMN custom BOOLEAN NOT NULL DEFAULT FALSE,
    DROP COLUMN banner_object_id,
    DROP COLUMN banner_alignment,
    DROP COLUMN links,
    DROP COLUMN footer_text,
    DROP COLUMN footer_links;
