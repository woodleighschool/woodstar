-- +goose Up

CREATE TYPE osquery_report_snapshot_status AS ENUM ('collected', 'error');

ALTER TABLE osquery_report_snapshots
    RENAME COLUMN collected_at TO reported_at;

ALTER INDEX osquery_report_snapshots_report_collected_idx
    RENAME TO osquery_report_snapshots_report_reported_idx;

ALTER TABLE osquery_report_snapshots
    ADD COLUMN status osquery_report_snapshot_status NOT NULL DEFAULT 'collected',
    ADD COLUMN error TEXT NOT NULL DEFAULT '',
    ADD CONSTRAINT osquery_report_snapshots_observation_check CHECK (
        (status = 'collected' AND error = '')
        OR (status = 'error' AND NULLIF(btrim(error), '') IS NOT NULL)
    ),
    ADD CONSTRAINT osquery_report_snapshots_error_rows_check CHECK (
        status = 'collected' OR rows = '[]'::jsonb
    );
