package reports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// Snapshots returns every currently targeted host and its latest observation
// for one report. A nil CollectedAt identifies a host that has not reported.
func (s *Store) Snapshots(
	ctx context.Context,
	reportID int64,
	status ReportSnapshotStatus,
) ([]ReportSnapshot, error) {
	collected, err := collectedFilter(status)
	if err != nil {
		return nil, err
	}
	if _, err := s.GetByID(ctx, reportID); err != nil {
		return nil, err
	}
	qrows, err := s.db.Pool().Query(ctx, `
		SELECT
			report_row.id AS report_id,
			report_row.name AS report_name,
			report_row.description AS report_description,
			host_row.id AS host_id,
			host_row.display_name AS host_name,
			snapshot.rows,
			snapshot.collected_at
		FROM osquery_reports report_row
		JOIN osquery_report_assignments assignment
			ON assignment.report_id = report_row.id
		JOIN hosts host_row
			ON host_row.id = assignment.host_id
		LEFT JOIN osquery_report_snapshots snapshot
			ON snapshot.report_id = report_row.id
		   AND snapshot.host_id = host_row.id
		WHERE report_row.id = $1
		  AND ($2::boolean IS NULL OR (snapshot.collected_at IS NOT NULL) = $2)
		ORDER BY lower(host_row.display_name), host_row.id`,
		reportID,
		collected,
	)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[snapshotRow])
	if err != nil {
		return nil, err
	}

	snapshots := make([]ReportSnapshot, 0, len(rows))
	for _, row := range rows {
		snapshot, err := snapshotFromRow(row)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

// HostSnapshots returns reports assigned to a host with each report's complete
// latest observation, whether or not the report is currently scheduled.
func (s *Store) HostSnapshots(
	ctx context.Context,
	host *hosts.Host,
	status ReportSnapshotStatus,
) ([]ReportSnapshot, error) {
	collected, err := collectedFilter(status)
	if err != nil {
		return nil, err
	}
	qrows, err := s.db.Pool().Query(ctx, `
		SELECT
			report_row.id AS report_id,
			report_row.name AS report_name,
			report_row.description AS report_description,
			host_row.id AS host_id,
			host_row.display_name AS host_name,
			snapshot.rows,
			snapshot.collected_at
		FROM osquery_reports report_row
		JOIN osquery_report_assignments assignment
			ON assignment.report_id = report_row.id
		   AND assignment.host_id = $1
		JOIN hosts host_row
			ON host_row.id = assignment.host_id
		LEFT JOIN osquery_report_snapshots snapshot
			ON snapshot.report_id = report_row.id
		   AND snapshot.host_id = host_row.id
		WHERE $2::boolean IS NULL OR (snapshot.collected_at IS NOT NULL) = $2
		ORDER BY lower(report_row.name), report_row.id`,
		host.ID,
		collected,
	)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[snapshotRow])
	if err != nil {
		return nil, err
	}
	snapshots := make([]ReportSnapshot, 0, len(rows))
	for _, row := range rows {
		snapshot, err := snapshotFromRow(row)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

type snapshotRow struct {
	ReportID          int64      `db:"report_id"`
	ReportName        string     `db:"report_name"`
	ReportDescription string     `db:"report_description"`
	HostID            int64      `db:"host_id"`
	HostName          string     `db:"host_name"`
	Rows              []byte     `db:"rows"`
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

func collectedFilter(status ReportSnapshotStatus) (*bool, error) {
	switch status {
	case "":
		return nil, nil
	case ReportSnapshotStatusCollected:
		value := true
		return &value, nil
	case ReportSnapshotStatusPending:
		value := false
		return &value, nil
	default:
		return nil, fmt.Errorf("%w: unknown report snapshot status %q", dbutil.ErrInvalidInput, status)
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
