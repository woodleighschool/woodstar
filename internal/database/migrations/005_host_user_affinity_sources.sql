-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'host_user_affinity_source') THEN
        CREATE TYPE host_user_affinity_source AS ENUM ('manual', 'orbit_profile', 'santa_primary_user');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'host_directory_user_source') THEN
        CREATE TYPE host_directory_user_source AS ENUM ('manual', 'reported_user_affinity');
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE IF EXISTS host_emails
    ALTER COLUMN source TYPE host_user_affinity_source
    USING source::host_user_affinity_source;

ALTER TABLE IF EXISTS host_user_affinity_mappings
    ALTER COLUMN source TYPE host_user_affinity_source
    USING source::host_user_affinity_source;

ALTER TABLE host_directory_user
    DROP CONSTRAINT IF EXISTS host_directory_user_source_check;

UPDATE host_directory_user
SET source = 'reported_user_affinity'
WHERE source::text = 'mdm_email';

ALTER TABLE host_directory_user
    ALTER COLUMN source TYPE host_directory_user_source
    USING source::host_directory_user_source;

-- +goose Down

ALTER TABLE host_directory_user
    ALTER COLUMN source TYPE TEXT
    USING CASE
        WHEN source::text = 'reported_user_affinity' THEN 'mdm_email'
        ELSE source::text
    END;

ALTER TABLE host_directory_user
    ADD CONSTRAINT host_directory_user_source_check CHECK (source IN ('manual', 'mdm_email'));

ALTER TABLE IF EXISTS host_emails
    ALTER COLUMN source TYPE TEXT
    USING source::text;

ALTER TABLE IF EXISTS host_user_affinity_mappings
    ALTER COLUMN source TYPE TEXT
    USING source::text;

DROP TYPE IF EXISTS host_directory_user_source;
DROP TYPE IF EXISTS host_user_affinity_source;
