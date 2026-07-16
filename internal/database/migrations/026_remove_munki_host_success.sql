-- +goose Up

ALTER TABLE munki_host_status
    DROP COLUMN success;
