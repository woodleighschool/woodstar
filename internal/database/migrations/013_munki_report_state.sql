-- +goose Up

ALTER TABLE munki_host_status
    ADD COLUMN last_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN last_successful_at TIMESTAMPTZ,
    ADD COLUMN collection_error TEXT NOT NULL DEFAULT '',
    ADD COLUMN has_report BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE munki_host_status
SET last_attempt_at = COALESCE(run_ended_at, run_started_at, now()),
    last_successful_at = COALESCE(run_ended_at, run_started_at, now()),
    has_report = TRUE;

-- +goose Down

ALTER TABLE munki_host_status
    DROP COLUMN has_report,
    DROP COLUMN collection_error,
    DROP COLUMN last_successful_at,
    DROP COLUMN last_attempt_at;
