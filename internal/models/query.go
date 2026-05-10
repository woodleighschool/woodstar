package models

import (
	"context"
	"encoding/json"
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

// QueryLoggingType is the storage mode for scheduled query results.
type QueryLoggingType string

const (
	QueryLoggingSnapshot QueryLoggingType = "snapshot"
)

// Query is admin-authored osquery SQL. Scheduled snapshot queries are reports.
type Query struct {
	ID                int64
	Name              string
	Description       string
	Query             string
	Platform          *string
	MinOsqueryVersion *string
	ScheduleInterval  int
	LoggingType       QueryLoggingType
	LabelScope        hosts.LabelScope
	CreatedByUserID   *int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// QueryCreate contains editable query fields.
type QueryCreate struct {
	Name              string
	Description       string
	Query             string
	Platform          *string
	MinOsqueryVersion *string
	ScheduleInterval  int
	LoggingType       QueryLoggingType
	LabelScope        hosts.LabelScope
	CreatedByUserID   *int64
}

// QueryUpdate replaces editable query fields.
type QueryUpdate QueryCreate

// QueryListParams filters saved query lists.
type QueryListParams struct {
	store.ListParams

	Platform string
}

// QueryResult is one stored report row from one host.
type QueryResult struct {
	QueryID     int64
	QueryName   string
	HostID      int64
	HostName    string
	Columns     map[string]string
	LastFetched time.Time
}

// HostReport is a scheduled report as it appears on one host detail page.
type HostReport struct {
	ReportID        int64
	Name            string
	Description     string
	LastFetched     *time.Time
	FirstResult     map[string]string
	HostResultCount int
}

type snapshotResultRow struct {
	data        *json.RawMessage
	lastFetched time.Time
}

// QueryStore persists saved queries and scheduled report results.
type QueryStore struct {
	db *db.DB
}

// NewQueryStore returns a query store backed by db.
func NewQueryStore(db *db.DB) *QueryStore {
	return &QueryStore{db: db}
}

// List returns saved queries matching params.
func (s *QueryStore) List(ctx context.Context, params QueryListParams) ([]Query, int, error) {
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
	for rows.Next() {
		query, err := scanQuery(rows)
		if err != nil {
			return nil, 0, err
		}
		scope, err := s.loadScope(ctx, "query_labels", "query_id", query.ID)
		if err != nil {
			return nil, 0, err
		}
		query.LabelScope = scope
		queries = append(queries, *query)
	}
	return queries, count, rows.Err()
}

// GetByID returns a saved query by database ID.
func (s *QueryStore) GetByID(ctx context.Context, id int64) (*Query, error) {
	query, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	scope, err := s.loadScope(ctx, "query_labels", "query_id", query.ID)
	if err != nil {
		return nil, err
	}
	query.LabelScope = scope
	return query, nil
}

func (s *QueryStore) getByID(ctx context.Context, id int64) (*Query, error) {
	query, err := scanQuery(s.db.Pool().QueryRow(ctx, querySelectSQL+" WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return query, err
}

// Create inserts a saved query.
func (s *QueryStore) Create(ctx context.Context, params QueryCreate) (*Query, error) {
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
			string(params.LoggingType),
			params.CreatedByUserID,
		)
		query, err := scanQuery(row)
		if err != nil {
			if store.IsUniqueViolation(err) {
				return store.ErrAlreadyExists
			}
			return err
		}
		if err := replaceScope(ctx, tx, "query_labels", "query_id", query.ID, params.LabelScope); err != nil {
			return err
		}
		query.LabelScope = hosts.NormalizeLabelScope(params.LabelScope)
		created = query
		return nil
	})
	return created, err
}

// Update replaces a saved query.
func (s *QueryStore) Update(ctx context.Context, id int64, params QueryUpdate) (*Query, error) {
	cleaned, err := cleanQueryCreate(QueryCreate(params))
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
			string(cleaned.LoggingType),
			id,
		)
		query, err := scanQuery(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return store.ErrNotFound
		}
		if err != nil {
			if store.IsUniqueViolation(err) {
				return store.ErrAlreadyExists
			}
			return err
		}
		if err := replaceScope(ctx, tx, "query_labels", "query_id", query.ID, cleaned.LabelScope); err != nil {
			return err
		}
		query.LabelScope = hosts.NormalizeLabelScope(cleaned.LabelScope)
		updated = query
		return nil
	})
	return updated, err
}

// Delete removes a saved query.
func (s *QueryStore) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(ctx, "DELETE FROM queries WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple saved queries. Missing IDs are ignored for bulk idempotency.
func (s *QueryStore) DeleteMany(ctx context.Context, ids []int64) (int, error) {
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
func (s *QueryStore) ScheduledForHost(ctx context.Context, host hosts.Host) ([]Query, error) {
	rows, err := s.db.Pool().Query(ctx, querySelectSQL+" WHERE schedule_interval > 0 ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	queries := make([]Query, 0)
	for rows.Next() {
		query, err := scanQuery(rows)
		if err != nil {
			return nil, err
		}
		if !hosts.QueryMatchesHost(query.Platform, query.MinOsqueryVersion, host) {
			continue
		}
		scope, err := s.loadScope(ctx, "query_labels", "query_id", query.ID)
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
		query.LabelScope = scope
		queries = append(queries, *query)
	}
	return queries, rows.Err()
}

// OverwriteResults replaces the snapshot rows for a query on one host.
func (s *QueryStore) OverwriteResults(
	ctx context.Context,
	queryID int64,
	hostID int64,
	rows []map[string]string,
	fetchedAt time.Time,
) error {
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	resultRows, err := snapshotResultRows(rows, fetchedAt)
	if err != nil {
		return err
	}
	if len(resultRows) > 1000 {
		return nil
	}

	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(
			ctx,
			"DELETE FROM query_results WHERE query_id = $1 AND host_id = $2",
			queryID,
			hostID,
		); err != nil {
			return err
		}
		for _, row := range resultRows {
			var data any
			if row.data != nil {
				data = []byte(*row.data)
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO query_results (query_id, host_id, data, last_fetched)
				 VALUES ($1, $2, $3::jsonb, $4)`,
				queryID,
				hostID,
				data,
				row.lastFetched,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

// Results returns stored report rows for one query.
func (s *QueryStore) Results(ctx context.Context, queryID int64) ([]QueryResult, error) {
	rows, err := s.db.Pool().Query(ctx,
		`SELECT r.query_id, q.name, r.host_id, h.display_name, r.data, r.last_fetched
		 FROM query_results r
		 JOIN queries q ON q.id = r.query_id
		 JOIN hosts h ON h.id = r.host_id
		 WHERE r.query_id = $1 AND r.data IS NOT NULL
		 ORDER BY r.last_fetched DESC, r.host_id, r.id`,
		queryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]QueryResult, 0)
	for rows.Next() {
		result, err := scanQueryResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

// HostReports returns scheduled reports and their latest host-specific result.
func (s *QueryStore) HostReports(ctx context.Context, host hosts.Host) ([]HostReport, error) {
	queries, err := s.ScheduledForHost(ctx, host)
	if err != nil {
		return nil, err
	}
	results := make([]HostReport, 0, len(queries))
	for _, query := range queries {
		report := HostReport{
			ReportID:    query.ID,
			Name:        query.Name,
			Description: query.Description,
		}
		if err := s.loadHostReportState(ctx, query.ID, host.ID, &report); err != nil {
			return nil, err
		}
		results = append(results, report)
	}
	return results, nil
}

// HostQueryResults returns all stored rows for one host and report.
func (s *QueryStore) HostQueryResults(
	ctx context.Context,
	hostID int64,
	queryID int64,
) ([]QueryResult, *time.Time, error) {
	rows, err := s.db.Pool().Query(ctx,
		`SELECT r.query_id, q.name, r.host_id, h.display_name, r.data, r.last_fetched
		 FROM query_results r
		 JOIN queries q ON q.id = r.query_id
		 JOIN hosts h ON h.id = r.host_id
		 WHERE r.query_id = $1 AND r.host_id = $2
		 ORDER BY r.last_fetched DESC, r.id`,
		queryID,
		hostID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	results := make([]QueryResult, 0)
	var lastFetched *time.Time
	for rows.Next() {
		result, hasData, err := scanQueryResultRow(rows)
		if err != nil {
			return nil, nil, err
		}
		if lastFetched == nil {
			fetched := result.LastFetched
			lastFetched = &fetched
		}
		if hasData {
			results = append(results, result)
		}
	}
	return results, lastFetched, rows.Err()
}

// TrimResults keeps the newest maxRows scheduled-query result rows per query.
func (s *QueryStore) TrimResults(ctx context.Context, maxRows int) error {
	if maxRows <= 0 {
		return nil
	}
	_, err := s.db.Pool().Exec(ctx,
		`DELETE FROM query_results r
		 USING (
		     SELECT id
		     FROM (
		         SELECT r.id,
		                row_number() OVER (PARTITION BY r.query_id ORDER BY r.last_fetched DESC, r.id DESC) AS rn
		         FROM query_results r
		         JOIN queries q ON q.id = r.query_id
		         WHERE q.schedule_interval > 0 AND r.data IS NOT NULL
		     ) ranked
		     WHERE rn > $1
		     LIMIT 500
		 ) doomed
		 WHERE r.id = doomed.id`,
		maxRows,
	)
	return err
}

func cleanQueryCreate(params QueryCreate) (QueryCreate, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = strings.TrimSpace(params.Query)
	params.Platform = cleanPlatformPtr(params.Platform)
	params.MinOsqueryVersion = cleanStringPtr(params.MinOsqueryVersion)
	if params.LoggingType == "" {
		params.LoggingType = QueryLoggingSnapshot
	}
	params.LabelScope = hosts.NormalizeLabelScope(params.LabelScope)
	if params.Name == "" {
		return QueryCreate{}, fmt.Errorf("%w: name is required", store.ErrInvalidInput)
	}
	if params.Query == "" {
		return QueryCreate{}, fmt.Errorf("%w: query is required", store.ErrInvalidInput)
	}
	if params.ScheduleInterval < 0 {
		return QueryCreate{}, fmt.Errorf("%w: schedule interval cannot be negative", store.ErrInvalidInput)
	}
	if params.LoggingType != QueryLoggingSnapshot {
		return QueryCreate{}, fmt.Errorf("%w: logging type must be snapshot", store.ErrInvalidInput)
	}
	return params, nil
}

func cleanQueryListParams(params QueryListParams) QueryListParams {
	params.ListParams = store.CleanListParams(params.ListParams)
	params.Platform = platform.CleanPlatform(params.Platform)
	return params
}

func snapshotResultRows(rows []map[string]string, fetchedAt time.Time) ([]snapshotResultRow, error) {
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	if len(rows) == 0 {
		return []snapshotResultRow{{lastFetched: fetchedAt}}, nil
	}

	out := make([]snapshotResultRow, 0, len(rows))
	for _, columns := range rows {
		data, err := json.Marshal(columns)
		if err != nil {
			return nil, err
		}
		raw := json.RawMessage(data)
		out = append(out, snapshotResultRow{data: &raw, lastFetched: fetchedAt})
	}
	return out, nil
}

func scanQuery(row pgx.Row) (*Query, error) {
	var query Query
	var loggingType string
	err := row.Scan(
		&query.ID,
		&query.Name,
		&query.Description,
		&query.Query,
		&query.Platform,
		&query.MinOsqueryVersion,
		&query.ScheduleInterval,
		&loggingType,
		&query.CreatedByUserID,
		&query.CreatedAt,
		&query.UpdatedAt,
	)
	query.LoggingType = QueryLoggingType(loggingType)
	return &query, err
}

func scanQueryResult(row pgx.Row) (QueryResult, error) {
	result, _, err := scanQueryResultRow(row)
	return result, err
}

func scanQueryResultRow(row pgx.Row) (QueryResult, bool, error) {
	var result QueryResult
	var data []byte
	err := row.Scan(
		&result.QueryID,
		&result.QueryName,
		&result.HostID,
		&result.HostName,
		&data,
		&result.LastFetched,
	)
	if err != nil {
		return QueryResult{}, false, err
	}
	if data == nil {
		return result, false, nil
	}
	if err := json.Unmarshal(data, &result.Columns); err != nil {
		return QueryResult{}, false, err
	}
	return result, true, nil
}

func (s *QueryStore) loadHostReportState(ctx context.Context, queryID int64, hostID int64, report *HostReport) error {
	var fetched time.Time
	err := s.db.Pool().QueryRow(ctx,
		`SELECT last_fetched
		 FROM query_results
		 WHERE query_id = $1 AND host_id = $2
		 ORDER BY last_fetched DESC, id DESC
		 LIMIT 1`,
		queryID,
		hostID,
	).Scan(&fetched)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil {
		report.LastFetched = &fetched
	}

	if err := s.db.Pool().QueryRow(ctx,
		`SELECT count(*)
		 FROM query_results
		 WHERE query_id = $1 AND host_id = $2 AND data IS NOT NULL`,
		queryID,
		hostID,
	).Scan(&report.HostResultCount); err != nil {
		return err
	}

	var data []byte
	err = s.db.Pool().QueryRow(ctx,
		`SELECT data
		 FROM query_results
		 WHERE query_id = $1 AND host_id = $2 AND data IS NOT NULL
		 ORDER BY last_fetched DESC, id DESC
		 LIMIT 1`,
		queryID,
		hostID,
	).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &report.FirstResult)
}

func queryListWhere(params QueryListParams) (string, []any) {
	return store.NameSearchAndPlatformWhere(params.Q, params.Platform)
}

func queryListSQL(where string, args []any, params QueryListParams) (string, []any, error) {
	return store.ListQuery{
		SelectSQL: querySelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]store.OrderExpr{
			"name":               {SQL: "name"},
			"created_at":         {SQL: "created_at"},
			store.OrderUpdatedAt: {SQL: store.OrderUpdatedAt},
			"schedule_interval":  {SQL: "schedule_interval"},
		},
		DefaultOrder: []store.OrderExpr{{SQL: store.OrderUpdatedAt}, {SQL: "id"}},
		Params:       params.ListParams,
	}.Build()
}

const querySelectSQL = `
SELECT id, name, description, query, platform, min_osquery_version, schedule_interval,
       logging_type, created_by_user_id, created_at, updated_at
FROM queries`

const queryInsertSQL = `
INSERT INTO queries (
    name, description, query, platform, min_osquery_version, schedule_interval,
    logging_type, created_by_user_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, name, description, query, platform, min_osquery_version, schedule_interval,
          logging_type, created_by_user_id, created_at, updated_at`

const queryUpdateSQL = `
UPDATE queries
SET name = $1,
    description = $2,
    query = $3,
    platform = $4,
    min_osquery_version = $5,
    schedule_interval = $6,
    logging_type = $7,
    updated_at = now()
WHERE id = $8
RETURNING id, name, description, query, platform, min_osquery_version, schedule_interval,
          logging_type, created_by_user_id, created_at, updated_at`

func (s *QueryStore) loadScope(
	ctx context.Context,
	table string,
	ownerColumn string,
	ownerID int64,
) (hosts.LabelScope, error) {
	rows, err := s.db.Pool().Query(ctx,
		fmt.Sprintf("SELECT label_id, exclude, require_all FROM %s WHERE %s = $1 ORDER BY label_id", table, ownerColumn),
		ownerID,
	)
	if err != nil {
		return hosts.LabelScope{}, err
	}
	defer rows.Close()
	return scanScopeRows(rows)
}

func scanScopeRows(rows pgx.Rows) (hosts.LabelScope, error) {
	scope := hosts.LabelScope{Mode: hosts.ScopeNone}
	for rows.Next() {
		var labelID int64
		var exclude bool
		var requireAll bool
		if err := rows.Scan(&labelID, &exclude, &requireAll); err != nil {
			return hosts.LabelScope{}, err
		}
		scope.LabelIDs = append(scope.LabelIDs, labelID)
		switch {
		case exclude:
			scope.Mode = hosts.ScopeExcludeAny
		case requireAll && scope.Mode != hosts.ScopeExcludeAny:
			scope.Mode = hosts.ScopeIncludeAll
		case scope.Mode == hosts.ScopeNone:
			scope.Mode = hosts.ScopeIncludeAny
		}
	}
	if err := rows.Err(); err != nil {
		return hosts.LabelScope{}, err
	}
	return hosts.NormalizeLabelScope(scope), nil
}

func replaceScope(
	ctx context.Context,
	tx pgx.Tx,
	table string,
	ownerColumn string,
	ownerID int64,
	scope hosts.LabelScope,
) error {
	scope = hosts.NormalizeLabelScope(scope)
	if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = $1", table, ownerColumn), ownerID); err != nil {
		return err
	}
	exclude := scope.Mode == hosts.ScopeExcludeAny
	requireAll := scope.Mode == hosts.ScopeIncludeAll
	for _, labelID := range scope.LabelIDs {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf(
				"INSERT INTO %s (%s, label_id, exclude, require_all) VALUES ($1, $2, $3, $4)",
				table,
				ownerColumn,
			),
			ownerID,
			labelID,
			exclude,
			requireAll,
		); err != nil {
			return err
		}
	}
	return nil
}
