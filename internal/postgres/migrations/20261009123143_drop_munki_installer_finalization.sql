-- +goose Up

-- River's tables exist only on a database that an earlier release migrated;
-- a new database creates them after these migrations.
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('river_job') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS munki_installer_finalization_finished ON river_job;
        DELETE FROM river_job WHERE kind = 'munki_finalize_installer';
    END IF;
END;
$$;
-- +goose StatementEnd

DROP FUNCTION IF EXISTS finish_munki_installer_finalization();
DROP TABLE IF EXISTS munki_installer_finalization_claims;
DROP TABLE IF EXISTS munki_installer_finalizations;
