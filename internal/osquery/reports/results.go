package reports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// OverwriteSnapshot replaces a host's latest snapshot when the report query
// and assignment are still current. Older observations never replace newer
// state.
func (s *Store) OverwriteSnapshot(
	ctx context.Context,
	reportID int64,
	queryHash string,
	hostID int64,
	rows []map[string]string,
	collectedAt time.Time,
) error {
	if collectedAt.IsZero() {
		collectedAt = time.Now().UTC()
	}
	snapshotRows, err := json.Marshal(normalizeSnapshotRows(rows))
	if err != nil {
		return err
	}

	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
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
				collected_at
			) VALUES ($1, $2, $3, $4)
			ON CONFLICT (report_id, host_id) DO UPDATE
			SET
				rows = EXCLUDED.rows,
				collected_at = EXCLUDED.collected_at
			WHERE EXCLUDED.collected_at >= osquery_report_snapshots.collected_at`,
			reportID,
			hostID,
			snapshotRows,
			collectedAt,
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
		if err := s.db.Pool().QueryRow(
			ctx,
			`SELECT EXISTS (SELECT 1 FROM osquery_reports WHERE id = $1)`,
			reportID,
		).Scan(&exists); err != nil {
			return nil, 0, err
		}
		if !exists {
			return nil, 0, dbutil.ErrNotFound
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
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	if err := validateReportSnapshotStatusFilter(params.Status); err != nil {
		return nil, 0, err
	}

	var where dbutil.WhereBuilder
	where.Add(scopeSQL + " = " + where.Arg(scopeID))
	switch params.Status {
	case ReportSnapshotStatusCollected:
		where.Add("snapshot.collected_at IS NOT NULL")
	case ReportSnapshotStatusPending:
		where.Add("snapshot.collected_at IS NULL")
	}

	searchPattern := ""
	if params.Q != "" {
		searchPattern = where.Arg("%" + params.Q + "%")
		where.Add(
			"(" + parentSearchSQL + " ILIKE " + searchPattern +
				" OR matched_rows.returned_row_count > 0)",
		)
	}
	whereSQL, args := where.Build()
	listQuery := dbutil.ListQuery{
		SelectSQL: snapshotSelectSQL(parentSearchSQL, searchPattern),
		WhereSQL:  whereSQL,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			nameSortKey:        {SQL: nameOrderSQL},
			"status":           {SQL: snapshotStatusOrderSQL()},
			"collected_at":     {SQL: "snapshot.collected_at", NullOrder: dbutil.NullsLast},
			"result_row_count": {SQL: snapshotResultRowCountSQL()},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: nameOrderSQL}, {SQL: stableOrderSQL}},
		Params:       params.ListParams,
	}
	rows, count, err := dbutil.ListWithCount[snapshotRow](ctx, s.db.Pool(), listQuery)
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
				THEN COALESCE(snapshot.rows, '[]'::jsonb)
			ELSE matched_rows.rows
		END`
		returnedRowCountSQL = `CASE
			WHEN ` + parentSearchSQL + ` ILIKE ` + searchPattern + `
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
			snapshot.collected_at
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
	return `CASE WHEN snapshot.collected_at IS NOT NULL THEN 0 ELSE 1 END`
}

type snapshotRow struct {
	ReportID          int64      `db:"report_id"`
	ReportName        string     `db:"report_name"`
	ReportDescription string     `db:"report_description"`
	HostID            int64      `db:"host_id"`
	HostName          string     `db:"host_name"`
	Rows              []byte     `db:"rows"`
	ResultRowCount    int32      `db:"result_row_count"`
	ReturnedRowCount  int32      `db:"returned_row_count"`
	CollectedAt       *time.Time `db:"collected_at"`
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
		Status:            reportSnapshotStatus(row.CollectedAt),
		ResultRowCount:    row.ResultRowCount,
		ReturnedRowCount:  row.ReturnedRowCount,
		Rows:              rows,
		CollectedAt:       row.CollectedAt,
	}, nil
}

func reportSnapshotStatus(collectedAt *time.Time) ReportSnapshotStatus {
	if collectedAt == nil {
		return ReportSnapshotStatusPending
	}
	return ReportSnapshotStatusCollected
}

func validateReportSnapshotStatusFilter(status ReportSnapshotStatus) error {
	switch status {
	case "", ReportSnapshotStatusCollected, ReportSnapshotStatusPending:
		return nil
	default:
		return dbutil.ErrInvalidInput
	}
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
