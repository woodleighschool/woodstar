package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) ListForHost(
	ctx context.Context,
	hostID int64,
	params HostSoftwareListParams,
) ([]HostSoftwareRow, int, error) {
	params.SoftwareSources = dbutil.SplitListValues(params.SoftwareSources)
	whereSQL, args := hostSoftwareWhere(hostID, params)
	listQuery := hostSoftwareTitleListQuery(whereSQL, args, params)

	var total int
	countSQL, countArgs := listQuery.BuildCount()
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	titleIDs, err := s.hostSoftwareTitleIDs(ctx, listQuery)
	if err != nil {
		return nil, 0, err
	}
	if len(titleIDs) == 0 {
		return []HostSoftwareRow{}, total, nil
	}

	software, err := s.hostSoftwareRows(ctx, hostID, titleIDs)
	if err != nil {
		return nil, 0, err
	}
	return software, total, nil
}

const hostSoftwareRowsSQL = `
SELECT
    st.id AS title_id,
    st.name AS title_name,
    st.display_name,
    st.source,
    st.extension_for,
    s.id AS software_id,
    s.version,
    s.bundle_identifier,
    hs.last_opened_at,
    COALESCE(paths.installed_path, '') AS installed_path,
    COALESCE(paths.team_identifier, '') AS team_identifier,
    COALESCE(paths.cdhash_sha256, '') AS cdhash_sha256,
    COALESCE(paths.executable_sha256, '') AS executable_sha256,
    COALESCE(paths.executable_path, '') AS executable_path
FROM host_software hs
JOIN software s ON s.id = hs.software_id
JOIN software_titles st ON st.id = s.title_id
LEFT JOIN host_software_installed_paths paths
    ON paths.host_id = hs.host_id AND paths.software_id = hs.software_id
WHERE hs.host_id = $1
  AND st.id = ANY($2::bigint[])
ORDER BY array_position($2::bigint[], st.id), lower(s.version), paths.installed_path`

func (s *Store) hostSoftwareRows(
	ctx context.Context,
	hostID int64,
	titleIDs []int64,
) ([]HostSoftwareRow, error) {
	qrows, err := s.db.Pool().Query(ctx, hostSoftwareRowsSQL, hostID, titleIDs)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[hostSoftwareScanRow])
	if err != nil {
		return nil, err
	}
	return buildHostSoftwareRows(rows)
}

type hostSoftwareAccumulator struct {
	titles       orderedGroup[int64, HostSoftwareRow]
	versionByKey map[string]int
}

func buildHostSoftwareRows(rows []hostSoftwareScanRow) ([]HostSoftwareRow, error) {
	acc := hostSoftwareAccumulator{
		titles:       newOrderedGroup[int64, HostSoftwareRow](),
		versionByKey: make(map[string]int),
	}
	for _, row := range rows {
		acc.add(row)
	}
	return acc.rows(), nil
}

func (acc *hostSoftwareAccumulator) rows() []HostSoftwareRow {
	return acc.titles.values()
}

func (acc *hostSoftwareAccumulator) add(row hostSoftwareScanRow) {
	title := acc.title(row)
	versionIndex := acc.versionIndex(title, row)
	if row.InstalledPath == "" {
		return
	}
	version := &title.InstalledVersions[versionIndex]
	version.InstalledPaths = append(version.InstalledPaths, row.InstalledPath)
	version.SignatureInformation = append(version.SignatureInformation, PathSignatureInformation{
		InstalledPath:    row.InstalledPath,
		TeamIdentifier:   row.TeamIdentifier,
		CDHashSHA256:     row.CDHashSHA256,
		ExecutableSHA256: row.ExecutableSHA256,
		ExecutablePath:   row.ExecutablePath,
	})
}

func (acc *hostSoftwareAccumulator) title(row hostSoftwareScanRow) *HostSoftwareRow {
	return acc.titles.get(row.TitleID, func() HostSoftwareRow {
		return HostSoftwareRow{
			ID:           row.TitleID,
			Name:         row.TitleName,
			DisplayName:  row.DisplayName,
			Source:       row.Source,
			ExtensionFor: row.ExtensionFor,
		}
	})
}

func (acc *hostSoftwareAccumulator) versionIndex(title *HostSoftwareRow, row hostSoftwareScanRow) int {
	key := fmt.Sprintf("%d:%d", row.TitleID, row.SoftwareID)
	versionIndex, ok := acc.versionByKey[key]
	if ok {
		return versionIndex
	}
	title.InstalledVersions = append(title.InstalledVersions, HostSoftwareInstalledVersion{
		Version:          row.Version,
		BundleIdentifier: row.BundleIdentifier,
		LastOpenedAt:     row.LastOpenedAt,
	})
	versionIndex = len(title.InstalledVersions) - 1
	acc.versionByKey[key] = versionIndex
	return versionIndex
}

func (s *Store) hostSoftwareTitleIDs(
	ctx context.Context,
	listQuery dbutil.ListQuery,
) ([]int64, error) {
	query, queryArgs, err := listQuery.Build()
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Pool().Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func hostSoftwareTitleListQuery(
	whereSQL string,
	args []any,
	params HostSoftwareListParams,
) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: `
SELECT st.id
FROM host_software hs
JOIN software s ON s.id = hs.software_id
JOIN software_titles st ON st.id = s.title_id`,
		WhereSQL:   whereSQL,
		GroupBySQL: "GROUP BY st.id",
		Args:       args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":           {SQL: "MIN(lower(st.name))"},
			"version":        {SQL: "MIN(lower(s.version))"},
			"source":         {SQL: "MIN(lower(st.source))"},
			"last_opened_at": {SQL: "MAX(hs.last_opened_at)", NullOrder: dbutil.NullsLast},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "MIN(lower(st.name))"}, {SQL: "st.id"}},
		Params:       params.ListParams,
	}
}

func hostSoftwareWhere(hostID int64, params HostSoftwareListParams) (string, []any) {
	var where dbutil.WhereBuilder
	where.Add("hs.host_id = " + where.Arg(hostID))
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			st.name ILIKE ` + search + `
			OR st.display_name ILIKE ` + search + `
			OR st.source ILIKE ` + search + `
			OR st.extension_for ILIKE ` + search + `
			OR st.bundle_identifier ILIKE ` + search + `
			OR s.version ILIKE ` + search + `
			OR s.bundle_identifier ILIKE ` + search + `
			OR EXISTS (
				SELECT 1
				FROM host_software_installed_paths paths
				WHERE paths.host_id = hs.host_id
					AND paths.software_id = hs.software_id
					AND paths.installed_path ILIKE ` + search + `
			)
		)`)
	}
	if len(params.SoftwareSources) > 0 {
		where.Add("st.source = ANY(" + where.Arg(params.SoftwareSources) + "::text[])")
	}
	return where.Build()
}

type hostSoftwareScanRow struct {
	TitleID          int64      `db:"title_id"`
	TitleName        string     `db:"title_name"`
	DisplayName      string     `db:"display_name"`
	Source           string     `db:"source"`
	ExtensionFor     string     `db:"extension_for"`
	SoftwareID       int64      `db:"software_id"`
	Version          string     `db:"version"`
	BundleIdentifier string     `db:"bundle_identifier"`
	LastOpenedAt     *time.Time `db:"last_opened_at"`
	InstalledPath    string     `db:"installed_path"`
	TeamIdentifier   string     `db:"team_identifier"`
	CDHashSHA256     string     `db:"cdhash_sha256"`
	ExecutableSHA256 string     `db:"executable_sha256"`
	ExecutablePath   string     `db:"executable_path"`
}
