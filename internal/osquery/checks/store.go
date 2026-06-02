// Package checks persists query-backed pass/fail policies and per-host results.
package checks

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// Store persists checks and per-host membership state.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: sqlc.New(db.Pool())}
}

func (s *Store) List(ctx context.Context, params CheckListParams) ([]Check, int, error) {
	where, args := checkListWhere(params)
	listQuery := checkListQuery(where, args, params)

	var count int
	countSQL, countArgs := listQuery.BuildCount()
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}

	query, args, err := listQuery.Build()
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	checks := make([]Check, 0)
	checkIDs := make([]int64, 0)
	for rows.Next() {
		check, err := scanCheck(rows)
		if err != nil {
			return nil, 0, err
		}
		checks = append(checks, *check)
		checkIDs = append(checkIDs, check.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
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

func (s *Store) GetByID(ctx context.Context, id int64) (*Check, error) {
	row, err := s.q.GetCheckByID(ctx, sqlc.GetCheckByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	check := checkFromSQLC(row)
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

func (s *Store) Create(ctx context.Context, params CheckMutation) (*Check, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	var created *Check
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).CreateCheck(ctx, sqlc.CreateCheckParams{
			Name:            params.Name,
			Description:     params.Description,
			Query:           params.Query,
			CreatedByUserID: params.CreatedByUserID,
		})
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		check := checkFromSQLC(row)
		if err := replaceCheckTargets(ctx, tx, check.ID, params.Targets); err != nil {
			return err
		}
		check.Targets = params.Targets
		created = check
		return nil
	})
	return created, err
}

func (s *Store) Update(ctx context.Context, id int64, params CheckMutation) (*Check, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	var updated *Check
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).UpdateCheck(ctx, sqlc.UpdateCheckParams{
			Name:        params.Name,
			Description: params.Description,
			Query:       params.Query,
			ID:          id,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		}
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		check := checkFromSQLC(row)
		if err := replaceCheckTargets(ctx, tx, check.ID, params.Targets); err != nil {
			return err
		}
		check.Targets = params.Targets
		updated = check
		return nil
	})
	return updated, err
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteCheck(ctx, sqlc.DeleteCheckParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	if err != nil {
		return err
	}
	return nil
}

// DeleteMany removes multiple checks. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deletedIDs, err := s.q.DeleteChecks(ctx, sqlc.DeleteChecksParams{Ids: ids})
	if err != nil {
		return 0, err
	}
	return len(deletedIDs), nil
}

func (s *Store) ApplicableForHost(ctx context.Context, host *hosts.Host) ([]Check, error) {
	rows, err := s.q.ListApplicableChecksForHost(ctx, sqlc.ListApplicableChecksForHostParams{HostID: host.ID})
	if err != nil {
		return nil, err
	}
	checks := make([]Check, 0, len(rows))
	for _, row := range rows {
		checks = append(checks, *checkFromSQLC(row))
	}
	return checks, nil
}

// UpsertMembership records a check result. A nil passes value means not run.
func (s *Store) UpsertMembership(ctx context.Context, checkID int64, hostID int64, passes *bool) error {
	return s.q.UpsertCheckMembership(ctx, sqlc.UpsertCheckMembershipParams{
		CheckID: checkID,
		HostID:  hostID,
		Passes:  passes,
	})
}

func (s *Store) HostStatuses(ctx context.Context, checkID int64) ([]CheckHostStatus, error) {
	rows, err := s.q.ListCheckHostStatuses(ctx, sqlc.ListCheckHostStatusesParams{CheckID: checkID})
	if err != nil {
		return nil, err
	}
	return checkHostStatusesFromCheckRows(rows), nil
}

func (s *Store) HostChecks(ctx context.Context, host *hosts.Host) ([]CheckHostStatus, error) {
	rows, err := s.q.ListHostCheckStatusesForHost(ctx, sqlc.ListHostCheckStatusesForHostParams{HostID: host.ID})
	if err != nil {
		return nil, err
	}
	return checkHostStatusesFromHostRows(rows), nil
}

func scanCheck(row pgx.Row) (*Check, error) {
	var check Check
	err := row.Scan(
		&check.ID,
		&check.Name,
		&check.Description,
		&check.Query,
		&check.CreatedByUserID,
		&check.CreatedAt,
		&check.UpdatedAt,
	)
	return &check, err
}

func checkFromSQLC(row sqlc.Check) *Check {
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

func checkHostStatusesFromCheckRows(rows []sqlc.ListCheckHostStatusesRow) []CheckHostStatus {
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

func checkHostStatusesFromHostRows(rows []sqlc.ListHostCheckStatusesForHostRow) []CheckHostStatus {
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

type checkCounts struct {
	Passing int32
	Failing int32
}

func (s *Store) loadCheckCounts(ctx context.Context, checkIDs []int64) (map[int64]checkCounts, error) {
	if len(checkIDs) == 0 {
		return map[int64]checkCounts{}, nil
	}
	rows, err := s.q.ListCheckCounts(ctx, sqlc.ListCheckCountsParams{CheckIds: checkIDs})
	if err != nil {
		return nil, err
	}

	counts := make(map[int64]checkCounts, len(checkIDs))
	for _, row := range rows {
		counts[row.CheckID] = checkCounts{
			Passing: row.PassingHostCount,
			Failing: row.FailingHostCount,
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

func checkListQuery(where string, args []any, params CheckListParams) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: checkSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":                {SQL: "c.name"},
			"created_at":          {SQL: "c.created_at"},
			dbutil.OrderUpdatedAt: {SQL: "c.updated_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "c.updated_at"}, {SQL: "c.id"}},
		Params:       params.ListParams,
	}
}

const checkSelectSQL = `
SELECT c.id, c.name, c.description, c.query,
       c.created_by_user_id, c.created_at, c.updated_at
FROM checks c`
