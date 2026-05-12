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

func (s *Store) loadHostReportState(ctx context.Context, queryID int64, hostID int64, report *HostReport) error {
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
