// Package checks persists query-backed pass/fail policies and per-host results.
package checks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/platform"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// Store persists checks and per-host membership state.
type Store struct {
	db     *database.DB
	scopes *scope.Store
}

// NewStore returns a check store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{db: db, scopes: scope.NewStore(db)}
}

// List returns checks matching params.
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

// GetByID returns one check.
func (s *Store) GetByID(ctx context.Context, id int64) (*Check, error) {
	check, err := scanCheck(s.db.Pool().QueryRow(ctx, checkSelectSQL+" WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	labelScope, err := s.scopes.LoadCheck(ctx, check.ID)
	if err != nil {
		return nil, err
	}
	check.LabelScope = labelScope
	return check, nil
}

// Create inserts a check.
func (s *Store) Create(ctx context.Context, params CheckCreate) (*Check, error) {
	params, err := cleanCheckCreate(params)
	if err != nil {
		return nil, err
	}

	var created *Check
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, checkInsertSQL,
			params.Name,
			params.Description,
			params.Query,
			params.Platform,
			params.MinOsqueryVersion,
			params.CreatedByUserID,
		)
		check, err := scanCheck(row)
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		if err := s.scopes.ReplaceCheck(ctx, tx, check.ID, params.LabelScope); err != nil {
			return err
		}
		check.LabelScope = scope.NormalizeLabelScope(params.LabelScope)
		created = check
		return nil
	})
	return created, err
}

// Update replaces a check.
func (s *Store) Update(ctx context.Context, id int64, params CheckUpdate) (*Check, error) {
	cleaned, err := cleanCheckCreate(CheckCreate(params))
	if err != nil {
		return nil, err
	}

	var updated *Check
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, checkUpdateSQL,
			cleaned.Name,
			cleaned.Description,
			cleaned.Query,
			cleaned.Platform,
			cleaned.MinOsqueryVersion,
			id,
		)
		check, err := scanCheck(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		}
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		if err := s.scopes.ReplaceCheck(ctx, tx, check.ID, cleaned.LabelScope); err != nil {
			return err
		}
		check.LabelScope = scope.NormalizeLabelScope(cleaned.LabelScope)
		updated = check
		return nil
	})
	return updated, err
}

// Delete removes a check.
func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(ctx, "DELETE FROM checks WHERE id = $1", id)
	if err != nil {
		return err
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
	tag, err := s.db.Pool().Exec(ctx, "DELETE FROM checks WHERE id = ANY($1::bigint[])", ids)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ApplicableForHost returns checks that should run on host.
func (s *Store) ApplicableForHost(ctx context.Context, host *hosts.Host) ([]Check, error) {
	rows, err := s.db.Pool().Query(ctx, checkSelectSQL+" ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	checks := make([]Check, 0)
	checkIDs := make([]int64, 0)
	for rows.Next() {
		check, err := scanCheck(rows)
		if err != nil {
			return nil, err
		}
		if !hosts.QueryMatchesHost(check.Platform, check.MinOsqueryVersion, host) {
			continue
		}
		checks = append(checks, *check)
		checkIDs = append(checkIDs, check.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	scopes, err := s.scopes.LoadChecks(ctx, checkIDs)
	if err != nil {
		return nil, err
	}
	matches, err := s.scopes.MatchHostScopes(ctx, host.ID, scopes)
	if err != nil {
		return nil, err
	}
	out := checks[:0]
	for _, check := range checks {
		labelScope := scopes[check.ID]
		if !matches[check.ID] {
			continue
		}
		check.LabelScope = labelScope
		out = append(out, check)
	}
	return out, nil
}

// UpsertMembership records a check result. A nil passes value means no-response.
func (s *Store) UpsertMembership(ctx context.Context, checkID int64, hostID int64, passes *bool) error {
	_, err := s.db.Pool().Exec(ctx,
		`INSERT INTO check_membership (
		     check_id, host_id, passes, first_failed_at, last_evaluated_at, updated_at
		 )
		 VALUES ($1, $2, $3, CASE WHEN $3::boolean IS FALSE THEN now() ELSE NULL END, now(), now())
		 ON CONFLICT (check_id, host_id) DO UPDATE SET
		     passes = EXCLUDED.passes,
		     first_failed_at = CASE
		         WHEN EXCLUDED.passes IS TRUE THEN NULL
		         WHEN EXCLUDED.passes IS FALSE THEN
		             CASE
		                 WHEN check_membership.passes IS FALSE THEN check_membership.first_failed_at
		                 ELSE now()
		             END
		         ELSE check_membership.first_failed_at
		     END,
		     last_evaluated_at = now(),
		     updated_at = now()`,
		checkID,
		hostID,
		passes,
	)
	return err
}

// HostStatuses returns check status rows for one check.
func (s *Store) HostStatuses(ctx context.Context, checkID int64) ([]CheckHostStatus, error) {
	rows, err := s.db.Pool().Query(ctx,
		`SELECT c.id, c.name, h.id, h.display_name,
		        m.passes, m.first_failed_at, m.last_evaluated_at
		 FROM checks c
		 CROSS JOIN hosts h
		 LEFT JOIN check_membership m ON m.host_id = h.id AND m.check_id = c.id
		 WHERE c.id = $1 AND h.deleted_at IS NULL
		 ORDER BY m.passes ASC NULLS FIRST, h.display_name`,
		checkID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCheckStatuses(rows)
}

// HostChecks returns check status rows applicable to one host.
func (s *Store) HostChecks(ctx context.Context, host *hosts.Host) ([]CheckHostStatus, error) {
	checks, err := s.ApplicableForHost(ctx, host)
	if err != nil {
		return nil, err
	}
	checkIDs := make([]int64, 0, len(checks))
	for _, check := range checks {
		checkIDs = append(checkIDs, check.ID)
	}
	states, err := s.hostCheckStates(ctx, host.ID, checkIDs)
	if err != nil {
		return nil, err
	}
	out := make([]CheckHostStatus, 0, len(checks))
	for _, check := range checks {
		status := CheckHostStatus{
			CheckID:   check.ID,
			CheckName: check.Name,
			HostID:    host.ID,
			HostName:  host.DisplayName,
		}
		if state, ok := states[check.ID]; ok {
			status.Passes = state.passes
			status.FirstFailedAt = state.firstFailedAt
			status.LastEvaluatedAt = state.lastEvaluatedAt
		}
		out = append(out, status)
	}
	return out, nil
}

type hostCheckState struct {
	passes          *bool
	firstFailedAt   *time.Time
	lastEvaluatedAt *time.Time
}

func (s *Store) hostCheckStates(
	ctx context.Context,
	hostID int64,
	checkIDs []int64,
) (map[int64]hostCheckState, error) {
	states := make(map[int64]hostCheckState, len(checkIDs))
	if len(checkIDs) == 0 {
		return states, nil
	}
	rows, err := s.db.Pool().Query(ctx,
		`SELECT check_id, passes, first_failed_at, last_evaluated_at
		 FROM check_membership
		 WHERE host_id = $1 AND check_id = ANY($2::bigint[])`,
		hostID,
		checkIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var checkID int64
		var state hostCheckState
		if err := rows.Scan(&checkID, &state.passes, &state.firstFailedAt, &state.lastEvaluatedAt); err != nil {
			return nil, err
		}
		states[checkID] = state
	}
	return states, rows.Err()
}

func cleanCheckCreate(params CheckCreate) (CheckCreate, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = strings.TrimSpace(params.Query)
	params.Platform = platform.CleanPtr(params.Platform)
	params.MinOsqueryVersion = dbutil.CleanStringPtr(params.MinOsqueryVersion)
	params.LabelScope = scope.NormalizeLabelScope(params.LabelScope)
	if params.Name == "" {
		return CheckCreate{}, fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if params.Query == "" {
		return CheckCreate{}, fmt.Errorf("%w: query is required", dbutil.ErrInvalidInput)
	}
	return params, nil
}

func cleanCheckListParams(params CheckListParams) CheckListParams {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	params.Platform = platform.CleanPlatform(params.Platform)
	return params
}

func scanCheck(row pgx.Row) (*Check, error) {
	var check Check
	err := row.Scan(
		&check.ID,
		&check.Name,
		&check.Description,
		&check.Query,
		&check.Platform,
		&check.MinOsqueryVersion,
		&check.CreatedByUserID,
		&check.CreatedAt,
		&check.UpdatedAt,
	)
	return &check, err
}

func scanCheckStatuses(rows pgx.Rows) ([]CheckHostStatus, error) {
	statuses := make([]CheckHostStatus, 0)
	for rows.Next() {
		var status CheckHostStatus
		if err := rows.Scan(
			&status.CheckID,
			&status.CheckName,
			&status.HostID,
			&status.HostName,
			&status.Passes,
			&status.FirstFailedAt,
			&status.LastEvaluatedAt,
		); err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}
	return statuses, rows.Err()
}

func checkListWhere(params CheckListParams) (string, []any) {
	return dbutil.NameSearchAndPlatformWhere(params.Q, params.Platform)
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
SELECT id, name, description, query, platform, min_osquery_version,
       created_by_user_id, created_at, updated_at
FROM checks`

const checkInsertSQL = `
INSERT INTO checks (
    name, description, query, platform, min_osquery_version, created_by_user_id
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, description, query, platform, min_osquery_version,
          created_by_user_id, created_at, updated_at`

const checkUpdateSQL = `
UPDATE checks
SET name = $1,
    description = $2,
    query = $3,
    platform = $4,
    min_osquery_version = $5,
    updated_at = now()
WHERE id = $6
RETURNING id, name, description, query, platform, min_osquery_version,
          created_by_user_id, created_at, updated_at`
