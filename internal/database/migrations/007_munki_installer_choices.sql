-- +goose Up

ALTER TABLE munki_packages
    ALTER COLUMN installer_choices_xml DROP DEFAULT,
    ALTER COLUMN installer_choices_xml TYPE JSONB
        USING '[]'::JSONB,
    ALTER COLUMN installer_choices_xml SET DEFAULT '[]'::JSONB;

ALTER TABLE munki_packages
    ADD CONSTRAINT munki_packages_installer_choices_xml_array
    CHECK (jsonb_typeof(installer_choices_xml) = 'array');

-- +goose Down

ALTER TABLE munki_packages
    DROP CONSTRAINT munki_packages_installer_choices_xml_array;

ALTER TABLE munki_packages
    ALTER COLUMN installer_choices_xml DROP DEFAULT,
    ALTER COLUMN installer_choices_xml TYPE TEXT
        USING '',
    ALTER COLUMN installer_choices_xml SET DEFAULT '';
