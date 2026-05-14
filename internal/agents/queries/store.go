package queries

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/platform"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// Store persists saved queries and scheduled report results.
type Store struct {
	db     *database.DB
	scopes *scope.Store
}

// NewStore returns a query store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{db: db, scopes: scope.NewStore(db)}
}

// List returns saved queries matching params.
func (s *Store) List(ctx context.Context, params QueryListParams) ([]Query, int, error) {
	params = cleanQueryListParams(params)
	where, args := queryListWhere(params)

	var count int
	if err := s.db.Pool().QueryRow(ctx, "SELECT count(*) FROM queries "+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	query, args, err := queryListSQL(where, args, params)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	queries := make([]Query, 0)
	queryIDs := make([]int64, 0)
	for rows.Next() {
		query, err := scanQuery(rows)
		if err != nil {
			return nil, 0, err
		}
		queries = append(queries, *query)
		queryIDs = append(queryIDs, query.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	scopes, err := s.scopes.LoadQueries(ctx, queryIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range queries {
		queries[i].LabelScope = scopes[queries[i].ID]
	}
	return queries, count, nil
}

// GetByID returns a saved query by database ID.
func (s *Store) GetByID(ctx context.Context, id int64) (*Query, error) {
	query, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	labelScope, err := s.scopes.LoadQuery(ctx, query.ID)
	if err != nil {
		return nil, err
	}
	query.LabelScope = labelScope
	return query, nil
}

func (s *Store) getByID(ctx context.Context, id int64) (*Query, error) {
	query, err := scanQuery(s.db.Pool().QueryRow(ctx, querySelectSQL+" WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	return query, err
}

// Create inserts a saved query.
func (s *Store) Create(ctx context.Context, params QueryCreate) (*Query, error) {
	params, err := cleanQueryCreate(params)
	if err != nil {
		return nil, err
	}

	var created *Query
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, queryInsertSQL,
			params.Name,
			params.Description,
			params.Query,
			params.Platform,
			params.MinOsqueryVersion,
			params.ScheduleInterval,
			params.CreatedByUserID,
		)
		query, err := scanQuery(row)
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		if err := s.scopes.ReplaceQuery(ctx, tx, query.ID, params.LabelScope); err != nil {
			return err
		}
		query.LabelScope = scope.NormalizeLabelScope(params.LabelScope)
		created = query
		return nil
	})
	return created, err
}

// Update replaces a saved query.
func (s *Store) Update(ctx context.Context, id int64, params QueryUpdate) (*Query, error) {
	cleaned, err := cleanQueryUpdate(params)
	if err != nil {
		return nil, err
	}

	var updated *Query
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, queryUpdateSQL,
			cleaned.Name,
			cleaned.Description,
			cleaned.Query,
			cleaned.Platform,
			cleaned.MinOsqueryVersion,
			cleaned.ScheduleInterval,
			id,
		)
		query, err := scanQuery(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		}
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		if err := s.scopes.ReplaceQuery(ctx, tx, query.ID, cleaned.LabelScope); err != nil {
			return err
		}
		query.LabelScope = scope.NormalizeLabelScope(cleaned.LabelScope)
		updated = query
		return nil
	})
	return updated, err
}

// Delete removes a saved query.
func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(ctx, "DELETE FROM queries WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple saved queries. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := s.db.Pool().Exec(ctx, "DELETE FROM queries WHERE id = ANY($1::bigint[])", ids)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ScheduledForHost returns scheduled report queries applicable to host.
func (s *Store) ScheduledForHost(ctx context.Context, host *hosts.Host) ([]Query, error) {
	rows, err := s.db.Pool().Query(ctx, querySelectSQL+" WHERE schedule_interval > 0 ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	queries := make([]Query, 0)
	queryIDs := make([]int64, 0)
	for rows.Next() {
		query, err := scanQuery(rows)
		if err != nil {
			return nil, err
		}
		if !hosts.QueryMatchesHost(query.Platform, query.MinOsqueryVersion, host) {
			continue
		}
		queries = append(queries, *query)
		queryIDs = append(queryIDs, query.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	scopes, err := s.scopes.LoadQueries(ctx, queryIDs)
	if err != nil {
		return nil, err
	}
	matches, err := s.scopes.MatchHostScopes(ctx, host.ID, scopes)
	if err != nil {
		return nil, err
	}
	out := queries[:0]
	for _, query := range queries {
		labelScope := scopes[query.ID]
		if !matches[query.ID] {
			continue
		}
		query.LabelScope = labelScope
		out = append(out, query)
	}
	return out, nil
}

func cleanQueryCreate(params QueryCreate) (QueryCreate, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = strings.TrimSpace(params.Query)
	params.Platform = platform.CleanPtr(params.Platform)
	params.MinOsqueryVersion = dbutil.CleanStringPtr(params.MinOsqueryVersion)
	params.LabelScope = scope.NormalizeLabelScope(params.LabelScope)
	if params.Name == "" {
		return QueryCreate{}, fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if params.Query == "" {
		return QueryCreate{}, fmt.Errorf("%w: query is required", dbutil.ErrInvalidInput)
	}
	if params.ScheduleInterval < 0 {
		return QueryCreate{}, fmt.Errorf("%w: schedule interval cannot be negative", dbutil.ErrInvalidInput)
	}
	return params, nil
}

func cleanQueryUpdate(params QueryUpdate) (QueryUpdate, error) {
	cleaned, err := cleanQueryCreate(QueryCreate{
		Name:              params.Name,
		Description:       params.Description,
		Query:             params.Query,
		Platform:          params.Platform,
		MinOsqueryVersion: params.MinOsqueryVersion,
		ScheduleInterval:  params.ScheduleInterval,
		LabelScope:        params.LabelScope,
	})
	if err != nil {
		return QueryUpdate{}, err
	}
	return QueryUpdate{
		Name:              cleaned.Name,
		Description:       cleaned.Description,
		Query:             cleaned.Query,
		Platform:          cleaned.Platform,
		MinOsqueryVersion: cleaned.MinOsqueryVersion,
		ScheduleInterval:  cleaned.ScheduleInterval,
		LabelScope:        cleaned.LabelScope,
	}, nil
}

func cleanQueryListParams(params QueryListParams) QueryListParams {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	params.Platform = platform.CleanPlatform(params.Platform)
	return params
}

func scanQuery(row pgx.Row) (*Query, error) {
	var query Query
	err := row.Scan(
		&query.ID,
		&query.Name,
		&query.Description,
		&query.Query,
		&query.Platform,
		&query.MinOsqueryVersion,
		&query.ScheduleInterval,
		&query.CreatedByUserID,
		&query.CreatedAt,
		&query.UpdatedAt,
	)
	return &query, err
}

func queryListWhere(params QueryListParams) (string, []any) {
	return dbutil.NameSearchAndPlatformWhere(params.Q, params.Platform)
}

func queryListSQL(where string, args []any, params QueryListParams) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL: querySelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":                {SQL: "name"},
			"created_at":          {SQL: "created_at"},
			dbutil.OrderUpdatedAt: {SQL: dbutil.OrderUpdatedAt},
			"schedule_interval":   {SQL: "schedule_interval"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: dbutil.OrderUpdatedAt}, {SQL: "id"}},
		Params:       params.ListParams,
	}.Build()
}

const querySelectSQL = `
SELECT id, name, description, query, platform, min_osquery_version, schedule_interval,
       created_by_user_id, created_at, updated_at
FROM queries`

const queryInsertSQL = `
INSERT INTO queries (
    name, description, query, platform, min_osquery_version, schedule_interval,
    created_by_user_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, description, query, platform, min_osquery_version, schedule_interval,
          created_by_user_id, created_at, updated_at`

const queryUpdateSQL = `
UPDATE queries
SET name = $1,
    description = $2,
    query = $3,
    platform = $4,
    min_osquery_version = $5,
    schedule_interval = $6,
    updated_at = now()
WHERE id = $7
RETURNING id, name, description, query, platform, min_osquery_version, schedule_interval,
          created_by_user_id, created_at, updated_at`
