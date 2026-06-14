-- +goose Up

CREATE TABLE storage_objects (
    id BIGSERIAL PRIMARY KEY,
    prefix TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT CHECK (size_bytes IS NULL OR size_bytes >= 0),
    sha256 TEXT CHECK (sha256 IS NULL OR sha256 ~ '^[0-9a-f]{64}$'),
    available_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX storage_objects_prefix_idx
    ON storage_objects (prefix);

-- +goose Down

DROP TABLE storage_objects;
