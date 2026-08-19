-- +goose Up

-- Remediation executions are transient current state. Existing rows used GET as
-- a claim and cannot be migrated reliably into the read-only delivery model.
TRUNCATE TABLE osquery_policy_remediation_runs;

DROP INDEX osquery_policy_remediation_runs_pending_idx;

ALTER TABLE osquery_policies
    ADD COLUMN remediation_revision BIGINT NOT NULL DEFAULT 1,
    ADD CONSTRAINT osquery_policies_remediation_revision_check CHECK (
        remediation_revision > 0
    );

ALTER TABLE osquery_policy_membership
    ADD COLUMN remediation_failure_sequence BIGINT NOT NULL DEFAULT 0,
    ADD CONSTRAINT osquery_policy_membership_remediation_failure_sequence_check CHECK (
        remediation_failure_sequence >= 0
    );

UPDATE osquery_policy_membership
SET remediation_failure_sequence = last_completed_sequence
WHERE status = 'fail';

ALTER TABLE osquery_policy_remediation_runs
    DROP COLUMN claimed_at,
    DROP COLUMN cancelled_at,
    DROP COLUMN timeout_seconds,
    DROP COLUMN evaluation_revision,
    ADD COLUMN remediation_revision BIGINT NOT NULL,
    ADD COLUMN failure_sequence BIGINT NOT NULL,
    ADD CONSTRAINT osquery_policy_remediation_runs_remediation_revision_check CHECK (
        remediation_revision > 0
    ),
    ADD CONSTRAINT osquery_policy_remediation_runs_failure_sequence_check CHECK (
        failure_sequence > 0
    ),
    ADD CONSTRAINT osquery_policy_remediation_runs_membership_fkey
        FOREIGN KEY (policy_id, host_id)
        REFERENCES osquery_policy_membership (policy_id, host_id)
        ON DELETE CASCADE;

CREATE INDEX osquery_policy_remediation_runs_pending_idx
    ON osquery_policy_remediation_runs (host_id, queued_at, execution_id)
    WHERE reported_at IS NULL;
