package checks

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

// Store persists checks and per-host membership state.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, in CheckCreateMutation) (*Check, error) {
	in.normalize()
	if err := in.Validate(); err != nil {
		return nil, err
	}
	write := newCheckWrite(in.CheckMutation)
	write.CreatedByUserID = in.CreatedByUserID

	var id int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO osquery_checks (name, description, query, created_by_user_id)
			VALUES (@name, @description, @query, @created_by_user_id)
			RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
			return postgres.MutationError(err)
		}
		return replaceCheckTargets(ctx, tx, id, in.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, in CheckMutation) (*Check, error) {
	in.normalize()
	if err := in.Validate(); err != nil {
		return nil, err
	}
	write := newCheckWrite(in)
	write.ID = id

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var queryChanged bool
		if err := tx.QueryRow(ctx, `
			WITH current AS (
				SELECT id, query
				FROM osquery_checks
				WHERE id = @id
				FOR UPDATE
			)
			UPDATE osquery_checks c
			SET
				name = @name,
				description = @description,
				query = @query,
				updated_at = now()
			FROM current
			WHERE c.id = current.id
			RETURNING current.query IS DISTINCT FROM @query`,
			pgx.StructArgs(write),
		).Scan(&queryChanged); err != nil {
			return postgres.MutationError(err)
		}
		if err := replaceCheckTargets(ctx, tx, id, in.Targets); err != nil {
			return err
		}
		// Query edits invalidate every prior answer. Retargeting only removes
		// answers outside the newly completed assignment set.
		_, err := tx.Exec(ctx, `
			DELETE FROM osquery_check_membership membership
			WHERE membership.check_id = $1
			  AND (
				  $2
				  OR NOT EXISTS (
					  SELECT 1
					  FROM osquery_check_assignments assignment
					  WHERE assignment.check_id = membership.check_id
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

func (s *Store) GetByID(ctx context.Context, id int64) (*Check, error) {
	if id <= 0 {
		return nil, fault.ErrNotFound
	}
	row, err := postgres.GetOne[checkRow](ctx, s.pool, checkSelectSQL()+"\nWHERE c.id = $1", id)
	if err != nil {
		return nil, err
	}
	check := checkFromRow(row)
	targets, err := s.loadCheckTarget(ctx, check.ID)
	if err != nil {
		return nil, err
	}
	check.Targets = targets
	counts, err := s.loadCheckCounts(ctx, []int64{check.ID})
	if err != nil {
		return nil, err
	}
	check.PassingHostCount = counts[check.ID].Passing
	check.FailingHostCount = counts[check.ID].Failing
	return check, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM osquery_checks WHERE id = $1`, id)
	if err != nil {
		return postgres.DeleteConflict(err, "Check is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return fault.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple checks. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.pool.Query(ctx, `DELETE FROM osquery_checks WHERE id = ANY($1::bigint[]) RETURNING id`, ids)
	if err != nil {
		return 0, err
	}
	deletedIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	return len(deletedIDs), nil
}

func (s *Store) List(ctx context.Context, params CheckListParams) ([]Check, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
	where, args := checkListWhere(params)
	listQuery := postgres.ListQuery{
		SelectSQL: checkSelectSQL(),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: checkOrderKeys(),
		DefaultOrder: []postgres.OrderExpr{
			{SQL: "c.updated_at"},
			{SQL: "c.id"},
		},
		Params: params.ListParams,
	}
	rows, count, err := postgres.ListWithCount[checkRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}
	checks := checksFromRows(rows)
	checkIDs := make([]int64, 0, len(checks))
	for _, check := range checks {
		checkIDs = append(checkIDs, check.ID)
	}
	targets, err := s.loadCheckTargets(ctx, checkIDs)
	if err != nil {
		return nil, 0, err
	}
	counts, err := s.loadCheckCounts(ctx, checkIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range checks {
		checks[i].Targets = targets[checks[i].ID]
		checks[i].PassingHostCount = counts[checks[i].ID].Passing
		checks[i].FailingHostCount = counts[checks[i].ID].Failing
	}
	return checks, count, nil
}

func (s *Store) ApplicableForHost(ctx context.Context, host *hosts.Host) ([]Check, error) {
	rows, err := s.pool.Query(ctx, `
		WITH host_row AS (
			SELECT id
			FROM hosts h
			WHERE h.id = $1
		)
		SELECT
			c.id,
			c.name,
			c.description,
			c.query,
			c.created_by_user_id,
			c.created_at,
			c.updated_at
		FROM osquery_checks c
		JOIN host_row h ON true
		JOIN osquery_check_assignments assignment
			ON assignment.check_id = c.id
		   AND assignment.host_id = h.id
		ORDER BY c.id`, host.ID)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[checkRow])
	if err != nil {
		return nil, err
	}
	return checksFromRows(records), nil
}

// UpsertMembership records a check result when its query hash and host assignment
// are still current. A nil passes value means the query did not run.
func (s *Store) UpsertMembership(
	ctx context.Context,
	checkID int64,
	queryHash string,
	hostID int64,
	passes *bool,
) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var accepted bool
		err := tx.QueryRow(ctx, `
			SELECT true
			FROM osquery_checks check_row
			JOIN osquery_check_assignments assignment
				ON assignment.check_id = check_row.id
			   AND assignment.host_id = $3
			WHERE check_row.id = $1
			  AND encode(sha256(convert_to(check_row.query, 'UTF8')), 'hex') = $2
			FOR UPDATE OF check_row`,
			checkID,
			queryHash,
			hostID,
		).Scan(&accepted)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO osquery_check_membership (check_id, host_id, passes, updated_at)
			VALUES ($1, $2, $3, now())
			ON CONFLICT (check_id, host_id) DO UPDATE SET
				passes = EXCLUDED.passes,
				updated_at = now()`,
			checkID,
			hostID,
			passes,
		)
		return err
	})
}

func (s *Store) CheckResults(
	ctx context.Context,
	checkID int64,
	params CheckResultListParams,
) ([]CheckHostStatus, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
	params.Statuses = listing.NormalizeValues(params.Statuses)
	if err := validateCheckStatusFilters(params.Statuses); err != nil {
		return nil, 0, err
	}
	where, args := checkResultListWhere(params, "c.id", checkID, "h.display_name")
	listQuery := postgres.ListQuery{
		SelectSQL: `
		SELECT
			c.id AS check_id,
			c.name AS check_name,
			h.id AS host_id,
			h.display_name AS host_name,
			m.passes,
			m.updated_at
		FROM osquery_checks c
		JOIN osquery_check_assignments assignment ON assignment.check_id = c.id
		JOIN hosts h ON h.id = assignment.host_id
		LEFT JOIN osquery_check_membership m
			ON m.host_id = h.id
		   AND m.check_id = c.id`,
		WhereSQL: where,
		Args:     args,
		OrderKeys: map[string]postgres.OrderExpr{
			"host_name":  {SQL: "lower(h.display_name)"},
			"status":     {SQL: checkStatusOrderSQL()},
			"updated_at": {SQL: "m.updated_at", NullOrder: postgres.NullsLast},
		},
		DefaultOrder: []postgres.OrderExpr{
			{SQL: checkStatusOrderSQL()},
			{SQL: "lower(h.display_name)"},
			{SQL: "h.id"},
		},
		Params: params.ListParams,
	}
	records, count, err := postgres.ListWithCount[checkHostStatusRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		var exists bool
		if err := s.pool.QueryRow(
			ctx,
			`SELECT EXISTS (SELECT 1 FROM osquery_checks WHERE id = $1)`,
			checkID,
		).Scan(&exists); err != nil {
			return nil, 0, err
		}
		if !exists {
			return nil, 0, fault.ErrNotFound
		}
	}
	return checkHostStatusesFromRows(records), count, nil
}

func (s *Store) HostChecks(
	ctx context.Context,
	host *hosts.Host,
	params CheckResultListParams,
) ([]CheckHostStatus, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
	params.Statuses = listing.NormalizeValues(params.Statuses)
	if err := validateCheckStatusFilters(params.Statuses); err != nil {
		return nil, 0, err
	}
	where, args := checkResultListWhere(params, "h.id", host.ID, "c.name")
	listQuery := postgres.ListQuery{
		SelectSQL: `
		SELECT
			c.id AS check_id,
			c.name AS check_name,
			h.id AS host_id,
			h.display_name AS host_name,
			m.passes,
			m.updated_at
		FROM osquery_checks c
		JOIN osquery_check_assignments assignment ON assignment.check_id = c.id
		JOIN hosts h ON h.id = assignment.host_id
		LEFT JOIN osquery_check_membership m
			ON m.host_id = h.id
		   AND m.check_id = c.id`,
		WhereSQL: where,
		Args:     args,
		OrderKeys: map[string]postgres.OrderExpr{
			"check_name": {SQL: "lower(c.name)"},
			"status":     {SQL: checkStatusOrderSQL()},
			"updated_at": {SQL: "m.updated_at", NullOrder: postgres.NullsLast},
		},
		DefaultOrder: []postgres.OrderExpr{
			{SQL: checkStatusOrderSQL()},
			{SQL: "lower(c.name)"},
			{SQL: "c.id"},
		},
		Params: params.ListParams,
	}
	records, count, err := postgres.ListWithCount[checkHostStatusRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}
	return checkHostStatusesFromRows(records), count, nil
}

func checkResultListWhere(
	params CheckResultListParams,
	scopeSQL string,
	scopeID int64,
	nameSQL string,
) (string, []any) {
	var where postgres.WhereBuilder
	where.Add(scopeSQL + " = " + where.Arg(scopeID))
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add(nameSQL + " ILIKE " + search)
	}
	if len(params.Statuses) > 0 {
		statuses := where.Arg(params.Statuses)
		where.Add(`(
			('pass' = ANY(` + statuses + `::text[]) AND m.passes IS TRUE)
			OR ('fail' = ANY(` + statuses + `::text[]) AND m.passes IS FALSE)
			OR ('pending' = ANY(` + statuses + `::text[]) AND m.passes IS NULL)
		)`)
	}
	return where.Build()
}

func checkStatusOrderSQL() string {
	return `CASE
		WHEN m.passes IS FALSE THEN 0
		WHEN m.passes IS NULL THEN 1
		ELSE 2
	END`
}

func checkHostStatusesFromRows(rows []checkHostStatusRow) []CheckHostStatus {
	statuses := make([]CheckHostStatus, 0, len(rows))
	for _, row := range rows {
		statuses = append(statuses, CheckHostStatus{
			CheckID:   row.CheckID,
			CheckName: row.CheckName,
			HostID:    row.HostID,
			HostName:  row.HostName,
			Status:    checkStatusFromPasses(row.Passes),
			UpdatedAt: row.UpdatedAt,
		})
	}
	return statuses
}

func checkStatusFromPasses(passes *bool) CheckStatus {
	if passes == nil {
		return CheckStatusPending
	}
	if *passes {
		return CheckStatusPass
	}
	return CheckStatusFail
}

func validateCheckStatusFilters(statuses []CheckStatus) error {
	for _, status := range statuses {
		switch status {
		case CheckStatusPass, CheckStatusFail, CheckStatusPending:
		default:
			return fault.ErrInvalidInput
		}
	}
	return nil
}

type checkCounts struct {
	Passing int32
	Failing int32
}

func (s *Store) loadCheckCounts(ctx context.Context, checkIDs []int64) (map[int64]checkCounts, error) {
	if len(checkIDs) == 0 {
		return map[int64]checkCounts{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT
			membership.check_id,
			COUNT(*) FILTER (WHERE membership.passes IS TRUE)::integer AS passing_host_count,
			COUNT(*) FILTER (WHERE membership.passes IS FALSE)::integer AS failing_host_count
		FROM osquery_check_membership membership
		JOIN osquery_check_assignments assignment
			ON assignment.check_id = membership.check_id
		   AND assignment.host_id = membership.host_id
		WHERE membership.check_id = ANY($1::bigint[])
		GROUP BY membership.check_id`, checkIDs)
	if err != nil {
		return nil, err
	}
	type countRow struct {
		CheckID          int64 `db:"check_id"`
		PassingHostCount int32 `db:"passing_host_count"`
		FailingHostCount int32 `db:"failing_host_count"`
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[countRow])
	if err != nil {
		return nil, err
	}
	counts := make(map[int64]checkCounts, len(checkIDs))
	for _, r := range records {
		counts[r.CheckID] = checkCounts{
			Passing: r.PassingHostCount,
			Failing: r.FailingHostCount,
		}
	}
	return counts, nil
}

func checkListWhere(params CheckListParams) (string, []any) {
	var where postgres.WhereBuilder
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add("(c.name ILIKE " + search + " OR c.description ILIKE " + search + " OR c.query ILIKE " + search + ")")
	}
	return where.Build()
}

func checkOrderKeys() map[string]postgres.OrderExpr {
	return map[string]postgres.OrderExpr{
		"name":       {SQL: "c.name"},
		"created_at": {SQL: "c.created_at"},
		"updated_at": {SQL: "c.updated_at"},
	}
}

type checkRow struct {
	ID              int64     `db:"id"`
	Name            string    `db:"name"`
	Description     string    `db:"description"`
	Query           string    `db:"query"`
	CreatedByUserID *int64    `db:"created_by_user_id"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type checkHostStatusRow struct {
	CheckID   int64      `db:"check_id"`
	CheckName string     `db:"check_name"`
	HostID    int64      `db:"host_id"`
	HostName  string     `db:"host_name"`
	Passes    *bool      `db:"passes"`
	UpdatedAt *time.Time `db:"updated_at"`
}

func checkFromRow(row checkRow) *Check {
	return &Check{
		ID:              row.ID,
		Name:            row.Name,
		Description:     row.Description,
		Query:           row.Query,
		CreatedByUserID: row.CreatedByUserID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func checksFromRows(rows []checkRow) []Check {
	checks := make([]Check, len(rows))
	for i, row := range rows {
		checks[i] = *checkFromRow(row)
	}
	return checks
}

type checkWrite struct {
	ID              int64  `db:"id"`
	Name            string `db:"name"`
	Description     string `db:"description"`
	Query           string `db:"query"`
	CreatedByUserID *int64 `db:"created_by_user_id"`
}

func newCheckWrite(in CheckMutation) checkWrite {
	return checkWrite{
		Name:        in.Name,
		Description: in.Description,
		Query:       in.Query,
	}
}

func checkSelectSQL() string {
	return `
SELECT
	c.id,
	c.name,
	c.description,
	c.query,
	c.created_by_user_id,
	c.created_at,
	c.updated_at
FROM osquery_checks c`
}
