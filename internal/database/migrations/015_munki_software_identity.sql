-- +goose Up

ALTER TABLE munki_software
    ADD COLUMN display_name TEXT NOT NULL DEFAULT '';

UPDATE munki_software
SET display_name = name;

ALTER TABLE munki_software
    ADD CONSTRAINT munki_software_name_unique UNIQUE (name);

ALTER TABLE munki_software
    ALTER COLUMN display_name DROP DEFAULT;

-- +goose Down

ALTER TABLE munki_software
    DROP CONSTRAINT munki_software_name_unique,
    DROP COLUMN display_name;
