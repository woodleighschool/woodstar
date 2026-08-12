package policies

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/randtoken"
)

const (
	DefaultRemediationTimeoutSeconds = 300
	remediationResponseGrace         = time.Minute
	remediationExecutionIDBytes      = 18
	maxRemediationOutputRunes        = 10_000
	maxPolicyErrorRunes              = 4_096
)

// PolicyRemediationRunStatus is the user-facing state of the latest run.
type PolicyRemediationRunStatus string

const (
	PolicyRemediationRunStatusQueued     PolicyRemediationRunStatus = "queued"
	PolicyRemediationRunStatusInProgress PolicyRemediationRunStatus = "in_progress"
	PolicyRemediationRunStatusSucceeded  PolicyRemediationRunStatus = "succeeded"
	PolicyRemediationRunStatusFailed     PolicyRemediationRunStatus = "failed"
	PolicyRemediationRunStatusNoResponse PolicyRemediationRunStatus = "no_response"
	PolicyRemediationRunStatusCancelled  PolicyRemediationRunStatus = "cancelled"
)

var policyRemediationRunStatusValues = []PolicyRemediationRunStatus{
	PolicyRemediationRunStatusQueued,
	PolicyRemediationRunStatusInProgress,
	PolicyRemediationRunStatusSucceeded,
	PolicyRemediationRunStatusFailed,
	PolicyRemediationRunStatusNoResponse,
	PolicyRemediationRunStatusCancelled,
}

func (PolicyRemediationRunStatus) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(policyRemediationRunStatusValues...)
}

// PolicyRemediationRunSummary is the non-sensitive latest-run projection.
type PolicyRemediationRunSummary struct {
	ExecutionID string                     `json:"-"`
	Status      PolicyRemediationRunStatus `json:"status"`
	Automatic   bool                       `json:"automatic"`
	QueuedAt    time.Time                  `json:"-"`
	StartedAt   *time.Time                 `json:"-"`
	CompletedAt *time.Time                 `json:"-"`
	ExitCode    *int                       `json:"-"`
}

// PolicyRemediationRun includes administrator-only execution output.
type PolicyRemediationRun struct {
	PolicyRemediationRunSummary

	Output         string `json:"output"`
	RuntimeSeconds *int   `json:"runtime_seconds,omitempty"`
	ExitCode       *int   `json:"exit_code,omitempty"`
	TimeoutSeconds int    `json:"-"`
}

// ClaimedRemediation is the immutable script returned to Orbit.
type ClaimedRemediation struct {
	HostID         int64
	ExecutionID    string
	ScriptContents string
	TimeoutSeconds int
}

// RemediationResult is Orbit's execution result.
type RemediationResult struct {
	ExecutionID    string
	Output         string
	RuntimeSeconds int
	ExitCode       int
}

func (s *Store) RemediationSource(ctx context.Context, policyID int64) (*PolicyRemediationSource, error) {
	var source PolicyRemediationSource
	err := s.pool.QueryRow(
		ctx,
		`SELECT remediation_script FROM osquery_policies WHERE id = $1`,
		policyID,
	).Scan(&source.Script)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fault.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &source, nil
}

// RunRemediation queues an explicit rerun without changing policy state.
func (s *Store) RunRemediation(
	ctx context.Context,
	policyID, hostID int64,
) (*PolicyRemediationRunSummary, error) {
	var summary *PolicyRemediationRunSummary
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var script string
		var revision int64
		var scriptExecutionAvailable bool
		var status PolicyStatus
		err := tx.QueryRow(ctx, `
			SELECT
				policy.remediation_script,
				policy.evaluation_revision,
				COALESCE(host.orbit_scripts_enabled, false)
					AND host.orbit_node_key <> '',
				membership.status::text
			FROM osquery_policies policy
			JOIN osquery_policy_assignments assignment
				ON assignment.policy_id = policy.id
			   AND assignment.host_id = $2
			JOIN osquery_policy_membership membership
				ON membership.policy_id = policy.id
			   AND membership.host_id = assignment.host_id
			JOIN hosts host ON host.id = assignment.host_id
			WHERE policy.id = $1
			FOR UPDATE OF policy, membership`, policyID, hostID).Scan(
			&script,
			&revision,
			&scriptExecutionAvailable,
			&status,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return fault.ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != PolicyStatusFail {
			return fmt.Errorf("%w: policy is not failing for this host", fault.ErrConflict)
		}
		if script == "" {
			return fmt.Errorf("%w: policy has no remediation script", fault.ErrConflict)
		}
		if !scriptExecutionAvailable {
			return fmt.Errorf("%w: host cannot execute Orbit scripts", fault.ErrConflict)
		}
		queued, err := enqueueRemediationTx(ctx, tx, policyID, hostID, revision, script, false)
		if err != nil {
			return err
		}
		if queued == nil {
			return fmt.Errorf("%w: remediation is already queued or in progress", fault.ErrConflict)
		}
		summary = queued
		return nil
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func enqueueRemediationTx(
	ctx context.Context,
	tx pgx.Tx,
	policyID, hostID int64,
	revision int64,
	script string,
	automatic bool,
) (*PolicyRemediationRunSummary, error) {
	executionID, err := randtoken.Generate(remediationExecutionIDBytes)
	if err != nil {
		return nil, fmt.Errorf("generate remediation execution ID: %w", err)
	}
	var row remediationRunRow
	err = tx.QueryRow(ctx, `
		INSERT INTO osquery_policy_remediation_runs (
			policy_id,
			host_id,
			execution_id,
			script_contents,
			evaluation_revision,
			automatic,
			timeout_seconds
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (policy_id, host_id) DO UPDATE SET
			execution_id = EXCLUDED.execution_id,
			script_contents = EXCLUDED.script_contents,
			evaluation_revision = EXCLUDED.evaluation_revision,
			automatic = EXCLUDED.automatic,
			queued_at = now(),
			claimed_at = NULL,
			reported_at = NULL,
			cancelled_at = NULL,
			output = '',
			runtime_seconds = NULL,
			exit_code = NULL,
			timeout_seconds = EXCLUDED.timeout_seconds
		WHERE osquery_policy_remediation_runs.reported_at IS NOT NULL
		   OR osquery_policy_remediation_runs.cancelled_at IS NOT NULL
		   OR (
			   osquery_policy_remediation_runs.claimed_at IS NOT NULL
			   AND osquery_policy_remediation_runs.reported_at IS NULL
			   AND osquery_policy_remediation_runs.claimed_at
			       + make_interval(
			           secs => osquery_policy_remediation_runs.timeout_seconds + $8
			       ) < now()
		   )
		RETURNING
			policy_id,
			host_id,
			execution_id,
			automatic,
			queued_at,
			claimed_at,
			reported_at,
			cancelled_at,
			output,
			runtime_seconds,
			exit_code,
			timeout_seconds`,
		policyID,
		hostID,
		executionID,
		script,
		revision,
		automatic,
		DefaultRemediationTimeoutSeconds,
		int(remediationResponseGrace.Seconds()),
	).Scan(
		&row.PolicyID,
		&row.HostID,
		&row.ExecutionID,
		&row.Automatic,
		&row.QueuedAt,
		&row.ClaimedAt,
		&row.ReportedAt,
		&row.CancelledAt,
		&row.Output,
		&row.RuntimeSeconds,
		&row.ExitCode,
		&row.TimeoutSeconds,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return remediationRunSummary(row, time.Now()), nil
}

func cancelQueuedRemediationTx(ctx context.Context, tx pgx.Tx, policyID, hostID int64) error {
	_, err := tx.Exec(ctx, `
		UPDATE osquery_policy_remediation_runs
		SET cancelled_at = now()
		WHERE policy_id = $1
		  AND host_id = $2
		  AND claimed_at IS NULL
		  AND reported_at IS NULL
		  AND cancelled_at IS NULL`, policyID, hostID)
	return err
}

func (s *Store) PendingRemediationExecutionIDs(ctx context.Context, hostID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT execution_id
		FROM osquery_policy_remediation_runs
		WHERE host_id = $1
		  AND claimed_at IS NULL
		  AND reported_at IS NULL
		  AND cancelled_at IS NULL
		ORDER BY queued_at, execution_id`, hostID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// ClaimRemediation atomically makes one execution ineligible for redelivery.
func (s *Store) ClaimRemediation(
	ctx context.Context,
	hostID int64,
	executionID string,
) (*ClaimedRemediation, error) {
	var claimed ClaimedRemediation
	err := s.pool.QueryRow(ctx, `
		UPDATE osquery_policy_remediation_runs
		SET claimed_at = now()
		WHERE host_id = $1
		  AND execution_id = $2
		  AND claimed_at IS NULL
		  AND reported_at IS NULL
		  AND cancelled_at IS NULL
		RETURNING host_id, execution_id, script_contents, timeout_seconds`,
		hostID,
		executionID,
	).Scan(
		&claimed.HostID,
		&claimed.ExecutionID,
		&claimed.ScriptContents,
		&claimed.TimeoutSeconds,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fault.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &claimed, nil
}

// RecordRemediationResult stores only the first report for a claimed execution.
func (s *Store) RecordRemediationResult(
	ctx context.Context,
	hostID int64,
	result RemediationResult,
) error {
	if result.ExecutionID == "" || result.RuntimeSeconds < 0 {
		return fault.ErrInvalidInput
	}
	output := truncateRunes(result.Output, maxRemediationOutputRunes)
	tag, err := s.pool.Exec(ctx, `
		UPDATE osquery_policy_remediation_runs
		SET
			reported_at = now(),
			output = $3,
			runtime_seconds = $4,
			exit_code = $5
		WHERE host_id = $1
		  AND execution_id = $2
		  AND claimed_at IS NOT NULL
		  AND reported_at IS NULL
		  AND cancelled_at IS NULL`,
		hostID,
		result.ExecutionID,
		output,
		result.RuntimeSeconds,
		result.ExitCode,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	var alreadyReported bool
	err = s.pool.QueryRow(ctx, `
		SELECT reported_at IS NOT NULL
		FROM osquery_policy_remediation_runs
		WHERE host_id = $1 AND execution_id = $2`, hostID, result.ExecutionID).Scan(&alreadyReported)
	if errors.Is(err, pgx.ErrNoRows) {
		return fault.ErrNotFound
	}
	if err != nil {
		return err
	}
	if alreadyReported {
		return nil
	}
	return fault.ErrConflict
}

func (s *Store) RemediationRun(
	ctx context.Context,
	policyID, hostID int64,
) (*PolicyRemediationRun, error) {
	row, err := s.remediationRun(ctx, policyID, hostID)
	if err != nil {
		return nil, err
	}
	summary := remediationRunSummary(row, time.Now())
	return &PolicyRemediationRun{
		PolicyRemediationRunSummary: *summary,
		Output:                      row.Output,
		RuntimeSeconds:              row.RuntimeSeconds,
		ExitCode:                    row.ExitCode,
		TimeoutSeconds:              row.TimeoutSeconds,
	}, nil
}

func (s *Store) remediationRun(
	ctx context.Context,
	policyID, hostID int64,
) (remediationRunRow, error) {
	var row remediationRunRow
	err := s.pool.QueryRow(ctx, `
		SELECT
			policy_id,
			host_id,
			execution_id,
			automatic,
			queued_at,
			claimed_at,
			reported_at,
			cancelled_at,
			output,
			runtime_seconds,
			exit_code,
			timeout_seconds
		FROM osquery_policy_remediation_runs
		WHERE policy_id = $1 AND host_id = $2`, policyID, hostID).Scan(
		&row.PolicyID,
		&row.HostID,
		&row.ExecutionID,
		&row.Automatic,
		&row.QueuedAt,
		&row.ClaimedAt,
		&row.ReportedAt,
		&row.CancelledAt,
		&row.Output,
		&row.RuntimeSeconds,
		&row.ExitCode,
		&row.TimeoutSeconds,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return remediationRunRow{}, fault.ErrNotFound
	}
	return row, err
}

type remediationRunRow struct {
	PolicyID       int64
	HostID         int64
	ExecutionID    string
	Automatic      bool
	QueuedAt       time.Time
	ClaimedAt      *time.Time
	ReportedAt     *time.Time
	CancelledAt    *time.Time
	Output         string
	RuntimeSeconds *int
	ExitCode       *int
	TimeoutSeconds int
}

func remediationRunSummaryFromRow(row policyHostStatusRow) *PolicyRemediationRunSummary {
	if row.ExecutionID == nil || row.RemediationAutomatic == nil ||
		row.RemediationQueuedAt == nil || row.RemediationTimeoutSeconds == nil {
		return nil
	}
	return remediationRunSummary(remediationRunRow{
		ExecutionID:    *row.ExecutionID,
		Automatic:      *row.RemediationAutomatic,
		QueuedAt:       *row.RemediationQueuedAt,
		ClaimedAt:      row.RemediationClaimedAt,
		ReportedAt:     row.RemediationReportedAt,
		CancelledAt:    row.RemediationCancelledAt,
		ExitCode:       row.RemediationExitCode,
		TimeoutSeconds: *row.RemediationTimeoutSeconds,
	}, time.Now())
}

func remediationRunSummary(row remediationRunRow, now time.Time) *PolicyRemediationRunSummary {
	status := PolicyRemediationRunStatusQueued
	switch {
	case row.CancelledAt != nil:
		status = PolicyRemediationRunStatusCancelled
	case row.ReportedAt != nil && row.ExitCode != nil && *row.ExitCode == 0:
		status = PolicyRemediationRunStatusSucceeded
	case row.ReportedAt != nil:
		status = PolicyRemediationRunStatusFailed
	case row.ClaimedAt != nil && now.After(
		row.ClaimedAt.Add(time.Duration(row.TimeoutSeconds)*time.Second+remediationResponseGrace),
	):
		status = PolicyRemediationRunStatusNoResponse
	case row.ClaimedAt != nil:
		status = PolicyRemediationRunStatusInProgress
	}
	return &PolicyRemediationRunSummary{
		ExecutionID: row.ExecutionID,
		Status:      status,
		Automatic:   row.Automatic,
		QueuedAt:    row.QueuedAt,
		StartedAt:   row.ClaimedAt,
		CompletedAt: row.ReportedAt,
		ExitCode:    row.ExitCode,
	}
}
