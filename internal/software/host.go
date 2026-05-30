package software

import (
	"context"
	"fmt"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
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

func (s *Store) hostSoftwareRows(
	ctx context.Context,
	hostID int64,
	titleIDs []int64,
) ([]HostSoftwareRow, error) {
	rows, err := s.q.ListHostSoftwareRows(ctx, sqlc.ListHostSoftwareRowsParams{
		HostID:   hostID,
		TitleIds: titleIDs,
	})
	if err != nil {
		return nil, err
	}
	return scanHostSoftwareRows(rows)
}

type hostSoftwareAccumulator struct {
	titles       orderedGroup[int64, HostSoftwareRow]
	versionByKey map[string]int
}

func scanHostSoftwareRows(rows []sqlc.ListHostSoftwareRowsRow) ([]HostSoftwareRow, error) {
	acc := hostSoftwareAccumulator{
		titles:       newOrderedGroup[int64, HostSoftwareRow](),
		versionByKey: make(map[string]int),
	}
	for _, row := range rows {
		acc.add(hostSoftwareDBRowFromSQLC(row))
	}
	return acc.rows(), nil
}

func (acc *hostSoftwareAccumulator) rows() []HostSoftwareRow {
	return acc.titles.values()
}

func (acc *hostSoftwareAccumulator) add(row hostSoftwareDBRow) {
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

func (acc *hostSoftwareAccumulator) title(row hostSoftwareDBRow) *HostSoftwareRow {
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

func (acc *hostSoftwareAccumulator) versionIndex(title *HostSoftwareRow, row hostSoftwareDBRow) int {
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

type hostSoftwareDBRow struct {
	TitleID          int64
	TitleName        string
	DisplayName      string
	Source           string
	ExtensionFor     string
	SoftwareID       int64
	Version          string
	BundleIdentifier string
	LastOpenedAt     *time.Time
	InstalledPath    string
	TeamIdentifier   string
	CDHashSHA256     string
	ExecutableSHA256 string
	ExecutablePath   string
}

func hostSoftwareDBRowFromSQLC(row sqlc.ListHostSoftwareRowsRow) hostSoftwareDBRow {
	return hostSoftwareDBRow{
		TitleID:          row.TitleID,
		TitleName:        row.TitleName,
		DisplayName:      row.DisplayName,
		Source:           row.Source,
		ExtensionFor:     row.ExtensionFor,
		SoftwareID:       row.SoftwareID,
		Version:          row.Version,
		BundleIdentifier: row.BundleIdentifier,
		LastOpenedAt:     row.LastOpenedAt,
		InstalledPath:    row.InstalledPath,
		TeamIdentifier:   row.TeamIdentifier,
		CDHashSHA256:     row.CdhashSha256,
		ExecutableSHA256: row.ExecutableSha256,
		ExecutablePath:   row.ExecutablePath,
	}
}
