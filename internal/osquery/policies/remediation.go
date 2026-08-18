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
	remediationResponseGrace    = time.Minute
	remediationExecutionIDBytes = 18
	maxRemediationOutputRunes   = 10_000
	maxPolicyErrorRunes         = 4_096
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

// PolicyRemediationStatusFilter is a latest-run state accepted by result listings.
type PolicyRemediationStatusFilter string

const (
	PolicyRemediationFilterNotRun     PolicyRemediationStatusFilter = "not_run"
	PolicyRemediationFilterQueued     PolicyRemediationStatusFilter = "queued"
	PolicyRemediationFilterInProgress PolicyRemediationStatusFilter = "in_progress"
	PolicyRemediationFilterSucceeded  PolicyRemediationStatusFilter = "succeeded"
	PolicyRemediationFilterFailed     PolicyRemediationStatusFilter = "failed"
	PolicyRemediationFilterNoResponse PolicyRemediationStatusFilter = "no_response"
	PolicyRemediationFilterCancelled  PolicyRemediationStatusFilter = "cancelled"
)

var policyRemediationStatusFilterValues = []PolicyRemediationStatusFilter{
	PolicyRemediationFilterNotRun,
	PolicyRemediationFilterQueued,
	PolicyRemediationFilterInProgress,
	PolicyRemediationFilterSucceeded,
	PolicyRemediationFilterFailed,
	PolicyRemediationFilterNoResponse,
	PolicyRemediationFilterCancelled,
}

func (PolicyRemediationStatusFilter) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(policyRemediationStatusFilterValues...)
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

// ClaimedRemediation is the immutable script returned to Orbit.
type ClaimedRemediation struct {
	HostID         int64
	ExecutionID    string
	ScriptContents string
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
		var revision int64
		err := tx.QueryRow(ctx, `
			SELECT remediation_script, evaluation_revision
			FROM osquery_policies
			WHERE id = $1
			FOR UPDATE`, policyID).Scan(&script, &revision)
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
			err := row.Scan(&candidate.HostID, &candidate.Eligible)
			return candidate, err
		})
		if err != nil {
			return err
		}

		queueHostIDs := make([]int64, 0, len(candidates))
		executionIDs := make([]string, 0, len(candidates))
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
		}
		queued, err := s.enqueueRemediationsTx(
			ctx,
			tx,
			policyID,
			queueHostIDs,
			executionIDs,
			revision,
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
	HostID   int64
	Eligible bool
}

func (s *Store) enqueueRemediationTx(
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
	rows, err := s.enqueueRemediationsTx(
		ctx,
		tx,
		policyID,
		[]int64{hostID},
		[]string{executionID},
		revision,
		script,
		automatic,
	)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return remediationRunSummary(rows[0], time.Now(), s.remediationExecutionTimeout), nil
}

func (s *Store) enqueueRemediationsTx(
	ctx context.Context,
	tx pgx.Tx,
	policyID int64,
	hostIDs []int64,
	executionIDs []string,
	revision int64,
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
			evaluation_revision,
			automatic
		)
		SELECT
			$1,
			batch.host_id,
			batch.execution_id,
			$4,
			$5,
			$6
		FROM unnest($2::bigint[], $3::text[]) AS batch(host_id, execution_id)
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
			exit_code = NULL
		WHERE osquery_policy_remediation_runs.reported_at IS NOT NULL
		   OR osquery_policy_remediation_runs.cancelled_at IS NOT NULL
		   OR (
			   osquery_policy_remediation_runs.claimed_at IS NOT NULL
			   AND osquery_policy_remediation_runs.reported_at IS NULL
			   AND osquery_policy_remediation_runs.claimed_at
			       + make_interval(secs => $7) < now()
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
			exit_code`,
		policyID,
		hostIDs,
		executionIDs,
		script,
		revision,
		automatic,
		int(remediationNoResponseAfter(s.remediationExecutionTimeout).Seconds()),
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
			&run.ClaimedAt,
			&run.ReportedAt,
			&run.CancelledAt,
			&run.Output,
			&run.RuntimeSeconds,
			&run.ExitCode,
		)
		return run, err
	})
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
		RETURNING host_id, execution_id, script_contents`,
		hostID,
		executionID,
	).Scan(
		&claimed.HostID,
		&claimed.ExecutionID,
		&claimed.ScriptContents,
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
	summary := remediationRunSummary(row, time.Now(), s.remediationExecutionTimeout)
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
			exit_code
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
		StartedAt:   row.RemediationClaimedAt,
		CompletedAt: row.RemediationReportedAt,
		ExitCode:    row.RemediationExitCode,
	}
}

func remediationRunSummary(
	row remediationRunRow,
	now time.Time,
	remediationExecutionTimeout time.Duration,
) *PolicyRemediationRunSummary {
	status := PolicyRemediationRunStatusQueued
	switch {
	case row.CancelledAt != nil:
		status = PolicyRemediationRunStatusCancelled
	case row.ReportedAt != nil && row.ExitCode != nil && *row.ExitCode == 0:
		status = PolicyRemediationRunStatusSucceeded
	case row.ReportedAt != nil:
		status = PolicyRemediationRunStatusFailed
	case row.ClaimedAt != nil && now.After(row.ClaimedAt.Add(
		remediationNoResponseAfter(remediationExecutionTimeout),
	)):
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

func remediationNoResponseAfter(remediationExecutionTimeout time.Duration) time.Duration {
	return remediationExecutionTimeout + remediationResponseGrace
}

func remediationStatusSQL(noResponseSecondsSQL string) string {
	return `CASE
		WHEN run.execution_id IS NULL THEN 'not_run'
		WHEN run.cancelled_at IS NOT NULL THEN 'cancelled'
		WHEN run.reported_at IS NOT NULL AND run.exit_code = 0 THEN 'succeeded'
		WHEN run.reported_at IS NOT NULL THEN 'failed'
		WHEN run.claimed_at IS NOT NULL
			AND run.claimed_at + make_interval(secs => ` + noResponseSecondsSQL + `) < now() THEN 'no_response'
		WHEN run.claimed_at IS NOT NULL THEN 'in_progress'
		ELSE 'queued'
	END`
}

func remediationStatusOrderSQL(statusSQL string) string {
	return `CASE (` + statusSQL + `)
		WHEN 'failed' THEN 0
		WHEN 'no_response' THEN 1
		WHEN 'in_progress' THEN 2
		WHEN 'queued' THEN 3
		WHEN 'succeeded' THEN 4
		WHEN 'cancelled' THEN 5
		ELSE 6
	END`
}
