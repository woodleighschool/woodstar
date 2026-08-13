-- +goose Up
ALTER TABLE host_software_installed_paths
    RENAME COLUMN signature_valid TO signature_signed;
