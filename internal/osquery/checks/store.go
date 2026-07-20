// Package checks persists query-backed pass/fail policies and per-host results.
package checks

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// Store persists checks and per-host membership state.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, in CheckCreateMutation) (*Check, error) {
	in.normalize()
	if err := in.Validate(); err != nil {
		return nil, err
	}
	write := newCheckWrite(in.CheckMutation)
	write.CreatedByUserID = in.CreatedByUserID

	var id int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO checks (name, description, query, created_by_user_id)
			VALUES (@name, @description, @query, @created_by_user_id)
			RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
			return dbutil.MutationError(err)
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

	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var updatedID int64
		if err := tx.QueryRow(ctx, `
			UPDATE checks
			SET
				name = @name,
				description = @description,
				query = @query,
				updated_at = now()
			WHERE id = @id
			RETURNING id`, pgx.StructArgs(write)).Scan(&updatedID); err != nil {
			return dbutil.MutationError(err)
		}
		return replaceCheckTargets(ctx, tx, id, in.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Check, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := dbutil.GetOne[checkRow](ctx, s.db.Pool(), checkSelectSQL()+"\nWHERE c.id = $1", id)
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
	tag, err := s.db.Pool().Exec(ctx, `DELETE FROM checks WHERE id = $1`, id)
	if err != nil {
		return dbutil.DeleteConflict(err, "Check is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple checks. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.db.Pool().Query(ctx, `DELETE FROM checks WHERE id = ANY($1::bigint[]) RETURNING id`, ids)
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
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	where, args := checkListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL: checkSelectSQL(),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: checkOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{
			{SQL: "c.updated_at"},
			{SQL: "c.id"},
		},
		Params: params.ListParams,
	}
	rows, count, err := dbutil.ListWithCount[checkRow](ctx, s.db.Pool(), listQuery)
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
	rows, err := s.db.Pool().Query(ctx, `
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
		FROM checks c
		JOIN host_row h ON true
		WHERE EXISTS (
				SELECT 1
				FROM osquery_check_targets ct
				JOIN label_membership lm ON lm.label_id = ct.label_id AND lm.host_id = h.id
				WHERE ct.check_id = c.id
				  AND ct.direction = 'include'
			)
			AND NOT EXISTS (
				SELECT 1
				FROM osquery_check_targets ct
				JOIN label_membership lm ON lm.label_id = ct.label_id AND lm.host_id = h.id
				WHERE ct.check_id = c.id
				  AND ct.direction = 'exclude'
			)
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

// UpsertMembership records a check result. A nil passes value means not run.
func (s *Store) UpsertMembership(ctx context.Context, checkID int64, hostID int64, passes *bool) error {
	_, err := s.db.Pool().Exec(ctx, `
		INSERT INTO check_membership (check_id, host_id, passes, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (check_id, host_id) DO UPDATE SET
			passes = EXCLUDED.passes,
			updated_at = now()`, checkID, hostID, passes)
	return err
}

func (s *Store) CheckResults(ctx context.Context, checkID int64, response *CheckStatus) ([]CheckHostStatus, error) {
	if response != nil {
		passes, err := passesForCheckStatus(*response)
		if err != nil {
			return nil, err
		}
		rows, err := s.db.Pool().Query(ctx, `
			WITH check_row AS (
				SELECT id, name
				FROM checks c
				WHERE c.id = $1
			)
			SELECT
				c.id AS check_id,
				c.name AS check_name,
				h.id AS host_id,
				h.display_name AS host_name,
				m.passes,
				m.updated_at
			FROM check_row c
			JOIN check_membership m ON m.check_id = c.id AND m.passes = $2::boolean
			JOIN hosts h ON h.id = m.host_id
			ORDER BY lower(h.display_name), h.id`, checkID, passes)
		if err != nil {
			return nil, err
		}
		records, err := pgx.CollectRows(rows, pgx.RowToStructByName[checkHostStatusRow])
		if err != nil {
			return nil, err
		}
		return checkHostStatusesFromRows(records), nil
	}
	rows, err := s.db.Pool().Query(ctx, `
		WITH check_row AS (
			SELECT id, name
			FROM checks c
			WHERE c.id = $1
		),
		host_rows AS (
			SELECT id, display_name
			FROM hosts
		)
		SELECT
			c.id AS check_id,
			c.name AS check_name,
			h.id AS host_id,
			h.display_name AS host_name,
			m.passes,
			m.updated_at
		FROM check_row c
		JOIN host_rows h ON true
		LEFT JOIN check_membership m ON m.host_id = h.id AND m.check_id = c.id
		WHERE EXISTS (
				SELECT 1
				FROM osquery_check_targets ct
				JOIN label_membership lm ON lm.label_id = ct.label_id AND lm.host_id = h.id
				WHERE ct.check_id = c.id
				  AND ct.direction = 'include'
			)
			AND NOT EXISTS (
				SELECT 1
				FROM osquery_check_targets ct
				JOIN label_membership lm ON lm.label_id = ct.label_id AND lm.host_id = h.id
				WHERE ct.check_id = c.id
				  AND ct.direction = 'exclude'
			)
		ORDER BY
			CASE
				WHEN m.passes IS FALSE THEN 0
				WHEN m.passes IS NULL THEN 1
				ELSE 2
			END,
			lower(h.display_name),
			h.id`, checkID)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[checkHostStatusRow])
	if err != nil {
		return nil, err
	}
	return checkHostStatusesFromRows(records), nil
}

func (s *Store) HostChecks(ctx context.Context, host *hosts.Host) ([]CheckHostStatus, error) {
	rows, err := s.db.Pool().Query(ctx, `
		WITH host_row AS (
			SELECT id, display_name
			FROM hosts h
			WHERE h.id = $1
		)
		SELECT
			c.id AS check_id,
			c.name AS check_name,
			h.id AS host_id,
			h.display_name AS host_name,
			m.passes,
			m.updated_at
		FROM checks c
		JOIN host_row h ON true
		LEFT JOIN check_membership m ON m.host_id = h.id AND m.check_id = c.id
		WHERE EXISTS (
				SELECT 1
				FROM osquery_check_targets ct
				JOIN label_membership lm ON lm.label_id = ct.label_id AND lm.host_id = h.id
				WHERE ct.check_id = c.id
				  AND ct.direction = 'include'
			)
			AND NOT EXISTS (
				SELECT 1
				FROM osquery_check_targets ct
				JOIN label_membership lm ON lm.label_id = ct.label_id AND lm.host_id = h.id
				WHERE ct.check_id = c.id
				  AND ct.direction = 'exclude'
			)
		ORDER BY
			CASE
				WHEN m.passes IS FALSE THEN 0
				WHEN m.passes IS NULL THEN 1
				ELSE 2
			END,
			lower(c.name),
			c.id`, host.ID)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[checkHostStatusRow])
	if err != nil {
		return nil, err
	}
	return checkHostStatusesFromRows(records), nil
}

func checkHostStatusesFromRows(rows []checkHostStatusRow) []CheckHostStatus {
	statuses := make([]CheckHostStatus, 0, len(rows))
	for _, row := range rows {
		statuses = append(statuses, CheckHostStatus{
			CheckID:   row.CheckID,
			CheckName: row.CheckName,
			HostID:    row.HostID,
			HostName:  row.HostName,
			Response:  checkStatusFromPasses(row.Passes),
			UpdatedAt: row.UpdatedAt,
		})
	}
	return statuses
}

func checkStatusFromPasses(passes *bool) *CheckStatus {
	if passes == nil {
		return nil
	}
	status := CheckStatusFail
	if *passes {
		status = CheckStatusPass
	}
	return &status
}

func passesForCheckStatus(status CheckStatus) (bool, error) {
	switch status {
	case CheckStatusPass:
		return true, nil
	case CheckStatusFail:
		return false, nil
	default:
		return false, fmt.Errorf("%w: unknown check_response %q", dbutil.ErrInvalidInput, status)
	}
}

type checkCounts struct {
	Passing int32
	Failing int32
}

func (s *Store) loadCheckCounts(ctx context.Context, checkIDs []int64) (map[int64]checkCounts, error) {
	if len(checkIDs) == 0 {
		return map[int64]checkCounts{}, nil
	}
	rows, err := s.db.Pool().Query(ctx, `
		SELECT
			check_id,
			COUNT(*) FILTER (WHERE passes IS TRUE)::integer AS passing_host_count,
			COUNT(*) FILTER (WHERE passes IS FALSE)::integer AS failing_host_count
		FROM check_membership
		WHERE check_id = ANY($1::bigint[])
		GROUP BY check_id`, checkIDs)
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
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add("(c.name ILIKE " + search + " OR c.description ILIKE " + search + " OR c.query ILIKE " + search + ")")
	}
	return where.Build()
}

func checkOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
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
FROM checks c`
}
