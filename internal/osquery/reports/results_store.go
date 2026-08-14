package reports

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

const maxReportErrorRunes = 4096

// OverwriteSnapshot replaces a host's latest snapshot when the report query
// and assignment are still current. Older observations never replace newer
// state.
func (s *Store) OverwriteSnapshot(
	ctx context.Context,
	reportID int64,
	queryHash string,
	hostID int64,
	rows []map[string]string,
	reportedAt time.Time,
) error {
	snapshotRows, err := json.Marshal(normalizeSnapshotRows(rows))
	if err != nil {
		return err
	}
	return s.overwriteObservation(
		ctx,
		reportID,
		queryHash,
		hostID,
		ReportSnapshotStatusCollected,
		snapshotRows,
		"",
		reportedAt,
	)
}

// OverwriteError replaces a host's latest snapshot with a scheduled-query
// error when the report query and assignment are still current.
func (s *Store) OverwriteError(
	ctx context.Context,
	reportID int64,
	queryHash string,
	hostID int64,
	reportError string,
	reportedAt time.Time,
) error {
	reportError = strings.TrimSpace(reportError)
	if reportError == "" {
		return fault.ErrInvalidInput
	}
	return s.overwriteObservation(
		ctx,
		reportID,
		queryHash,
		hostID,
		ReportSnapshotStatusError,
		[]byte("[]"),
		truncateRunes(reportError, maxReportErrorRunes),
		reportedAt,
	)
}

func (s *Store) overwriteObservation(
	ctx context.Context,
	reportID int64,
	queryHash string,
	hostID int64,
	status ReportSnapshotStatus,
	rows []byte,
	reportError string,
	reportedAt time.Time,
) error {
	if reportedAt.IsZero() {
		reportedAt = time.Now().UTC()
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var accepted bool
		err := tx.QueryRow(ctx, `
			SELECT true
			FROM osquery_reports report_row
			JOIN osquery_report_assignments assignment
				ON assignment.report_id = report_row.id
			   AND assignment.host_id = $3
			WHERE report_row.id = $1
			  AND encode(sha256(convert_to(report_row.query, 'UTF8')), 'hex') = $2
			FOR UPDATE OF report_row`,
			reportID,
			queryHash,
			hostID,
		).Scan(&accepted)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO osquery_report_snapshots (
				report_id,
				host_id,
				rows,
				status,
				error,
				reported_at
			) VALUES ($1, $2, $3, $4::osquery_report_snapshot_status, $5, $6)
			ON CONFLICT (report_id, host_id) DO UPDATE
			SET
				rows = EXCLUDED.rows,
				status = EXCLUDED.status,
				error = EXCLUDED.error,
				reported_at = EXCLUDED.reported_at
			WHERE EXCLUDED.reported_at >= osquery_report_snapshots.reported_at`,
			reportID,
			hostID,
			rows,
			status,
			reportError,
			reportedAt,
		)
		return err
	})
}

// Snapshots returns a page of currently targeted hosts and their latest
// observation for one report.
func (s *Store) Snapshots(
	ctx context.Context,
	reportID int64,
	params ReportSnapshotListParams,
) ([]ReportSnapshot, int, error) {
	rows, count, err := s.listSnapshots(
		ctx,
		params,
		"report_row.id",
		reportID,
		"host_row.display_name",
		"host_name",
		"lower(host_row.display_name)",
		"host_row.id",
	)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		var exists bool
		if err := s.pool.QueryRow(
			ctx,
			`SELECT EXISTS (SELECT 1 FROM osquery_reports WHERE id = $1)`,
			reportID,
		).Scan(&exists); err != nil {
			return nil, 0, err
		}
		if !exists {
			return nil, 0, fault.ErrNotFound
		}
	}
	return rows, count, nil
}

// HostSnapshots returns a page of reports assigned to a host with each report's
// latest observation, whether or not the report is currently scheduled.
func (s *Store) HostSnapshots(
	ctx context.Context,
	host *hosts.Host,
	params ReportSnapshotListParams,
) ([]ReportSnapshot, int, error) {
	return s.listSnapshots(
		ctx,
		params,
		"host_row.id",
		host.ID,
		`CONCAT_WS(E'\n', report_row.name, report_row.description)`,
		"report_name",
		"lower(report_row.name)",
		"report_row.id",
	)
}

func (s *Store) listSnapshots(
	ctx context.Context,
	params ReportSnapshotListParams,
	scopeSQL string,
	scopeID int64,
	parentSearchSQL string,
	nameSortKey string,
	nameOrderSQL string,
	stableOrderSQL string,
) ([]ReportSnapshot, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
	if err := validateReportSnapshotStatusFilter(params.Status); err != nil {
		return nil, 0, err
	}

	var where postgres.WhereBuilder
	where.Add(scopeSQL + " = " + where.Arg(scopeID))
	switch params.Status {
	case ReportSnapshotStatusCollected:
		where.Add("snapshot.status = 'collected'")
	case ReportSnapshotStatusError:
		where.Add("snapshot.status = 'error'")
	case ReportSnapshotStatusPending:
		where.Add("snapshot.report_id IS NULL")
	}

	searchPattern := ""
	if params.ListParams.Q != "" {
		searchPattern = where.Arg("%" + params.ListParams.Q + "%")
		where.Add(
			"(" + parentSearchSQL + " ILIKE " + searchPattern +
				" OR snapshot.error ILIKE " + searchPattern +
				" OR matched_rows.returned_row_count > 0)",
		)
	}
	whereSQL, args := where.Build()
	listQuery := postgres.ListQuery{
		SelectSQL: snapshotSelectSQL(parentSearchSQL, searchPattern),
		WhereSQL:  whereSQL,
		Args:      args,
		OrderKeys: map[string]postgres.OrderExpr{
			nameSortKey:        {SQL: nameOrderSQL},
			"status":           {SQL: snapshotStatusOrderSQL()},
			"reported_at":      {SQL: "snapshot.reported_at", NullOrder: postgres.NullsLast},
			"result_row_count": {SQL: snapshotResultRowCountSQL()},
		},
		DefaultOrder: []postgres.OrderExpr{{SQL: nameOrderSQL}, {SQL: stableOrderSQL}},
		Params:       params.ListParams,
	}
	rows, count, err := postgres.ListWithCount[snapshotRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}

	snapshots := make([]ReportSnapshot, 0, len(rows))
	for _, row := range rows {
		snapshot, err := snapshotFromRow(row)
		if err != nil {
			return nil, 0, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, count, nil
}

func snapshotSelectSQL(parentSearchSQL, searchPattern string) string {
	rowsSQL := `COALESCE(snapshot.rows, '[]'::jsonb)`
	returnedRowCountSQL := snapshotResultRowCountSQL()
	lateralSQL := ""
	if searchPattern != "" {
		rowsSQL = `CASE
			WHEN ` + parentSearchSQL + ` ILIKE ` + searchPattern + `
				OR snapshot.error ILIKE ` + searchPattern + `
				THEN COALESCE(snapshot.rows, '[]'::jsonb)
			ELSE matched_rows.rows
		END`
		returnedRowCountSQL = `CASE
			WHEN ` + parentSearchSQL + ` ILIKE ` + searchPattern + `
				OR snapshot.error ILIKE ` + searchPattern + `
				THEN ` + snapshotResultRowCountSQL() + `
			ELSE matched_rows.returned_row_count
		END`
		lateralSQL = `
		LEFT JOIN LATERAL (
			SELECT
				COALESCE(
					jsonb_agg(result_row.value ORDER BY result_row.ordinality),
					'[]'::jsonb
				) AS rows,
				count(*)::integer AS returned_row_count
			FROM jsonb_array_elements(
				COALESCE(snapshot.rows, '[]'::jsonb)
			) WITH ORDINALITY AS result_row(value, ordinality)
			WHERE result_row.value::text ILIKE ` + searchPattern + `
		) matched_rows ON true`
	}

	return `
		SELECT
			report_row.id AS report_id,
			report_row.name AS report_name,
			report_row.description AS report_description,
			host_row.id AS host_id,
			host_row.display_name AS host_name,
			` + rowsSQL + ` AS rows,
			` + snapshotResultRowCountSQL() + ` AS result_row_count,
			` + returnedRowCountSQL + ` AS returned_row_count,
			COALESCE(snapshot.status::text, 'pending') AS status,
			COALESCE(snapshot.error, '') AS error,
			snapshot.reported_at
		FROM osquery_reports report_row
		JOIN osquery_report_assignments assignment
			ON assignment.report_id = report_row.id
		JOIN hosts host_row
			ON host_row.id = assignment.host_id
		LEFT JOIN osquery_report_snapshots snapshot
			ON snapshot.report_id = report_row.id
		   AND snapshot.host_id = host_row.id` + lateralSQL
}

func snapshotResultRowCountSQL() string {
	return `jsonb_array_length(COALESCE(snapshot.rows, '[]'::jsonb))`
}

func snapshotStatusOrderSQL() string {
	return `CASE
		WHEN snapshot.status = 'error' THEN 0
		WHEN snapshot.report_id IS NULL THEN 1
		ELSE 2
	END`
}

type snapshotRow struct {
	ReportID          int64                `db:"report_id"`
	ReportName        string               `db:"report_name"`
	ReportDescription string               `db:"report_description"`
	HostID            int64                `db:"host_id"`
	HostName          string               `db:"host_name"`
	Rows              []byte               `db:"rows"`
	ResultRowCount    int32                `db:"result_row_count"`
	ReturnedRowCount  int32                `db:"returned_row_count"`
	Status            ReportSnapshotStatus `db:"status"`
	Error             string               `db:"error"`
	ReportedAt        *time.Time           `db:"reported_at"`
}

func snapshotFromRow(row snapshotRow) (ReportSnapshot, error) {
	rows, err := decodeSnapshotRows(row.Rows)
	if err != nil {
		return ReportSnapshot{}, err
	}
	return ReportSnapshot{
		ReportID:          row.ReportID,
		ReportName:        row.ReportName,
		ReportDescription: row.ReportDescription,
		HostID:            row.HostID,
		HostName:          row.HostName,
		Status:            row.Status,
		ResultRowCount:    row.ResultRowCount,
		ReturnedRowCount:  row.ReturnedRowCount,
		Rows:              rows,
		Error:             row.Error,
		ReportedAt:        row.ReportedAt,
	}, nil
}

func validateReportSnapshotStatusFilter(status ReportSnapshotStatus) error {
	switch status {
	case "", ReportSnapshotStatusCollected, ReportSnapshotStatusError, ReportSnapshotStatusPending:
		return nil
	default:
		return fault.ErrInvalidInput
	}
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[len(runes)-limit:])
}

func normalizeSnapshotRows(rows []map[string]string) []map[string]string {
	if rows == nil {
		return []map[string]string{}
	}
	return rows
}

func decodeSnapshotRows(data []byte) ([]map[string]string, error) {
	if data == nil {
		return []map[string]string{}, nil
	}
	var rows []map[string]string
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	return normalizeSnapshotRows(rows), nil
}
