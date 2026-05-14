package queries

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/hosts"
)

const maxSnapshotResultRows = 1000

// trimBatchSize caps how many rows are deleted per TrimResults call; unrelated
// to the per-query retention limit maxRows.
const trimBatchSize = 500

// ErrSnapshotTooLarge is returned by OverwriteResults when the incoming
// snapshot exceeds maxSnapshotResultRows.
var ErrSnapshotTooLarge = errors.New("snapshot exceeds max result rows")

type snapshotResultRow struct {
	data        *json.RawMessage
	lastFetched time.Time
}

// OverwriteResults replaces the snapshot rows for a query on one host.
func (s *Store) OverwriteResults(
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
	if len(resultRows) > maxSnapshotResultRows {
		return fmt.Errorf("%w: got %d rows (max %d)", ErrSnapshotTooLarge, len(resultRows), maxSnapshotResultRows)
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
func (s *Store) Results(ctx context.Context, queryID int64) ([]QueryResult, error) {
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
func (s *Store) HostReports(ctx context.Context, host *hosts.Host) ([]HostReport, error) {
	queries, err := s.ScheduledForHost(ctx, host)
	if err != nil {
		return nil, err
	}
	queryIDs := make([]int64, 0, len(queries))
	for _, query := range queries {
		queryIDs = append(queryIDs, query.ID)
	}
	states, err := s.loadHostReportStates(ctx, host.ID, queryIDs)
	if err != nil {
		return nil, err
	}
	results := make([]HostReport, 0, len(queries))
	for _, query := range queries {
		report := HostReport{
			ReportID:        query.ID,
			Name:            query.Name,
			Description:     query.Description,
			LastFetched:     states[query.ID].lastFetched,
			FirstResult:     states[query.ID].firstResult,
			HostResultCount: states[query.ID].hostResultCount,
		}
		results = append(results, report)
	}
	return results, nil
}

// HostQueryResults returns all stored rows for one host and report.
func (s *Store) HostQueryResults(
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
			lastFetched = new(result.LastFetched)
		}
		if hasData {
			results = append(results, result)
		}
	}
	return results, lastFetched, rows.Err()
}

// TrimResults keeps the newest maxRows scheduled-query result rows per query.
func (s *Store) TrimResults(ctx context.Context, maxRows int) error {
	if maxRows <= 0 {
		return nil
	}
	_, err := s.db.Pool().Exec(ctx,
		fmt.Sprintf(`DELETE FROM query_results r
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
		     LIMIT %d
		 ) doomed
		 WHERE r.id = doomed.id`, trimBatchSize),
		maxRows,
	)
	return err
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

type hostReportState struct {
	lastFetched     *time.Time
	firstResult     map[string]string
	hostResultCount int
}

func (s *Store) loadHostReportStates(
	ctx context.Context,
	hostID int64,
	queryIDs []int64,
) (map[int64]hostReportState, error) {
	states := make(map[int64]hostReportState, len(queryIDs))
	if len(queryIDs) == 0 {
		return states, nil
	}

	rows, err := s.db.Pool().Query(ctx,
		`WITH requested AS (
		     SELECT unnest($1::bigint[]) AS query_id
		 ),
		 latest_fetch AS (
		     SELECT DISTINCT ON (query_id) query_id, last_fetched
		     FROM query_results
		     WHERE host_id = $2 AND query_id = ANY($1::bigint[])
		     ORDER BY query_id, last_fetched DESC, id DESC
		 ),
		 result_counts AS (
		     SELECT query_id, count(*)::integer AS host_result_count
		     FROM query_results
		     WHERE host_id = $2 AND query_id = ANY($1::bigint[]) AND data IS NOT NULL
		     GROUP BY query_id
		 ),
		 latest_data AS (
		     SELECT DISTINCT ON (query_id) query_id, data
		     FROM query_results
		     WHERE host_id = $2 AND query_id = ANY($1::bigint[]) AND data IS NOT NULL
		     ORDER BY query_id, last_fetched DESC, id DESC
		 )
		 SELECT r.query_id, lf.last_fetched, coalesce(rc.host_result_count, 0), ld.data
		 FROM requested r
		 LEFT JOIN latest_fetch lf ON lf.query_id = r.query_id
		 LEFT JOIN result_counts rc ON rc.query_id = r.query_id
		 LEFT JOIN latest_data ld ON ld.query_id = r.query_id`,
		queryIDs,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var queryID int64
		var state hostReportState
		var data []byte
		if err := rows.Scan(&queryID, &state.lastFetched, &state.hostResultCount, &data); err != nil {
			return nil, err
		}
		if data != nil {
			if err := json.Unmarshal(data, &state.firstResult); err != nil {
				return nil, err
			}
		}
		states[queryID] = state
	}
	return states, rows.Err()
}
