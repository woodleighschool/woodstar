-- +goose Up
ALTER TABLE osquery_policy_remediation_runs
    ALTER COLUMN timeout_seconds SET DEFAULT 300;
