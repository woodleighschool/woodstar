-- +goose Up

DROP VIEW osquery_check_assignments;

ALTER TABLE osquery_checks RENAME TO osquery_policies;
ALTER SEQUENCE osquery_checks_id_seq RENAME TO osquery_policies_id_seq;
ALTER TABLE osquery_policies
    RENAME CONSTRAINT osquery_checks_pkey TO osquery_policies_pkey;
ALTER TABLE osquery_policies
    RENAME CONSTRAINT osquery_checks_name_key TO osquery_policies_name_key;
ALTER TABLE osquery_policies
    RENAME CONSTRAINT osquery_checks_created_by_user_id_fkey
    TO osquery_policies_created_by_user_id_fkey;

ALTER TABLE osquery_check_membership RENAME TO osquery_policy_membership;
ALTER TABLE osquery_policy_membership RENAME COLUMN check_id TO policy_id;
ALTER TABLE osquery_policy_membership
    RENAME CONSTRAINT osquery_check_membership_pkey TO osquery_policy_membership_pkey;
ALTER TABLE osquery_policy_membership
    RENAME CONSTRAINT osquery_check_membership_check_id_fkey
    TO osquery_policy_membership_policy_id_fkey;
ALTER TABLE osquery_policy_membership
    RENAME CONSTRAINT osquery_check_membership_host_id_fkey
    TO osquery_policy_membership_host_id_fkey;
ALTER INDEX osquery_check_membership_passes_idx
    RENAME TO osquery_policy_membership_passes_idx;

ALTER TABLE osquery_check_targets RENAME TO osquery_policy_targets;
ALTER TABLE osquery_policy_targets RENAME COLUMN check_id TO policy_id;
ALTER TABLE osquery_policy_targets
    RENAME CONSTRAINT osquery_check_targets_pkey TO osquery_policy_targets_pkey;
ALTER TABLE osquery_policy_targets
    RENAME CONSTRAINT osquery_check_targets_check_id_label_id_key
    TO osquery_policy_targets_policy_id_label_id_key;
ALTER TABLE osquery_policy_targets
    RENAME CONSTRAINT osquery_check_targets_check_id_fkey
    TO osquery_policy_targets_policy_id_fkey;
ALTER TABLE osquery_policy_targets
    RENAME CONSTRAINT osquery_check_targets_label_id_fkey
    TO osquery_policy_targets_label_id_fkey;
ALTER TABLE osquery_policy_targets
    RENAME CONSTRAINT osquery_check_targets_position_check
    TO osquery_policy_targets_position_check;
ALTER INDEX osquery_check_targets_label_idx RENAME TO osquery_policy_targets_label_idx;

CREATE VIEW osquery_policy_assignments AS
SELECT DISTINCT include_target.policy_id, membership.host_id
FROM osquery_policy_targets include_target
JOIN label_membership membership
    ON membership.label_id = include_target.label_id
WHERE include_target.direction = 'include'
  AND NOT EXISTS (
      SELECT 1
      FROM osquery_policy_targets exclude_target
      JOIN label_membership excluded
          ON excluded.label_id = exclude_target.label_id
         AND excluded.host_id = membership.host_id
      WHERE exclude_target.policy_id = include_target.policy_id
        AND exclude_target.direction = 'exclude'
  );

CREATE TYPE osquery_policy_status AS ENUM ('pending', 'pass', 'fail', 'error');

ALTER TABLE osquery_policies
    ADD COLUMN resolution TEXT NOT NULL DEFAULT '',
    ADD COLUMN remediation_script TEXT NOT NULL DEFAULT '',
    ADD COLUMN automatic_remediation_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN evaluation_revision BIGINT NOT NULL DEFAULT 1,
    ADD CONSTRAINT osquery_policies_automatic_remediation_check CHECK (
        NOT automatic_remediation_enabled
        OR NULLIF(btrim(remediation_script), '') IS NOT NULL
    ),
    ADD CONSTRAINT osquery_policies_evaluation_revision_check CHECK (
        evaluation_revision > 0
    );

ALTER TABLE osquery_policy_membership
    ADD COLUMN status osquery_policy_status NOT NULL DEFAULT 'pending',
    ADD COLUMN last_conclusive_passes BOOLEAN,
    ADD COLUMN error TEXT NOT NULL DEFAULT '',
    ADD COLUMN last_issued_sequence BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN last_completed_sequence BIGINT NOT NULL DEFAULT 0,
    ADD CONSTRAINT osquery_policy_membership_sequence_check CHECK (
        last_issued_sequence >= 0
        AND last_completed_sequence >= 0
        AND last_completed_sequence <= last_issued_sequence
    );

UPDATE osquery_policy_membership
SET
    status = CASE
        WHEN passes IS TRUE THEN 'pass'::osquery_policy_status
        WHEN passes IS FALSE THEN 'fail'::osquery_policy_status
        ELSE 'pending'::osquery_policy_status
    END,
    last_conclusive_passes = passes;

DROP INDEX osquery_policy_membership_passes_idx;
ALTER TABLE osquery_policy_membership DROP COLUMN passes;
CREATE INDEX osquery_policy_membership_status_idx
    ON osquery_policy_membership (policy_id, status);

ALTER TABLE hosts ADD COLUMN orbit_scripts_enabled BOOLEAN;

CREATE TABLE osquery_policy_remediation_runs (
    policy_id BIGINT NOT NULL REFERENCES osquery_policies (id) ON DELETE CASCADE,
    host_id BIGINT NOT NULL REFERENCES hosts (id) ON DELETE CASCADE,
    execution_id TEXT NOT NULL UNIQUE,
    script_contents TEXT NOT NULL,
    evaluation_revision BIGINT NOT NULL,
    automatic BOOLEAN NOT NULL,
    queued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    claimed_at TIMESTAMPTZ,
    reported_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    output TEXT NOT NULL DEFAULT '',
    runtime_seconds INTEGER,
    exit_code INTEGER,
    timeout_seconds INTEGER NOT NULL,
    PRIMARY KEY (policy_id, host_id),
    CHECK (NULLIF(btrim(script_contents), '') IS NOT NULL),
    CHECK (evaluation_revision > 0),
    CHECK (runtime_seconds IS NULL OR runtime_seconds >= 0),
    CHECK (timeout_seconds > 0),
    CHECK (reported_at IS NULL OR claimed_at IS NOT NULL),
    CHECK (cancelled_at IS NULL OR claimed_at IS NULL)
);

CREATE INDEX osquery_policy_remediation_runs_pending_idx
    ON osquery_policy_remediation_runs (host_id, queued_at)
    WHERE claimed_at IS NULL AND reported_at IS NULL AND cancelled_at IS NULL;
