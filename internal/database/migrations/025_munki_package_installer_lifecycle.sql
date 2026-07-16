-- +goose Up

ALTER TABLE munki_packages
    DROP COLUMN eligible,
    ADD CONSTRAINT munki_packages_installer_object_check CHECK (
        (
            installer_type = 'nopkg'
            AND installer_object_id IS NULL
        )
        OR (
            installer_type IN ('pkg', 'copy_from_dmg')
            AND installer_object_id IS NOT NULL
        )
    );

DROP INDEX munki_packages_installer_object_idx;

CREATE UNIQUE INDEX munki_packages_installer_object_idx
    ON munki_packages (installer_object_id)
    WHERE installer_object_id IS NOT NULL;

ALTER TABLE storage_objects
    DROP CONSTRAINT storage_objects_state_check,
    ADD COLUMN multipart_upload_id TEXT
        CHECK (multipart_upload_id IS NULL OR btrim(multipart_upload_id) <> ''),
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
            AND multipart_upload_id IS NULL
        )
    );
