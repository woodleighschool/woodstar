-- +goose Up

CREATE TABLE osquery_live_query_runs (
    id BIGSERIAL PRIMARY KEY,
    query TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deadline_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    CHECK (deadline_at > started_at),
    CHECK (completed_at IS NULL OR completed_at >= started_at)
);

CREATE TABLE osquery_live_query_targets (
    run_id BIGINT NOT NULL
        REFERENCES osquery_live_query_runs (id) ON DELETE CASCADE,
    -- Targets are a short-lived snapshot and must survive canonical host deletion.
    host_id BIGINT NOT NULL,
    host_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'collected', 'error', 'stopped')),
    rows JSONB NOT NULL DEFAULT '[]'::JSONB
        CHECK (jsonb_typeof(rows) = 'array'),
    error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, host_id)
);

CREATE INDEX osquery_live_query_targets_pending_host_idx
    ON osquery_live_query_targets (host_id, run_id)
    WHERE status = 'pending';
