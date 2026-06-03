-- +goose Up

CREATE TYPE santa_file_access_decision AS ENUM (
    'unknown',
    'denied',
    'denied_invalid_signature',
    'audit_only'
);

DROP INDEX santa_execution_events_host_time_idx;

ALTER TABLE santa_execution_events
    ALTER COLUMN occurred_at SET NOT NULL;

CREATE INDEX santa_execution_events_host_time_idx
    ON santa_execution_events (host_id, occurred_at DESC);
CREATE INDEX santa_execution_events_executable_time_idx
    ON santa_execution_events (executable_id, occurred_at DESC);
CREATE INDEX santa_execution_events_user_time_idx
    ON santa_execution_events (executing_user, occurred_at DESC);

CREATE INDEX santa_executables_file_name_idx ON santa_executables (file_name);
CREATE INDEX santa_executables_signing_id_idx ON santa_executables (signing_id);
CREATE INDEX santa_executables_team_id_idx ON santa_executables (team_id);
CREATE INDEX santa_executables_cdhash_idx ON santa_executables (cdhash);

CREATE TABLE santa_file_access_events (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    rule_version TEXT NOT NULL DEFAULT '',
    rule_name TEXT NOT NULL DEFAULT '',
    target TEXT NOT NULL DEFAULT '',
    decision santa_file_access_decision NOT NULL,
    process_chain JSONB NOT NULL DEFAULT '[]'::JSONB,
    occurred_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (jsonb_typeof(process_chain) = 'array')
);

CREATE INDEX santa_file_access_events_host_time_idx
    ON santa_file_access_events (host_id, occurred_at DESC);
CREATE INDEX santa_file_access_events_decision_ingested_idx
    ON santa_file_access_events (decision, ingested_at DESC);
