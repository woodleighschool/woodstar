package policies

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

// Store persists policies and per-host membership state.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, in PolicyCreateMutation) (*Policy, error) {
	in.normalize()
	if err := in.Validate(); err != nil {
		return nil, err
	}
	write := newPolicyWrite(in.PolicyMutation)
	write.CreatedByUserID = in.CreatedByUserID

	var id int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO osquery_policies (
				name,
				description,
				resolution,
				query,
				remediation_script,
				automatic_remediation_enabled,
				created_by_user_id
			)
			VALUES (
				@name,
				@description,
				@resolution,
				@query,
				@remediation_script,
				@automatic_remediation_enabled,
				@created_by_user_id
			)
			RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
			return postgres.MutationError(err)
		}
		return replacePolicyTargets(ctx, tx, id, in.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, in PolicyMutation) (*Policy, error) {
	in.normalize()
	if err := in.Validate(); err != nil {
		return nil, err
	}
	write := newPolicyWrite(in)
	write.ID = id

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var queryChanged bool
		if err := tx.QueryRow(ctx, `
			WITH current AS (
				SELECT
					id,
					query,
					remediation_script,
					automatic_remediation_enabled,
					evaluation_revision,
					remediation_revision
				FROM osquery_policies
				WHERE id = @id
				FOR UPDATE
			)
			UPDATE osquery_policies c
			SET
				name = @name,
				description = @description,
				resolution = @resolution,
				query = @query,
				remediation_script = @remediation_script,
				automatic_remediation_enabled = @automatic_remediation_enabled,
				evaluation_revision = CASE
					WHEN current.query IS DISTINCT FROM @query
					THEN current.evaluation_revision + 1
					ELSE current.evaluation_revision
				END,
				remediation_revision = CASE
					WHEN current.remediation_script IS DISTINCT FROM @remediation_script
						OR current.automatic_remediation_enabled IS DISTINCT FROM @automatic_remediation_enabled
					THEN current.remediation_revision + 1
					ELSE current.remediation_revision
				END,
				updated_at = now()
			FROM current
			WHERE c.id = current.id
			RETURNING current.query IS DISTINCT FROM @query`,
			pgx.StructArgs(write),
		).Scan(&queryChanged); err != nil {
			return postgres.MutationError(err)
		}
		if err := replacePolicyTargets(ctx, tx, id, in.Targets); err != nil {
			return err
		}
		// Query edits invalidate every prior answer. Retargeting only removes
		// answers outside the newly completed assignment set.
		_, err := tx.Exec(ctx, `
			DELETE FROM osquery_policy_membership membership
			WHERE membership.policy_id = $1
			  AND (
				  $2
				  OR NOT EXISTS (
					  SELECT 1
					  FROM osquery_policy_assignments assignment
					  WHERE assignment.policy_id = membership.policy_id
					    AND assignment.host_id = membership.host_id
				  )
			  )`,
			id,
			queryChanged,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Policy, error) {
	if id <= 0 {
		return nil, fault.ErrNotFound
	}
	row, err := postgres.GetOne[policyRow](ctx, s.pool, policySelectSQL()+"\nWHERE c.id = $1", id)
	if err != nil {
		return nil, err
	}
	policy := policyFromRow(row)
	targets, err := s.loadPolicyTarget(ctx, policy.ID)
	if err != nil {
		return nil, err
	}
	policy.Targets = targets
	return policy, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM osquery_policies WHERE id = $1`, id)
	if err != nil {
		return postgres.DeleteConflict(err, "Policy is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return fault.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple policies. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.pool.Query(ctx, `DELETE FROM osquery_policies WHERE id = ANY($1::bigint[]) RETURNING id`, ids)
	if err != nil {
		return 0, err
	}
	deletedIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	return len(deletedIDs), nil
}

func (s *Store) List(ctx context.Context, params PolicyListParams) ([]Policy, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
	where, args := policyListWhere(params)
	listQuery := postgres.ListQuery{
		SelectSQL: policySelectSQL(),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: policyOrderKeys(),
		DefaultOrder: []postgres.OrderExpr{
			{SQL: "c.updated_at"},
			{SQL: "c.id"},
		},
		Params: params.ListParams,
	}
	rows, count, err := postgres.ListWithCount[policyRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}
	policies := policiesFromRows(rows)
	policyIDs := make([]int64, 0, len(policies))
	for _, policy := range policies {
		policyIDs = append(policyIDs, policy.ID)
	}
	targets, err := s.loadPolicyTargets(ctx, policyIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range policies {
		policies[i].Targets = targets[policies[i].ID]
	}
	return policies, count, nil
}

// IssueEvaluationsForHost records a monotonically increasing identity for each
// policy query returned to a host.
func (s *Store) IssueEvaluationsForHost(ctx context.Context, host *hosts.Host) ([]Evaluation, error) {
	rows, err := s.pool.Query(ctx, `
		WITH applicable AS (
			SELECT policy.id
			FROM osquery_policies policy
			JOIN osquery_policy_assignments assignment
				ON assignment.policy_id = policy.id
			   AND assignment.host_id = $1
		),
		issued AS (
			INSERT INTO osquery_policy_membership (
				policy_id,
				host_id,
				last_issued_sequence
			)
			SELECT applicable.id, $1, 1
			FROM applicable
			ON CONFLICT (policy_id, host_id) DO UPDATE SET
				last_issued_sequence =
					osquery_policy_membership.last_issued_sequence + 1
			RETURNING policy_id, last_issued_sequence
		)
		SELECT
			policy.id AS policy_id,
			policy.query,
			policy.evaluation_revision AS revision,
			issued.last_issued_sequence AS sequence
		FROM issued
		JOIN osquery_policies policy ON policy.id = issued.policy_id
		ORDER BY policy.id`, host.ID)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[policyEvaluationRow])
	if err != nil {
		return nil, err
	}
	evaluations := make([]Evaluation, len(records))
	for i, row := range records {
		evaluations[i] = Evaluation(row)
	}
	return evaluations, nil
}

// RecordEvaluation records an ordered policy result when its query revision and
// host assignment are still current.
func (s *Store) RecordEvaluation(
	ctx context.Context,
	policyID int64,
	queryHash string,
	revision int64,
	sequence int64,
	hostID int64,
	result EvaluationResult,
) error {
	if revision <= 0 || sequence <= 0 {
		return nil
	}
	switch result.Status {
	case PolicyStatusPass, PolicyStatusFail:
		result.Error = ""
	case PolicyStatusError:
		result.Error = truncateRunes(result.Error, maxPolicyErrorRunes)
	case PolicyStatusPending:
		return fault.ErrInvalidInput
	default:
		return fault.ErrInvalidInput
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var lastConclusivePasses *bool
		var remediationScript string
		var remediationRevision int64
		var automaticRemediationEnabled, orbitScriptExecutionAvailable bool
		err := tx.QueryRow(ctx, `
			SELECT
				membership.last_conclusive_passes,
				policy_row.remediation_script,
				policy_row.remediation_revision,
				policy_row.automatic_remediation_enabled,
				COALESCE(host.orbit_scripts_enabled, false)
					AND host.orbit_node_key <> ''
			FROM osquery_policy_membership membership
			JOIN osquery_policies policy_row ON policy_row.id = membership.policy_id
			JOIN osquery_policy_assignments assignment
				ON assignment.policy_id = policy_row.id
			   AND assignment.host_id = membership.host_id
			JOIN hosts host ON host.id = membership.host_id
			WHERE policy_row.id = $1
			  AND encode(sha256(convert_to(policy_row.query, 'UTF8')), 'hex') = $2
			  AND policy_row.evaluation_revision = $3
			  AND membership.host_id = $4
			  AND $5 = membership.last_issued_sequence
			  AND $5 > membership.last_completed_sequence
			FOR UPDATE OF policy_row, membership`,
			policyID,
			queryHash,
			revision,
			hostID,
			sequence,
		).Scan(
			&lastConclusivePasses,
			&remediationScript,
			&remediationRevision,
			&automaticRemediationEnabled,
			&orbitScriptExecutionAvailable,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		newlyFailing := result.Status == PolicyStatusFail &&
			(lastConclusivePasses == nil || *lastConclusivePasses)
		_, err = tx.Exec(ctx, `
			UPDATE osquery_policy_membership
			SET
				status = $3::osquery_policy_status,
				error = $4,
				last_conclusive_passes = CASE
					WHEN $3 = 'pass' THEN true
					WHEN $3 = 'fail' THEN false
					ELSE last_conclusive_passes
				END,
				remediation_failure_sequence = CASE
					WHEN $6 THEN $5
					ELSE remediation_failure_sequence
				END,
				last_completed_sequence = $5,
				updated_at = now()
			WHERE policy_id = $1 AND host_id = $2`,
			policyID,
			hostID,
			result.Status,
			result.Error,
			sequence,
			newlyFailing,
		)
		if err != nil {
			return err
		}
		if !newlyFailing || !automaticRemediationEnabled || !orbitScriptExecutionAvailable {
			return nil
		}
		_, err = s.enqueueRemediationTx(
			ctx,
			tx,
			policyID,
			hostID,
			remediationRevision,
			sequence,
			remediationScript,
			true,
		)
		return err
	})
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[len(runes)-limit:])
}

func (s *Store) PolicyResults(
	ctx context.Context,
	policyID int64,
	params PolicyResultListParams,
) ([]PolicyHostStatus, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
	params.Statuses = listing.NormalizeValues(params.Statuses)
	params.RemediationStatuses = listing.NormalizeValues(params.RemediationStatuses)
	if err := validatePolicyStatusFilters(params.Statuses); err != nil {
		return nil, 0, err
	}
	if err := validatePolicyRemediationStatusFilters(params.RemediationStatuses); err != nil {
		return nil, 0, err
	}
	where, args, remediationStatus := policyResultListWhere(
		params,
		"c.id",
		policyID,
		"h.display_name",
	)
	listQuery := postgres.ListQuery{
		SelectSQL: policyResultSelectSQL(remediationStatus),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]postgres.OrderExpr{
			"host_name":   {SQL: "lower(h.display_name)"},
			"status":      {SQL: policyStatusOrderSQL()},
			"remediation": {SQL: remediationStatusOrderSQL("remediation_status.value")},
			"updated_at":  {SQL: "m.updated_at", NullOrder: postgres.NullsLast},
		},
		DefaultOrder: []postgres.OrderExpr{
			{SQL: policyStatusOrderSQL()},
			{SQL: "lower(h.display_name)"},
			{SQL: "h.id"},
		},
		Params: params.ListParams,
	}
	records, count, err := postgres.ListWithCount[policyHostStatusRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		var exists bool
		if err := s.pool.QueryRow(
			ctx,
			`SELECT EXISTS (SELECT 1 FROM osquery_policies WHERE id = $1)`,
			policyID,
		).Scan(&exists); err != nil {
			return nil, 0, err
		}
		if !exists {
			return nil, 0, fault.ErrNotFound
		}
	}
	return policyHostStatusesFromRows(records), count, nil
}

func (s *Store) HostPolicies(
	ctx context.Context,
	host *hosts.Host,
	params PolicyResultListParams,
) ([]PolicyHostStatus, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
	params.Statuses = listing.NormalizeValues(params.Statuses)
	params.RemediationStatuses = listing.NormalizeValues(params.RemediationStatuses)
	if err := validatePolicyStatusFilters(params.Statuses); err != nil {
		return nil, 0, err
	}
	if err := validatePolicyRemediationStatusFilters(params.RemediationStatuses); err != nil {
		return nil, 0, err
	}
	where, args, remediationStatus := policyResultListWhere(
		params,
		"h.id",
		host.ID,
		"c.name",
	)
	listQuery := postgres.ListQuery{
		SelectSQL: policyResultSelectSQL(remediationStatus),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]postgres.OrderExpr{
			"policy_name": {SQL: "lower(c.name)"},
			"status":      {SQL: policyStatusOrderSQL()},
			"remediation": {SQL: remediationStatusOrderSQL("remediation_status.value")},
			"updated_at":  {SQL: "m.updated_at", NullOrder: postgres.NullsLast},
		},
		DefaultOrder: []postgres.OrderExpr{
			{SQL: policyStatusOrderSQL()},
			{SQL: "lower(c.name)"},
			{SQL: "c.id"},
		},
		Params: params.ListParams,
	}
	records, count, err := postgres.ListWithCount[policyHostStatusRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}
	return policyHostStatusesFromRows(records), count, nil
}

func policyResultListWhere(
	params PolicyResultListParams,
	scopeSQL string,
	scopeID int64,
	nameSQL string,
) (string, []any, string) {
	var where postgres.WhereBuilder
	remediationStatusExpr := remediationStatusSQL()
	where.Add(scopeSQL + " = " + where.Arg(scopeID))
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add(nameSQL + " ILIKE " + search)
	}
	if len(params.Statuses) > 0 {
		statuses := where.Arg(params.Statuses)
		where.Add(`COALESCE(m.status, 'pending')::text = ANY(` + statuses + `::text[])`)
	}
	if len(params.RemediationStatuses) > 0 {
		statuses := where.Arg(params.RemediationStatuses)
		where.Add("remediation_status.value = ANY(" + statuses + "::text[])")
	}
	whereSQL, args := where.Build()
	return whereSQL, args, remediationStatusExpr
}

func policyResultSelectSQL(remediationStatusExpr string) string {
	return `
	SELECT
		c.id AS policy_id,
		c.name AS policy_name,
		h.id AS host_id,
		h.display_name AS host_name,
		COALESCE(m.status, 'pending')::text AS status,
		COALESCE(m.error, '') AS error,
		CASE WHEN m.status = 'pending' THEN NULL ELSE m.updated_at END AS updated_at,
		run.execution_id,
		run.automatic AS remediation_automatic,
		run.queued_at AS remediation_queued_at,
		run.reported_at AS remediation_reported_at,
		run.exit_code AS remediation_exit_code,
		remediation_status.value AS remediation_status
	FROM osquery_policies c
	JOIN osquery_policy_assignments assignment ON assignment.policy_id = c.id
	JOIN hosts h ON h.id = assignment.host_id
	LEFT JOIN osquery_policy_membership m
		ON m.host_id = h.id
	   AND m.policy_id = c.id
	LEFT JOIN osquery_policy_remediation_runs run
		ON run.host_id = h.id
	   AND run.policy_id = c.id
	   AND m.status = 'fail'
	   AND run.remediation_revision = c.remediation_revision
	   AND run.failure_sequence = m.remediation_failure_sequence
	   AND run.script_contents = c.remediation_script
	LEFT JOIN LATERAL (
		SELECT (` + remediationStatusExpr + `) AS value
	) remediation_status ON true`
}

func policyStatusOrderSQL() string {
	return `CASE
		WHEN m.status = 'fail' THEN 0
		WHEN m.status = 'error' THEN 1
		WHEN m.status IS NULL OR m.status = 'pending' THEN 2
		ELSE 3
	END`
}

func policyHostStatusesFromRows(rows []policyHostStatusRow) []PolicyHostStatus {
	statuses := make([]PolicyHostStatus, 0, len(rows))
	for _, row := range rows {
		statuses = append(statuses, PolicyHostStatus{
			PolicyID:    row.PolicyID,
			PolicyName:  row.PolicyName,
			HostID:      row.HostID,
			HostName:    row.HostName,
			Status:      row.Status,
			Error:       row.Error,
			UpdatedAt:   row.UpdatedAt,
			Remediation: remediationRunSummaryFromRow(row),
		})
	}
	return statuses
}

func validatePolicyRemediationStatusFilters(statuses []PolicyRemediationStatusFilter) error {
	for _, status := range statuses {
		switch status {
		case PolicyRemediationFilterNotRun,
			PolicyRemediationFilterQueued,
			PolicyRemediationFilterSucceeded,
			PolicyRemediationFilterFailed:
		default:
			return fault.ErrInvalidInput
		}
	}
	return nil
}

func validatePolicyStatusFilters(statuses []PolicyStatus) error {
	for _, status := range statuses {
		switch status {
		case PolicyStatusPass, PolicyStatusFail, PolicyStatusPending, PolicyStatusError:
		default:
			return fault.ErrInvalidInput
		}
	}
	return nil
}

func policyListWhere(params PolicyListParams) (string, []any) {
	var where postgres.WhereBuilder
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add("(c.name ILIKE " + search + " OR c.description ILIKE " + search +
			" OR c.resolution ILIKE " + search + " OR c.query ILIKE " + search + ")")
	}
	return where.Build()
}

func policyOrderKeys() map[string]postgres.OrderExpr {
	return map[string]postgres.OrderExpr{
		"name":               {SQL: "c.name"},
		"passing_host_count": {SQL: "result_counts.passing_host_count"},
		"failing_host_count": {SQL: "result_counts.failing_host_count"},
		"error_host_count":   {SQL: "result_counts.error_host_count"},
		"pending_host_count": {SQL: "result_counts.pending_host_count"},
		"remediation": {
			SQL: `CASE
				WHEN NULLIF(btrim(c.remediation_script), '') IS NULL THEN 0
				WHEN c.automatic_remediation_enabled THEN 2
				ELSE 1
			END`,
		},
		"created_at": {SQL: "c.created_at"},
		"updated_at": {SQL: "c.updated_at"},
	}
}

type policyRow struct {
	ID                          int64     `db:"id"`
	Name                        string    `db:"name"`
	Description                 string    `db:"description"`
	Resolution                  string    `db:"resolution"`
	Query                       string    `db:"query"`
	RemediationConfigured       bool      `db:"remediation_configured"`
	AutomaticRemediationEnabled bool      `db:"automatic_remediation_enabled"`
	PassingHostCount            int32     `db:"passing_host_count"`
	FailingHostCount            int32     `db:"failing_host_count"`
	ErrorHostCount              int32     `db:"error_host_count"`
	PendingHostCount            int32     `db:"pending_host_count"`
	CreatedByUserID             *int64    `db:"created_by_user_id"`
	CreatedByName               string    `db:"created_by_name"`
	CreatedByEmail              string    `db:"created_by_email"`
	CreatedAt                   time.Time `db:"created_at"`
	UpdatedAt                   time.Time `db:"updated_at"`
}

type policyEvaluationRow struct {
	PolicyID int64  `db:"policy_id"`
	Query    string `db:"query"`
	Revision int64  `db:"revision"`
	Sequence int64  `db:"sequence"`
}

type policyHostStatusRow struct {
	PolicyID              int64                         `db:"policy_id"`
	PolicyName            string                        `db:"policy_name"`
	HostID                int64                         `db:"host_id"`
	HostName              string                        `db:"host_name"`
	Status                PolicyStatus                  `db:"status"`
	Error                 string                        `db:"error"`
	UpdatedAt             *time.Time                    `db:"updated_at"`
	ExecutionID           *string                       `db:"execution_id"`
	RemediationAutomatic  *bool                         `db:"remediation_automatic"`
	RemediationQueuedAt   *time.Time                    `db:"remediation_queued_at"`
	RemediationReportedAt *time.Time                    `db:"remediation_reported_at"`
	RemediationExitCode   *int                          `db:"remediation_exit_code"`
	RemediationStatus     PolicyRemediationStatusFilter `db:"remediation_status"`
}

func policyFromRow(row policyRow) *Policy {
	return &Policy{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Resolution:  row.Resolution,
		Query:       row.Query,
		Remediation: PolicyRemediationSummary{
			Configured: row.RemediationConfigured,
			Automatic:  row.AutomaticRemediationEnabled,
		},
		PassingHostCount: row.PassingHostCount,
		FailingHostCount: row.FailingHostCount,
		ErrorHostCount:   row.ErrorHostCount,
		PendingHostCount: row.PendingHostCount,
		CreatedBy:        userSummary(row.CreatedByUserID, row.CreatedByName, row.CreatedByEmail),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func policiesFromRows(rows []policyRow) []Policy {
	policies := make([]Policy, len(rows))
	for i, row := range rows {
		policies[i] = *policyFromRow(row)
	}
	return policies
}

type policyWrite struct {
	ID                          int64  `db:"id"`
	Name                        string `db:"name"`
	Description                 string `db:"description"`
	Resolution                  string `db:"resolution"`
	Query                       string `db:"query"`
	RemediationScript           string `db:"remediation_script"`
	AutomaticRemediationEnabled bool   `db:"automatic_remediation_enabled"`
	CreatedByUserID             *int64 `db:"created_by_user_id"`
}

func newPolicyWrite(in PolicyMutation) policyWrite {
	write := policyWrite{
		Name:        in.Name,
		Description: in.Description,
		Resolution:  in.Resolution,
		Query:       in.Query,
	}
	if in.Remediation != nil {
		write.RemediationScript = in.Remediation.Script
		write.AutomaticRemediationEnabled = in.Remediation.Automatic
	}
	return write
}

func policySelectSQL() string {
	return `
SELECT
	c.id,
	c.name,
	c.description,
	c.resolution,
	c.query,
	NULLIF(btrim(c.remediation_script), '') IS NOT NULL AS remediation_configured,
	c.automatic_remediation_enabled,
	result_counts.passing_host_count,
	result_counts.failing_host_count,
	result_counts.error_host_count,
	result_counts.pending_host_count,
	c.created_by_user_id,
	COALESCE(creator.name, '') AS created_by_name,
	COALESCE(creator.email, '') AS created_by_email,
	c.created_at,
	c.updated_at
FROM osquery_policies c
LEFT JOIN users creator ON creator.id = c.created_by_user_id
LEFT JOIN LATERAL (
	SELECT
		COUNT(*) FILTER (WHERE membership.status = 'pass')::integer AS passing_host_count,
		COUNT(*) FILTER (WHERE membership.status = 'fail')::integer AS failing_host_count,
		COUNT(*) FILTER (WHERE membership.status = 'error')::integer AS error_host_count,
		COUNT(*) FILTER (
			WHERE membership.status IS NULL OR membership.status = 'pending'
		)::integer AS pending_host_count
	FROM osquery_policy_assignments assignment
	LEFT JOIN osquery_policy_membership membership
		ON membership.policy_id = assignment.policy_id
	   AND membership.host_id = assignment.host_id
	WHERE assignment.policy_id = c.id
) result_counts ON true`
}

func userSummary(id *int64, name string, email string) *directory.UserSummary {
	if id == nil {
		return nil
	}
	return &directory.UserSummary{ID: *id, Name: name, Email: email}
}
