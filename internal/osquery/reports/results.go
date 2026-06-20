package reports

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

// ErrSnapshotTooLarge is returned by OverwriteResults when the incoming
// snapshot exceeds maxSnapshotResultRows.
var ErrSnapshotTooLarge = errors.New("snapshot exceeds max result rows")

type snapshotResultRow struct {
	data        *json.RawMessage
	lastFetched time.Time
}

// OverwriteResults replaces the snapshot rows for a report on one host.
func (s *Store) OverwriteResults(
	ctx context.Context,
	reportID int64,
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
		if _, err := tx.Exec(ctx,
			`DELETE FROM report_results WHERE report_id = $1 AND host_id = $2`,
			reportID, hostID,
		); err != nil {
			return err
		}
		if len(resultRows) == 0 {
			return nil
		}
		_, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{"report_results"},
			[]string{"report_id", "host_id", "data", "last_fetched"},
			pgx.CopyFromRows(copyFromSnapshotRows(reportID, hostID, resultRows)),
		)
		return err
	})
}

// Results returns stored snapshot rows for one report.
func (s *Store) Results(ctx context.Context, reportID int64) ([]ReportResult, error) {
	type resultRow struct {
		ReportID    int64     `db:"report_id"`
		Name        string    `db:"name"`
		HostID      int64     `db:"host_id"`
		DisplayName string    `db:"display_name"`
		Data        []byte    `db:"data"`
		LastFetched time.Time `db:"last_fetched"`
	}
	qrows, err := s.db.Pool().Query(ctx, `
		SELECT rr.report_id, r.name, rr.host_id, h.display_name, rr.data, rr.last_fetched
		FROM report_results rr
		JOIN reports r ON r.id = rr.report_id
		JOIN hosts h ON h.id = rr.host_id
		WHERE rr.report_id = $1 AND rr.data IS NOT NULL
		ORDER BY rr.last_fetched DESC, rr.host_id, rr.id`, reportID)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[resultRow])
	if err != nil {
		return nil, err
	}

	results := make([]ReportResult, 0, len(rows))
	for _, row := range rows {
		result, hasData, err := reportResultFromNullableFields(
			row.ReportID,
			row.Name,
			row.HostID,
			row.DisplayName,
			row.Data,
			row.LastFetched,
		)
		if err != nil {
			return nil, err
		}
		if hasData {
			results = append(results, result)
		}
	}
	return results, nil
}

// HostReports returns scheduled reports and their latest host-specific result.
func (s *Store) HostReports(ctx context.Context, host *hosts.Host) ([]HostReport, error) {
	rpts, err := s.ScheduledForHost(ctx, host)
	if err != nil {
		return nil, err
	}
	reportIDs := make([]int64, 0, len(rpts))
	for _, rpt := range rpts {
		reportIDs = append(reportIDs, rpt.ID)
	}
	states, err := s.loadHostReportStates(ctx, host.ID, reportIDs)
	if err != nil {
		return nil, err
	}
	results := make([]HostReport, 0, len(rpts))
	for _, rpt := range rpts {
		results = append(results, HostReport{
			ReportID:        rpt.ID,
			Name:            rpt.Name,
			Description:     rpt.Description,
			LastFetched:     states[rpt.ID].lastFetched,
			FirstResult:     states[rpt.ID].firstResult,
			HostResultCount: states[rpt.ID].hostResultCount,
		})
	}
	return results, nil
}

// HostResults returns all stored rows for one host and report.
func (s *Store) HostResults(
	ctx context.Context,
	hostID int64,
	reportID int64,
) ([]ReportResult, *time.Time, error) {
	type resultRow struct {
		ReportID    int64     `db:"report_id"`
		Name        string    `db:"name"`
		HostID      int64     `db:"host_id"`
		DisplayName string    `db:"display_name"`
		Data        []byte    `db:"data"`
		LastFetched time.Time `db:"last_fetched"`
	}
	qrows, err := s.db.Pool().Query(ctx, `
		SELECT rr.report_id, r.name, rr.host_id, h.display_name, rr.data, rr.last_fetched
		FROM report_results rr
		JOIN reports r ON r.id = rr.report_id
		JOIN hosts h ON h.id = rr.host_id
		WHERE rr.report_id = $1 AND rr.host_id = $2
		ORDER BY rr.last_fetched DESC, rr.id`, reportID, hostID)
	if err != nil {
		return nil, nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[resultRow])
	if err != nil {
		return nil, nil, err
	}

	results := make([]ReportResult, 0, len(rows))
	var lastFetched *time.Time
	for _, row := range rows {
		result, hasData, err := reportResultFromNullableFields(
			row.ReportID,
			row.Name,
			row.HostID,
			row.DisplayName,
			row.Data,
			row.LastFetched,
		)
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
	return results, lastFetched, nil
}

func copyFromSnapshotRows(reportID int64, hostID int64, rows []snapshotResultRow) [][]any {
	out := make([][]any, 0, len(rows))
	for _, row := range rows {
		var data any
		if row.data != nil {
			data = []byte(*row.data)
		}
		out = append(out, []any{reportID, hostID, data, row.lastFetched})
	}
	return out
}

func snapshotResultRows(rows []map[string]string, fetchedAt time.Time) ([]snapshotResultRow, error) {
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

func reportResultFromNullableFields(
	reportID int64,
	reportName string,
	hostID int64,
	hostName string,
	data []byte,
	lastFetched time.Time,
) (ReportResult, bool, error) {
	result := ReportResult{
		ReportID:    reportID,
		ReportName:  reportName,
		HostID:      hostID,
		HostName:    hostName,
		LastFetched: lastFetched,
	}
	if data == nil {
		return result, false, nil
	}
	if err := json.Unmarshal(data, &result.Columns); err != nil {
		return ReportResult{}, false, err
	}
	return result, true, nil
}

type hostReportState struct {
	lastFetched     *time.Time
	firstResult     map[string]string
	hostResultCount int32
}

func (s *Store) loadHostReportStates(
	ctx context.Context,
	hostID int64,
	reportIDs []int64,
) (map[int64]hostReportState, error) {
	states := make(map[int64]hostReportState, len(reportIDs))
	if len(reportIDs) == 0 {
		return states, nil
	}

	type stateRow struct {
		ReportID        int64      `db:"report_id"`
		LastFetched     *time.Time `db:"last_fetched"`
		HostResultCount int32      `db:"host_result_count"`
		Data            []byte     `db:"data"`
	}
	qrows, err := s.db.Pool().Query(ctx, `
		WITH requested AS (
		    SELECT unnest($1::bigint[])::bigint AS report_id
		),
		latest_fetch AS (
		    SELECT DISTINCT ON (report_id) report_id, last_fetched
		    FROM report_results rr
		    WHERE rr.host_id = $2 AND rr.report_id = ANY($1::bigint[])
		    ORDER BY report_id, last_fetched DESC, id DESC
		),
		result_counts AS (
		    SELECT report_id, count(*)::integer AS host_result_count
		    FROM report_results rr
		    WHERE rr.host_id = $2 AND rr.report_id = ANY($1::bigint[]) AND rr.data IS NOT NULL
		    GROUP BY report_id
		),
		latest_data AS (
		    SELECT DISTINCT ON (report_id) report_id, data
		    FROM report_results rr
		    WHERE rr.host_id = $2 AND rr.report_id = ANY($1::bigint[]) AND rr.data IS NOT NULL
		    ORDER BY report_id, last_fetched DESC, id DESC
		)
		SELECT
		    req.report_id,
		    lf.last_fetched,
		    coalesce(rc.host_result_count, 0)::integer AS host_result_count,
		    ld.data
		FROM requested req
		LEFT JOIN latest_fetch lf ON lf.report_id = req.report_id
		LEFT JOIN result_counts rc ON rc.report_id = req.report_id
		LEFT JOIN latest_data ld ON ld.report_id = req.report_id`,
		reportIDs, hostID,
	)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[stateRow])
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		var state hostReportState
		state.lastFetched = row.LastFetched
		state.hostResultCount = row.HostResultCount
		if row.Data != nil {
			if err := json.Unmarshal(row.Data, &state.firstResult); err != nil {
				return nil, err
			}
		}
		states[row.ReportID] = state
	}
	return states, nil
}
