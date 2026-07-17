-- +goose Up

ALTER TABLE munki_packages
    ADD COLUMN uninstallable BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE munki_packages
SET uninstallable = uninstall_method <> '';
