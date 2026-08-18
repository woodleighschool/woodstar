-- +goose Up
ALTER TABLE osquery_policy_remediation_runs
    ALTER COLUMN timeout_seconds SET DEFAULT 300;

-- +goose Down
ALTER TABLE osquery_policy_remediation_runs
    ALTER COLUMN timeout_seconds DROP DEFAULT;
