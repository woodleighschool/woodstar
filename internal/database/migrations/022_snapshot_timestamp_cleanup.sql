-- +goose Up

ALTER TABLE munki_host_status
    DROP COLUMN last_seen_at,
    DROP COLUMN updated_at;

ALTER TABLE munki_host_items
    DROP COLUMN run_ended_at,
    DROP COLUMN last_seen_at,
    DROP COLUMN updated_at;

ALTER TABLE host_software
    DROP COLUMN last_seen_at;

ALTER TABLE host_software_installed_paths
    DROP COLUMN last_seen_at;

ALTER TABLE host_users
    DROP COLUMN created_at,
    DROP COLUMN updated_at;

ALTER TABLE host_batteries
    DROP COLUMN created_at,
    DROP COLUMN updated_at;

ALTER TABLE host_certificates
    DROP COLUMN created_at,
    DROP COLUMN updated_at;
