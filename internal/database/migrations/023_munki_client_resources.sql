-- +goose Up

CREATE TABLE munki_client_resources (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    banner_object_id BIGINT NOT NULL
        REFERENCES storage_objects (id) ON DELETE RESTRICT,
    archive_object_id BIGINT NOT NULL
        REFERENCES storage_objects (id) ON DELETE RESTRICT,
    banner_alignment TEXT NOT NULL CHECK (banner_alignment IN ('left', 'center')),
    links JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(links) = 'array'),
    footer_text TEXT NOT NULL DEFAULT '',
    footer_links JSONB NOT NULL DEFAULT '[]'::JSONB CHECK (jsonb_typeof(footer_links) = 'array'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
