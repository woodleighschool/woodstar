-- +goose Up
CREATE TABLE munki_installer_finalizations (
    object_id bigint PRIMARY KEY REFERENCES storage_objects(id) ON DELETE CASCADE,
    job_id bigint UNIQUE NOT NULL,
    state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'completed', 'failed'))
);

CREATE TABLE munki_installer_finalization_claims (
    job_id bigint PRIMARY KEY REFERENCES river_job(id) ON DELETE CASCADE,
    object_id bigint UNIQUE NOT NULL REFERENCES storage_objects(id) ON DELETE RESTRICT
);

-- Retain the outcome beyond River's job-history retention, including jobs
-- discarded by rescue after a worker exits on its last attempt.
-- +goose StatementBegin
CREATE FUNCTION finish_munki_installer_finalization() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        UPDATE munki_installer_finalizations
        SET state = 'failed'
        WHERE job_id = OLD.id AND state = 'pending';
        RETURN OLD;
    END IF;
    IF NEW.state IN ('completed', 'discarded', 'cancelled') THEN
        UPDATE munki_installer_finalizations
        SET state = CASE WHEN NEW.state = 'completed' THEN 'completed' ELSE 'failed' END
        WHERE job_id = NEW.id AND state = 'pending';
        DELETE FROM munki_installer_finalization_claims WHERE job_id = NEW.id;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER munki_installer_finalization_finished
AFTER UPDATE OF state OR DELETE ON river_job
FOR EACH ROW EXECUTE FUNCTION finish_munki_installer_finalization();
