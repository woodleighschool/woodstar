-- +goose Up

ALTER TABLE storage_objects
    ADD COLUMN deletion_requested_at TIMESTAMPTZ;

CREATE INDEX storage_objects_deletion_requested_idx
    ON storage_objects (deletion_requested_at, id)
    WHERE deletion_requested_at IS NOT NULL;
