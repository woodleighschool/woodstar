package models

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/db"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/platform"
	"github.com/woodleighschool/woodstar/internal/store"
)

// Check is a query-backed pass/fail policy.
type Check struct {
	ID                int64
	Name              string
	Description       string
	Query             string
	Platform          *string
	MinOsqueryVersion *string
	LabelScope        hosts.LabelScope
	CreatedByUserID   *int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CheckCreate contains editable check fields.
type CheckCreate struct {
	Name              string
	Description       string
	Query             string
	Platform          *string
	MinOsqueryVersion *string
	LabelScope        hosts.LabelScope
	CreatedByUserID   *int64
}

// CheckUpdate replaces editable check fields.
type CheckUpdate CheckCreate

// CheckListParams filters checks.
type CheckListParams struct {
	store.ListParams

	Platform string
}

// CheckHostStatus is a check's current state for one host.
type CheckHostStatus struct {
	CheckID         int64
	CheckName       string
	HostID          int64
	HostName        string
	Passes          *bool
	FirstFailedAt   *time.Time
	LastEvaluatedAt *time.Time
}

// CheckStore persists checks and per-host membership state.
type CheckStore struct {
	db *db.DB
}

// NewCheckStore returns a check store backed by db.
func NewCheckStore(db *db.DB) *CheckStore {
	return &CheckStore{db: db}
}

// List returns checks matching params.
func (s *CheckStore) List(ctx context.Context, params CheckListParams) ([]Check, int, error) {
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
	for rows.Next() {
		check, err := scanCheck(rows)
		if err != nil {
			return nil, 0, err
		}
		scope, err := s.loadScope(ctx, check.ID)
		if err != nil {
			return nil, 0, err
		}
		check.LabelScope = scope
		checks = append(checks, *check)
	}
	return checks, count, rows.Err()
}

// GetByID returns one check.
func (s *CheckStore) GetByID(ctx context.Context, id int64) (*Check, error) {
	check, err := scanCheck(s.db.Pool().QueryRow(ctx, checkSelectSQL+" WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	scope, err := s.loadScope(ctx, check.ID)
	if err != nil {
		return nil, err
	}
	check.LabelScope = scope
	return check, nil
}

// Create inserts a check.
func (s *CheckStore) Create(ctx context.Context, params CheckCreate) (*Check, error) {
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
			if store.IsUniqueViolation(err) {
				return store.ErrAlreadyExists
			}
			return err
		}
		if err := replaceScope(ctx, tx, "check_labels", "check_id", check.ID, params.LabelScope); err != nil {
			return err
		}
		check.LabelScope = hosts.NormalizeLabelScope(params.LabelScope)
		created = check
		return nil
	})
	return created, err
}

// Update replaces a check.
func (s *CheckStore) Update(ctx context.Context, id int64, params CheckUpdate) (*Check, error) {
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
			return store.ErrNotFound
		}
		if err != nil {
			if store.IsUniqueViolation(err) {
				return store.ErrAlreadyExists
			}
			return err
		}
		if err := replaceScope(ctx, tx, "check_labels", "check_id", check.ID, cleaned.LabelScope); err != nil {
			return err
		}
		check.LabelScope = hosts.NormalizeLabelScope(cleaned.LabelScope)
		updated = check
		return nil
	})
	return updated, err
}

// Delete removes a check.
func (s *CheckStore) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(ctx, "DELETE FROM checks WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple checks. Missing IDs are ignored for bulk idempotency.
func (s *CheckStore) DeleteMany(ctx context.Context, ids []int64) (int, error) {
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
func (s *CheckStore) ApplicableForHost(ctx context.Context, host hosts.Host) ([]Check, error) {
	rows, err := s.db.Pool().Query(ctx, checkSelectSQL+" ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	checks := make([]Check, 0)
	for rows.Next() {
		check, err := scanCheck(rows)
		if err != nil {
			return nil, err
		}
		if !hosts.QueryMatchesHost(check.Platform, check.MinOsqueryVersion, host) {
			continue
		}
		scope, err := s.loadScope(ctx, check.ID)
		if err != nil {
			return nil, err
		}
		matches, err := hosts.HostMatchesScope(ctx, s.db, scope, host.ID)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}
		check.LabelScope = scope
		checks = append(checks, *check)
	}
	return checks, rows.Err()
}

// UpsertMembership records a check result. A nil passes value means no-response.
func (s *CheckStore) UpsertMembership(ctx context.Context, checkID int64, hostID int64, passes *bool) error {
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
func (s *CheckStore) HostStatuses(ctx context.Context, checkID int64) ([]CheckHostStatus, error) {
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
func (s *CheckStore) HostChecks(ctx context.Context, host hosts.Host) ([]CheckHostStatus, error) {
	checks, err := s.ApplicableForHost(ctx, host)
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
		err := s.db.Pool().QueryRow(ctx,
			`SELECT passes, first_failed_at, last_evaluated_at
			 FROM check_membership
			 WHERE check_id = $1 AND host_id = $2`,
			check.ID,
			host.ID,
		).Scan(&status.Passes, &status.FirstFailedAt, &status.LastEvaluatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			out = append(out, status)
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, status)
	}
	return out, nil
}

func cleanCheckCreate(params CheckCreate) (CheckCreate, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = strings.TrimSpace(params.Query)
	params.Platform = cleanPlatformPtr(params.Platform)
	params.MinOsqueryVersion = cleanStringPtr(params.MinOsqueryVersion)
	params.LabelScope = hosts.NormalizeLabelScope(params.LabelScope)
	if params.Name == "" {
		return CheckCreate{}, fmt.Errorf("%w: name is required", store.ErrInvalidInput)
	}
	if params.Query == "" {
		return CheckCreate{}, fmt.Errorf("%w: query is required", store.ErrInvalidInput)
	}
	return params, nil
}

func cleanCheckListParams(params CheckListParams) CheckListParams {
	params.ListParams = store.CleanListParams(params.ListParams)
	params.Platform = platform.CleanPlatform(params.Platform)
	return params
}

func (s *CheckStore) loadScope(ctx context.Context, checkID int64) (hosts.LabelScope, error) {
	rows, err := s.db.Pool().Query(ctx,
		"SELECT label_id, exclude, require_all FROM check_labels WHERE check_id = $1 ORDER BY label_id",
		checkID,
	)
	if err != nil {
		return hosts.LabelScope{}, err
	}
	defer rows.Close()
	return scanScopeRows(rows)
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
	return store.NameSearchAndPlatformWhere(params.Q, params.Platform)
}

func checkListSQL(where string, args []any, params CheckListParams) (string, []any, error) {
	return store.ListQuery{
		SelectSQL: checkSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]store.OrderExpr{
			"name":               {SQL: "name"},
			"created_at":         {SQL: "created_at"},
			store.OrderUpdatedAt: {SQL: store.OrderUpdatedAt},
		},
		DefaultOrder: []store.OrderExpr{{SQL: store.OrderUpdatedAt}, {SQL: "id"}},
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
