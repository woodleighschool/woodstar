package reports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
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
		if err := s.q.WithTx(tx).DeleteReportResults(ctx, sqlc.DeleteReportResultsParams{
			ReportID: reportID,
			HostID:   hostID,
		}); err != nil {
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
	rows, err := s.q.ListReportResults(ctx, sqlc.ListReportResultsParams{ReportID: reportID})
	if err != nil {
		return nil, err
	}

	results := make([]ReportResult, 0, len(rows))
	for _, row := range rows {
		result, err := reportResultFromFields(
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
		results = append(results, result)
	}
	return results, nil
}

// HostReports returns scheduled reports and their latest host-specific result.
func (s *Store) HostReports(ctx context.Context, host *hosts.Host) ([]HostReport, error) {
	reports, err := s.ScheduledForHost(ctx, host)
	if err != nil {
		return nil, err
	}
	reportIDs := make([]int64, 0, len(reports))
	for _, report := range reports {
		reportIDs = append(reportIDs, report.ID)
	}
	states, err := s.loadHostReportStates(ctx, host.ID, reportIDs)
	if err != nil {
		return nil, err
	}
	results := make([]HostReport, 0, len(reports))
	for _, report := range reports {
		results = append(results, HostReport{
			ReportID:        report.ID,
			Name:            report.Name,
			Description:     report.Description,
			LastFetched:     states[report.ID].lastFetched,
			FirstResult:     states[report.ID].firstResult,
			HostResultCount: states[report.ID].hostResultCount,
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
	rows, err := s.q.ListHostReportResults(ctx, sqlc.ListHostReportResultsParams{
		ReportID: reportID,
		HostID:   hostID,
	})
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

func reportResultFromFields(
	reportID int64,
	reportName string,
	hostID int64,
	hostName string,
	data []byte,
	lastFetched time.Time,
) (ReportResult, error) {
	result, hasData, err := reportResultFromNullableFields(reportID, reportName, hostID, hostName, data, lastFetched)
	if err != nil {
		return ReportResult{}, err
	}
	if !hasData {
		return ReportResult{}, nil
	}
	return result, nil
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

	rows, err := s.q.ListHostReportStates(ctx, sqlc.ListHostReportStatesParams{
		ReportIds:   reportIDs,
		StateHostID: hostID,
	})
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
