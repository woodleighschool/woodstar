-- +goose Up

DROP INDEX IF EXISTS munki_packages_uninstaller_object_idx;

ALTER TABLE munki_packages
    DROP COLUMN uninstaller_object_id;
