package reports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

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
func (s *Store) Snapshots(ctx context.Context, reportID int64) ([]ReportSnapshot, error) {
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
		ORDER BY lower(host_row.display_name), host_row.id`,
		reportID,
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
func (s *Store) HostSnapshots(ctx context.Context, host *hosts.Host) ([]ReportSnapshot, error) {
	rpts, err := s.assignedForHost(ctx, host)
	if err != nil {
		return nil, err
	}
	reportIDs := make([]int64, 0, len(rpts))
	for _, report := range rpts {
		reportIDs = append(reportIDs, report.ID)
	}
	states, err := s.loadHostSnapshotStates(ctx, host.ID, reportIDs)
	if err != nil {
		return nil, err
	}

	snapshots := make([]ReportSnapshot, 0, len(rpts))
	for _, report := range rpts {
		state := states[report.ID]
		snapshots = append(snapshots, ReportSnapshot{
			ReportID:          report.ID,
			ReportName:        report.Name,
			ReportDescription: report.Description,
			HostID:            host.ID,
			HostName:          host.DisplayName,
			Rows:              normalizeSnapshotRows(state.rows),
			CollectedAt:       state.collectedAt,
		})
	}
	return snapshots, nil
}

func (s *Store) assignedForHost(ctx context.Context, host *hosts.Host) ([]Report, error) {
	qrows, err := s.db.Pool().Query(ctx, reportSelectSQL()+`
		JOIN osquery_report_assignments assignment
			ON assignment.report_id = r.id
		   AND assignment.host_id = $1
		ORDER BY lower(r.name), r.id`, host.ID)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[reportRow])
	if err != nil {
		return nil, err
	}
	rpts := make([]Report, len(rows))
	for i, row := range rows {
		rpts[i] = reportFromRow(row)
	}
	return rpts, nil
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
		Rows:              rows,
		CollectedAt:       row.CollectedAt,
	}, nil
}

type hostSnapshotState struct {
	rows        []map[string]string
	collectedAt *time.Time
}

func (s *Store) loadHostSnapshotStates(
	ctx context.Context,
	hostID int64,
	reportIDs []int64,
) (map[int64]hostSnapshotState, error) {
	states := make(map[int64]hostSnapshotState, len(reportIDs))
	if len(reportIDs) == 0 {
		return states, nil
	}

	type stateRow struct {
		ReportID    int64     `db:"report_id"`
		Rows        []byte    `db:"rows"`
		CollectedAt time.Time `db:"collected_at"`
	}
	qrows, err := s.db.Pool().Query(ctx, `
		SELECT report_id, rows, collected_at
		FROM osquery_report_snapshots
		WHERE host_id = $1 AND report_id = ANY($2::bigint[])`,
		hostID,
		reportIDs,
	)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[stateRow])
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		snapshotRows, err := decodeSnapshotRows(row.Rows)
		if err != nil {
			return nil, err
		}
		collectedAt := row.CollectedAt
		states[row.ReportID] = hostSnapshotState{
			rows:        snapshotRows,
			collectedAt: &collectedAt,
		}
	}
	return states, nil
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
