package software

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

type softwareReportCount struct {
	Expected  int32 `db:"expected_host_count"`
	Installed int32 `db:"installed_host_count"`
}

type softwareReportCountRow struct {
	SoftwareID int64 `db:"software_id"`
	Expected   int32 `db:"expected_host_count"`
	Installed  int32 `db:"installed_host_count"`
}

type softwareReportHostRow struct {
	HostID         int64      `db:"host_id"`
	HostName       string     `db:"host_name"`
	HardwareSerial string     `db:"hardware_serial"`
	Installed      *bool      `db:"installed"`
	TargetVersion  *string    `db:"target_version"`
	EvaluatedAt    *time.Time `db:"evaluated_at"`
}

// ListReportHosts returns hosts whose effective assignment includes managed installs.
func (s *Store) ListReportHosts(
	ctx context.Context,
	softwareID int64,
	params SoftwareReportHostListParams,
) ([]SoftwareReportHost, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	where, args := softwareReportWhere(params, softwareID)
	listQuery := postgres.ListQuery{
		SelectSQL: `
SELECT
	h.id AS host_id,
	h.display_name AS host_name,
	h.hardware_serial,
	observed.installed,
	observed.target_version,
	COALESCE(report.run_ended_at, report.run_started_at) AS evaluated_at
FROM hosts h
CROSS JOIN LATERAL munki_resolved_software_for_host(h.id) resolved
LEFT JOIN munki_host_status report ON report.host_id = h.id
LEFT JOIN munki_host_items observed
	ON observed.host_id = h.id
	AND observed.name = resolved.name`,
		WhereSQL: where,
		Args:     args,
		OrderKeys: map[string]postgres.OrderExpr{
			"host_name":      {SQL: "lower(h.display_name)"},
			"status":         {SQL: softwareReportStatusOrderSQL()},
			"target_version": {SQL: "observed.target_version", NullOrder: postgres.NullsLast},
			"evaluated_at":   {SQL: "COALESCE(report.run_ended_at, report.run_started_at)", NullOrder: postgres.NullsLast},
		},
		DefaultOrder: []postgres.OrderExpr{
			{SQL: "lower(h.display_name)"},
			{SQL: "h.id"},
		},
		Params: params.ListParams,
	}
	rows, count, err := postgres.ListWithCount[softwareReportHostRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		var exists bool
		if err := s.pool.QueryRow(
			ctx,
			`SELECT EXISTS (SELECT 1 FROM munki_software WHERE id = $1)`,
			softwareID,
		).Scan(&exists); err != nil {
			return nil, 0, err
		}
		if !exists {
			return nil, 0, fault.ErrNotFound
		}
	}
	return softwareReportHostsFromRows(rows), count, nil
}

func (s *Store) loadSoftwareReportCounts(
	ctx context.Context,
	softwareIDs []int64,
) (map[int64]softwareReportCount, error) {
	counts := make(map[int64]softwareReportCount, len(softwareIDs))
	if len(softwareIDs) == 0 {
		return counts, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT
	resolved.software_id,
	COUNT(*)::integer AS expected_host_count,
	COUNT(*) FILTER (
		WHERE observed.installed IS TRUE
		AND observed.target_version = ''
	)::integer AS installed_host_count
FROM hosts h
CROSS JOIN LATERAL munki_resolved_software_for_host(h.id) resolved
LEFT JOIN munki_host_items observed
	ON observed.host_id = h.id
	AND observed.name = resolved.name
WHERE resolved.software_id = ANY($1::bigint[])
	AND 'managed_installs' = ANY(resolved.actions)
GROUP BY resolved.software_id`, softwareIDs)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[softwareReportCountRow])
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		counts[record.SoftwareID] = softwareReportCount{
			Expected:  record.Expected,
			Installed: record.Installed,
		}
	}
	return counts, nil
}

func softwareReportWhere(
	params SoftwareReportHostListParams,
	softwareID int64,
) (string, []any) {
	var where postgres.WhereBuilder
	where.Add("resolved.software_id = " + where.Arg(softwareID))
	where.Add("'managed_installs' = ANY(resolved.actions)")
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add("(h.display_name ILIKE " + search + " OR h.hardware_serial ILIKE " + search + ")")
	}
	if len(params.Statuses) > 0 {
		statuses := where.Arg(params.Statuses)
		where.Add(`(
			('installed' = ANY(` + statuses + `::text[])
				AND observed.installed IS TRUE
				AND observed.target_version = '')
			OR ('pending' = ANY(` + statuses + `::text[])
				AND observed.target_version <> '')
		)`)
	}
	return where.Build()
}

func softwareReportStatusOrderSQL() string {
	return `CASE
		WHEN observed.target_version <> '' THEN 0
		WHEN observed.installed IS TRUE THEN 1
		ELSE 2
	END`
}

func softwareReportHostsFromRows(rows []softwareReportHostRow) []SoftwareReportHost {
	hosts := make([]SoftwareReportHost, len(rows))
	for i, row := range rows {
		hosts[i] = SoftwareReportHost{
			HostID:         row.HostID,
			HostName:       row.HostName,
			HardwareSerial: row.HardwareSerial,
			Status:         softwareReportStatus(row),
			TargetVersion:  valueOrEmpty(row.TargetVersion),
			EvaluatedAt:    row.EvaluatedAt,
		}
	}
	return hosts
}

func softwareReportStatus(row softwareReportHostRow) *SoftwareReportStatus {
	if valueOrEmpty(row.TargetVersion) != "" {
		status := SoftwareReportStatusPending
		return &status
	}
	if valueOrFalse(row.Installed) {
		status := SoftwareReportStatusInstalled
		return &status
	}
	return nil
}
