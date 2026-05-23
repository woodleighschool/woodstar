// Package checks persists query-backed pass/fail policies and per-host results.
package checks

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/queries"
	"github.com/woodleighschool/woodstar/internal/platforms"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// Store persists checks and per-host membership state.
type Store struct {
	db     *database.DB
	q      *sqlc.Queries
	scopes *scope.Store
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: sqlc.New(db.Pool()), scopes: scope.NewStore(db)}
}

func (s *Store) List(ctx context.Context, params CheckListParams) ([]Check, int, error) {
	params = cleanCheckListParams(params)
	where, args := checkListWhere(params)

	var count int
	if err := s.db.Pool().QueryRow(ctx, "SELECT count(*) FROM checks "+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	query, args, err := checkListSQL(where, args, params)
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
	scopes, err := s.scopes.LoadChecks(ctx, checkIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range checks {
		checks[i].LabelScope = scopes[checks[i].ID]
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
	labelScope, err := s.scopes.LoadCheck(ctx, check.ID)
	if err != nil {
		return nil, err
	}
	check.LabelScope = labelScope
	return check, nil
}

func (s *Store) Create(ctx context.Context, params CheckCreate) (*Check, error) {
	params, err := cleanCheckCreate(params)
	if err != nil {
		return nil, err
	}

	var created *Check
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).CreateCheck(ctx, sqlc.CreateCheckParams{
			Name:            params.Name,
			Description:     params.Description,
			Query:           params.Query,
			Platforms:       toSQLCPlatforms(params.Platforms),
			CreatedByUserID: params.CreatedByUserID,
		})
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		check := checkFromSQLC(row)
		if err := s.scopes.ReplaceCheck(ctx, tx, check.ID, params.LabelScope); err != nil {
			return err
		}
		check.LabelScope = scope.NormalizeLabelScope(params.LabelScope)
		created = check
		return nil
	})
	return created, err
}

func (s *Store) Update(ctx context.Context, id int64, params CheckUpdate) (*Check, error) {
	cleaned, err := cleanCheckUpdate(params)
	if err != nil {
		return nil, err
	}

	var updated *Check
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).UpdateCheck(ctx, sqlc.UpdateCheckParams{
			Name:        cleaned.Name,
			Description: cleaned.Description,
			Query:       cleaned.Query,
			Platforms:   toSQLCPlatforms(cleaned.Platforms),
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
		if err := s.scopes.ReplaceCheck(ctx, tx, check.ID, cleaned.LabelScope); err != nil {
			return err
		}
		check.LabelScope = scope.NormalizeLabelScope(cleaned.LabelScope)
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

func cleanCheckCreate(params CheckCreate) (CheckCreate, error) {
	cleaned, err := queries.CleanQueryDefinition(queries.QueryDefinition{
		Name:        params.Name,
		Description: params.Description,
		Query:       params.Query,
		Platforms:   params.Platforms,
		LabelScope:  params.LabelScope,
	})
	if err != nil {
		return CheckCreate{}, err
	}
	params.Name = cleaned.Name
	params.Description = cleaned.Description
	params.Query = cleaned.Query
	params.Platforms = cleaned.Platforms
	params.LabelScope = cleaned.LabelScope
	return params, nil
}

func cleanCheckUpdate(params CheckUpdate) (CheckUpdate, error) {
	cleaned, err := cleanCheckCreate(CheckCreate{
		Name:        params.Name,
		Description: params.Description,
		Query:       params.Query,
		Platforms:   params.Platforms,
		LabelScope:  params.LabelScope,
	})
	if err != nil {
		return CheckUpdate{}, err
	}
	return CheckUpdate{
		Name:        cleaned.Name,
		Description: cleaned.Description,
		Query:       cleaned.Query,
		Platforms:   cleaned.Platforms,
		LabelScope:  cleaned.LabelScope,
	}, nil
}

func cleanCheckListParams(params CheckListParams) CheckListParams {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	params.Platform = platforms.CleanPlatform(params.Platform)
	return params
}

func scanCheck(row pgx.Row) (*Check, error) {
	var check Check
	var sqlcPlatforms []sqlc.Platform
	err := row.Scan(
		&check.ID,
		&check.Name,
		&check.Description,
		&check.Query,
		&sqlcPlatforms,
		&check.CreatedByUserID,
		&check.CreatedAt,
		&check.UpdatedAt,
	)
	check.Platforms = platformsFromSQLC(sqlcPlatforms)
	return &check, err
}

func checkFromSQLC(row sqlc.Check) *Check {
	return &Check{
		ID:              row.ID,
		Name:            row.Name,
		Description:     row.Description,
		Query:           row.Query,
		Platforms:       platformsFromSQLC(row.Platforms),
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

func toSQLCPlatforms(values []platforms.Platform) []sqlc.Platform {
	out := make([]sqlc.Platform, len(values))
	for i, value := range values {
		out[i] = sqlc.Platform(value)
	}
	return out
}

func platformsFromSQLC(values []sqlc.Platform) []platforms.Platform {
	out := make([]platforms.Platform, len(values))
	for i, value := range values {
		out[i] = platforms.Platform(value)
	}
	return out
}

func checkListWhere(params CheckListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + strings.ToLower(params.Q) + "%")
		where.Add("(lower(name) LIKE " + search + " OR lower(description) LIKE " + search + ")")
	}
	if params.Platform != "" {
		where.Add(where.Arg(params.Platform) + " = ANY(platforms::text[])")
	}
	return where.Build()
}

func checkListSQL(where string, args []any, params CheckListParams) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL: checkSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":                {SQL: "name"},
			"created_at":          {SQL: "created_at"},
			dbutil.OrderUpdatedAt: {SQL: dbutil.OrderUpdatedAt},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: dbutil.OrderUpdatedAt}, {SQL: "id"}},
		Params:       params.ListParams,
	}.Build()
}

const checkSelectSQL = `
SELECT id, name, description, query, platforms,
       created_by_user_id, created_at, updated_at
FROM checks`
