-- +goose Up

ALTER TABLE munki_software
    ALTER COLUMN display_name DROP NOT NULL;

UPDATE munki_software
SET display_name = NULL
WHERE display_name = name;
