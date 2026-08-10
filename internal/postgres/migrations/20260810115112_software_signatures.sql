-- +goose Up

ALTER TABLE host_software_installed_paths
    RENAME COLUMN cdhash_sha256 TO cdhash;

ALTER TABLE host_software_installed_paths
    ADD COLUMN signature_valid BOOLEAN;

UPDATE host_software_installed_paths
SET
    identifier = '',
    signing_authority = '',
    team_identifier = '',
    cdhash = NULL;
