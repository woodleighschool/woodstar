-- +goose Up

-- Pending values came from upload requests and are not authoritative.
UPDATE storage_objects
SET content_type = '',
    size_bytes = NULL,
    sha256 = NULL
WHERE available_at IS NULL;

-- Completed rows must carry representation metadata. Preserve existing values.
UPDATE storage_objects
SET content_type = 'application/octet-stream'
WHERE available_at IS NOT NULL
  AND btrim(content_type) = '';

ALTER TABLE storage_objects
    ADD CONSTRAINT storage_objects_state_check CHECK (
        (
            available_at IS NULL
            AND content_type = ''
            AND size_bytes IS NULL
            AND sha256 IS NULL
        )
        OR (
            available_at IS NOT NULL
            AND content_type <> ''
            AND size_bytes IS NOT NULL
            AND sha256 IS NOT NULL
        )
    );
