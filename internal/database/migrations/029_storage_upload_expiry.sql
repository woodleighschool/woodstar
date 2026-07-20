-- +goose Up

ALTER TABLE storage_objects
    ADD COLUMN expired_at TIMESTAMPTZ,
    ADD CONSTRAINT storage_objects_expiry_check CHECK (
        expired_at IS NULL OR available_at IS NULL
    );

CREATE INDEX storage_objects_pending_expiry_idx
    ON storage_objects (updated_at, id)
    WHERE available_at IS NULL;
