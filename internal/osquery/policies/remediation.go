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
	remediationExecutionIDBytes = 18
	maxRemediationOutputRunes   = 10_000
	maxPolicyErrorRunes         = 4_096
)

// PolicyRemediationRunStatus is the user-facing state of the current run.
type PolicyRemediationRunStatus string

const (
	PolicyRemediationRunStatusQueued    PolicyRemediationRunStatus = "queued"
	PolicyRemediationRunStatusSucceeded PolicyRemediationRunStatus = "succeeded"
	PolicyRemediationRunStatusFailed    PolicyRemediationRunStatus = "failed"
)

var policyRemediationRunStatusValues = []PolicyRemediationRunStatus{
	PolicyRemediationRunStatusQueued,
	PolicyRemediationRunStatusSucceeded,
	PolicyRemediationRunStatusFailed,
}

func (PolicyRemediationRunStatus) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(policyRemediationRunStatusValues...)
}

// PolicyRemediationStatusFilter is a current-run state accepted by result listings.
type PolicyRemediationStatusFilter string

const (
	PolicyRemediationFilterNotRun    PolicyRemediationStatusFilter = "not_run"
	PolicyRemediationFilterQueued    PolicyRemediationStatusFilter = "queued"
	PolicyRemediationFilterSucceeded PolicyRemediationStatusFilter = "succeeded"
	PolicyRemediationFilterFailed    PolicyRemediationStatusFilter = "failed"
)

var policyRemediationStatusFilterValues = []PolicyRemediationStatusFilter{
	PolicyRemediationFilterNotRun,
	PolicyRemediationFilterQueued,
	PolicyRemediationFilterSucceeded,
	PolicyRemediationFilterFailed,
}

func (PolicyRemediationStatusFilter) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(policyRemediationStatusFilterValues...)
}

// PolicyRemediationRunSummary is the non-sensitive current-run projection.
type PolicyRemediationRunSummary struct {
	ExecutionID string                     `json:"-"`
	Status      PolicyRemediationRunStatus `json:"status"`
	Automatic   bool                       `json:"automatic"`
	QueuedAt    time.Time                  `json:"-"`
	CompletedAt *time.Time                 `json:"-"`
	ExitCode    *int                       `json:"-"`
}

// PolicyRemediationBatchSummary reports an explicit batch request.
type PolicyRemediationBatchSummary struct {
	Queued  int `json:"queued"`
	Skipped int `json:"skipped"`
}

// PolicyRemediationRun includes administrator-only execution output.
type PolicyRemediationRun struct {
	PolicyRemediationRunSummary

	Output         string `json:"output"`
	RuntimeSeconds *int   `json:"runtime_seconds,omitempty"`
	ExitCode       *int   `json:"exit_code,omitempty"`
}

// RemediationExecution is the immutable script and optional terminal result returned to Orbit.
type RemediationExecution struct {
	HostID         int64
	ExecutionID    string
	ScriptContents string
	Output         string
	RuntimeSeconds *int
	ExitCode       *int
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

// RunRemediations queues the current script for selected failing hosts or every
// current failure when explicitly requested.
func (s *Store) RunRemediations(
	ctx context.Context,
	policyID int64,
	hostIDs []int64,
	allFailures bool,
) (*PolicyRemediationBatchSummary, error) {
	hostIDs, err := normalizeRemediationHostIDs(hostIDs)
	if err != nil {
		return nil, err
	}
	if allFailures == (len(hostIDs) > 0) {
		return nil, fmt.Errorf(
			"%w: choose either all failures or one or more hosts",
			fault.ErrInvalidInput,
		)
	}

	summary := &PolicyRemediationBatchSummary{}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var script string
		var remediationRevision int64
		err := tx.QueryRow(ctx, `
			SELECT remediation_script, remediation_revision
			FROM osquery_policies
			WHERE id = $1
			FOR UPDATE`, policyID).Scan(&script, &remediationRevision)
		if errors.Is(err, pgx.ErrNoRows) {
			return fault.ErrNotFound
		}
		if err != nil {
			return err
		}
		if script == "" {
			return fmt.Errorf("%w: policy has no remediation script", fault.ErrConflict)
		}

		rows, err := tx.Query(ctx, `
			SELECT
				assignment.host_id,
				membership.remediation_failure_sequence,
				COALESCE(host.orbit_scripts_enabled, false)
					AND host.orbit_node_key <> '' AS eligible
			FROM osquery_policy_assignments assignment
			JOIN osquery_policy_membership membership
				ON membership.policy_id = assignment.policy_id
			   AND membership.host_id = assignment.host_id
			JOIN hosts host ON host.id = assignment.host_id
			WHERE assignment.policy_id = $1
			  AND membership.status = 'fail'
			  AND ($3 OR assignment.host_id = ANY($2::bigint[]))
			ORDER BY assignment.host_id
			FOR UPDATE OF membership`, policyID, hostIDs, allFailures)
		if err != nil {
			return err
		}
		candidates, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (remediationCandidate, error) {
			var candidate remediationCandidate
			err := row.Scan(&candidate.HostID, &candidate.FailureSequence, &candidate.Eligible)
			return candidate, err
		})
		if err != nil {
			return err
		}

		queueHostIDs := make([]int64, 0, len(candidates))
		executionIDs := make([]string, 0, len(candidates))
		failureSequences := make([]int64, 0, len(candidates))
		for _, candidate := range candidates {
			if !candidate.Eligible {
				continue
			}
			executionID, err := randtoken.Generate(remediationExecutionIDBytes)
			if err != nil {
				return fmt.Errorf("generate remediation execution ID: %w", err)
			}
			queueHostIDs = append(queueHostIDs, candidate.HostID)
			executionIDs = append(executionIDs, executionID)
			failureSequences = append(failureSequences, candidate.FailureSequence)
		}
		queued, err := s.enqueueRemediationsTx(
			ctx,
			tx,
			policyID,
			queueHostIDs,
			executionIDs,
			failureSequences,
			remediationRevision,
			script,
			false,
		)
		if err != nil {
			return err
		}
		summary.Queued = len(queued)
		if allFailures {
			summary.Skipped = len(candidates) - summary.Queued
		} else {
			summary.Skipped = len(hostIDs) - summary.Queued
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func normalizeRemediationHostIDs(hostIDs []int64) ([]int64, error) {
	normalized := make([]int64, 0, len(hostIDs))
	seen := make(map[int64]struct{}, len(hostIDs))
	for _, hostID := range hostIDs {
		if hostID <= 0 {
			return nil, fmt.Errorf("%w: host IDs must be positive", fault.ErrInvalidInput)
		}
		if _, exists := seen[hostID]; exists {
			continue
		}
		seen[hostID] = struct{}{}
		normalized = append(normalized, hostID)
	}
	return normalized, nil
}

type remediationCandidate struct {
	HostID          int64
	FailureSequence int64
	Eligible        bool
}

func (s *Store) enqueueRemediationTx(
	ctx context.Context,
	tx pgx.Tx,
	policyID, hostID int64,
	remediationRevision, failureSequence int64,
	script string,
	automatic bool,
) (*PolicyRemediationRunSummary, error) {
	executionID, err := randtoken.Generate(remediationExecutionIDBytes)
	if err != nil {
		return nil, fmt.Errorf("generate remediation execution ID: %w", err)
	}
	rows, err := s.enqueueRemediationsTx(
		ctx,
		tx,
		policyID,
		[]int64{hostID},
		[]string{executionID},
		[]int64{failureSequence},
		remediationRevision,
		script,
		automatic,
	)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return remediationRunSummary(rows[0]), nil
}

func (s *Store) enqueueRemediationsTx(
	ctx context.Context,
	tx pgx.Tx,
	policyID int64,
	hostIDs []int64,
	executionIDs []string,
	failureSequences []int64,
	remediationRevision int64,
	script string,
	automatic bool,
) ([]remediationRunRow, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
		INSERT INTO osquery_policy_remediation_runs (
			policy_id,
			host_id,
			execution_id,
			script_contents,
			remediation_revision,
			failure_sequence,
			automatic
		)
		SELECT
			$1,
			batch.host_id,
			batch.execution_id,
			$5,
			$6,
			batch.failure_sequence,
			$7
		FROM unnest($2::bigint[], $3::text[], $4::bigint[])
			AS batch(host_id, execution_id, failure_sequence)
		ON CONFLICT (policy_id, host_id) DO UPDATE SET
			execution_id = EXCLUDED.execution_id,
			script_contents = EXCLUDED.script_contents,
			remediation_revision = EXCLUDED.remediation_revision,
			failure_sequence = EXCLUDED.failure_sequence,
			automatic = EXCLUDED.automatic,
			queued_at = now(),
			reported_at = NULL,
			output = '',
			runtime_seconds = NULL,
			exit_code = NULL
		WHERE osquery_policy_remediation_runs.reported_at IS NOT NULL
		   OR osquery_policy_remediation_runs.remediation_revision
			  <> EXCLUDED.remediation_revision
		   OR osquery_policy_remediation_runs.failure_sequence
			  <> EXCLUDED.failure_sequence
		   OR osquery_policy_remediation_runs.script_contents
			  IS DISTINCT FROM EXCLUDED.script_contents
		RETURNING
			policy_id,
			host_id,
			execution_id,
			automatic,
			queued_at,
			reported_at,
			output,
			runtime_seconds,
			exit_code`,
		policyID,
		hostIDs,
		executionIDs,
		failureSequences,
		script,
		remediationRevision,
		automatic,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (remediationRunRow, error) {
		var run remediationRunRow
		err := row.Scan(
			&run.PolicyID,
			&run.HostID,
			&run.ExecutionID,
			&run.Automatic,
			&run.QueuedAt,
			&run.ReportedAt,
			&run.Output,
			&run.RuntimeSeconds,
			&run.ExitCode,
		)
		return run, err
	})
}

func (s *Store) PendingRemediationExecutionIDs(ctx context.Context, hostID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT run.execution_id
		FROM osquery_policy_remediation_runs run
		JOIN osquery_policies policy ON policy.id = run.policy_id
		JOIN osquery_policy_membership membership
			ON membership.policy_id = run.policy_id
		   AND membership.host_id = run.host_id
		JOIN osquery_policy_assignments assignment
			ON assignment.policy_id = run.policy_id
		   AND assignment.host_id = run.host_id
		WHERE run.host_id = $1
		  AND run.reported_at IS NULL
		  AND membership.status = 'fail'
		  AND run.remediation_revision = policy.remediation_revision
		  AND run.failure_sequence = membership.remediation_failure_sequence
		  AND run.script_contents = policy.remediation_script
		ORDER BY run.queued_at, run.execution_id`, hostID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// RemediationExecution returns an immutable execution without consuming it.
func (s *Store) RemediationExecution(
	ctx context.Context,
	hostID int64,
	executionID string,
) (*RemediationExecution, error) {
	var execution RemediationExecution
	err := s.pool.QueryRow(ctx, `
		SELECT
			run.host_id,
			run.execution_id,
			run.script_contents,
			run.output,
			run.runtime_seconds,
			run.exit_code
		FROM osquery_policy_remediation_runs run
		JOIN osquery_policies policy ON policy.id = run.policy_id
		JOIN osquery_policy_membership membership
			ON membership.policy_id = run.policy_id
		   AND membership.host_id = run.host_id
		JOIN osquery_policy_assignments assignment
			ON assignment.policy_id = run.policy_id
		   AND assignment.host_id = run.host_id
		WHERE run.host_id = $1
		  AND run.execution_id = $2
		  AND (
			run.reported_at IS NOT NULL
			OR (
				membership.status = 'fail'
				AND run.remediation_revision = policy.remediation_revision
				AND run.failure_sequence = membership.remediation_failure_sequence
				AND run.script_contents = policy.remediation_script
			)
		  )`,
		hostID,
		executionID,
	).Scan(
		&execution.HostID,
		&execution.ExecutionID,
		&execution.ScriptContents,
		&execution.Output,
		&execution.RuntimeSeconds,
		&execution.ExitCode,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fault.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// RecordRemediationResult stores only the first terminal report.
func (s *Store) RecordRemediationResult(
	ctx context.Context,
	hostID int64,
	result RemediationResult,
) error {
	if result.ExecutionID == "" || result.RuntimeSeconds < 0 {
		return fault.ErrInvalidInput
	}
	output := truncateRunes(result.Output, maxRemediationOutputRunes)
	_, err := s.pool.Exec(ctx, `
		UPDATE osquery_policy_remediation_runs
		SET
			reported_at = now(),
			output = $3,
			runtime_seconds = $4,
			exit_code = $5
		WHERE host_id = $1
		  AND execution_id = $2
		  AND reported_at IS NULL`,
		hostID,
		result.ExecutionID,
		output,
		result.RuntimeSeconds,
		result.ExitCode,
	)
	if err != nil {
		return err
	}
	return err
}

func (s *Store) RemediationRun(
	ctx context.Context,
	policyID, hostID int64,
) (*PolicyRemediationRun, error) {
	row, err := s.remediationRun(ctx, policyID, hostID)
	if err != nil {
		return nil, err
	}
	summary := remediationRunSummary(row)
	return &PolicyRemediationRun{
		PolicyRemediationRunSummary: *summary,
		Output:                      row.Output,
		RuntimeSeconds:              row.RuntimeSeconds,
		ExitCode:                    row.ExitCode,
	}, nil
}

func (s *Store) remediationRun(
	ctx context.Context,
	policyID, hostID int64,
) (remediationRunRow, error) {
	var row remediationRunRow
	err := s.pool.QueryRow(ctx, `
		SELECT
			run.policy_id,
			run.host_id,
			run.execution_id,
			run.automatic,
			run.queued_at,
			run.reported_at,
			run.output,
			run.runtime_seconds,
			run.exit_code
		FROM osquery_policy_remediation_runs run
		JOIN osquery_policies policy ON policy.id = run.policy_id
		JOIN osquery_policy_membership membership
			ON membership.policy_id = run.policy_id
		   AND membership.host_id = run.host_id
		JOIN osquery_policy_assignments assignment
			ON assignment.policy_id = run.policy_id
		   AND assignment.host_id = run.host_id
		WHERE run.policy_id = $1
		  AND run.host_id = $2
		  AND membership.status = 'fail'
		  AND run.remediation_revision = policy.remediation_revision
		  AND run.failure_sequence = membership.remediation_failure_sequence
		  AND run.script_contents = policy.remediation_script`, policyID, hostID).Scan(
		&row.PolicyID,
		&row.HostID,
		&row.ExecutionID,
		&row.Automatic,
		&row.QueuedAt,
		&row.ReportedAt,
		&row.Output,
		&row.RuntimeSeconds,
		&row.ExitCode,
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
	ReportedAt     *time.Time
	Output         string
	RuntimeSeconds *int
	ExitCode       *int
}

func remediationRunSummaryFromRow(row policyHostStatusRow) *PolicyRemediationRunSummary {
	if row.ExecutionID == nil || row.RemediationAutomatic == nil ||
		row.RemediationQueuedAt == nil {
		return nil
	}
	return &PolicyRemediationRunSummary{
		ExecutionID: *row.ExecutionID,
		Status:      PolicyRemediationRunStatus(row.RemediationStatus),
		Automatic:   *row.RemediationAutomatic,
		QueuedAt:    *row.RemediationQueuedAt,
		CompletedAt: row.RemediationReportedAt,
		ExitCode:    row.RemediationExitCode,
	}
}

func remediationRunSummary(row remediationRunRow) *PolicyRemediationRunSummary {
	status := PolicyRemediationRunStatusQueued
	switch {
	case row.ReportedAt != nil && row.ExitCode != nil && *row.ExitCode == 0:
		status = PolicyRemediationRunStatusSucceeded
	case row.ReportedAt != nil:
		status = PolicyRemediationRunStatusFailed
	}
	return &PolicyRemediationRunSummary{
		ExecutionID: row.ExecutionID,
		Status:      status,
		Automatic:   row.Automatic,
		QueuedAt:    row.QueuedAt,
		CompletedAt: row.ReportedAt,
		ExitCode:    row.ExitCode,
	}
}

func remediationStatusSQL() string {
	return `CASE
		WHEN run.execution_id IS NULL THEN 'not_run'
		WHEN run.reported_at IS NOT NULL AND run.exit_code = 0 THEN 'succeeded'
		WHEN run.reported_at IS NOT NULL THEN 'failed'
		ELSE 'queued'
	END`
}

func remediationStatusOrderSQL(statusSQL string) string {
	return `CASE (` + statusSQL + `)
		WHEN 'failed' THEN 0
		WHEN 'queued' THEN 1
		WHEN 'succeeded' THEN 2
		ELSE 3
	END`
}
