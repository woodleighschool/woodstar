package software

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/store"
)

// ListForHost returns software installed on a host grouped by title.
func (s *SoftwareStore) ListForHost(
	ctx context.Context,
	hostID int64,
	params HostSoftwareListParams,
) ([]HostSoftwareRow, int, error) {
	params = cleanHostSoftwareListParams(params)
	whereSQL, args := hostSoftwareWhere(params, []any{hostID})

	countSQL := `
SELECT count(DISTINCT st.id)
FROM host_software hs
JOIN software s ON s.id = hs.software_id
JOIN software_titles st ON st.id = s.title_id
` + whereSQL
	var total int
	if err := s.db.Pool().QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderSQL, err := hostSoftwareOrder(params.OrderKey, params.OrderDirection)
	if err != nil {
		return nil, 0, err
	}
	titleIDs, err := s.hostSoftwareTitleIDs(ctx, whereSQL, orderSQL, args, params)
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

func (s *SoftwareStore) hostSoftwareRows(
	ctx context.Context,
	hostID int64,
	titleIDs []int64,
) ([]HostSoftwareRow, error) {
	rows, err := s.db.Pool().Query(ctx, hostSoftwareSQL, hostID, titleIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHostSoftwareRows(rows)
}

type hostSoftwareAccumulator struct {
	byTitle      map[int64]*HostSoftwareRow
	ordered      []int64
	versionByKey map[string]int
}

func scanHostSoftwareRows(rows pgx.Rows) ([]HostSoftwareRow, error) {
	acc := hostSoftwareAccumulator{
		byTitle:      make(map[int64]*HostSoftwareRow),
		ordered:      make([]int64, 0),
		versionByKey: make(map[string]int),
	}
	for rows.Next() {
		row, err := scanHostSoftwareDBRow(rows)
		if err != nil {
			return nil, err
		}
		acc.add(row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return acc.rows(), nil
}

func (acc *hostSoftwareAccumulator) rows() []HostSoftwareRow {
	software := make([]HostSoftwareRow, 0, len(acc.ordered))
	for _, id := range acc.ordered {
		software = append(software, *acc.byTitle[id])
	}
	return software
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
	title := acc.byTitle[row.TitleID]
	if title != nil {
		return title
	}
	title = &HostSoftwareRow{
		ID:           row.TitleID,
		Name:         row.TitleName,
		DisplayName:  row.DisplayName,
		Source:       row.Source,
		ExtensionFor: row.ExtensionFor,
	}
	acc.byTitle[row.TitleID] = title
	acc.ordered = append(acc.ordered, row.TitleID)
	return title
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

func scanHostSoftwareDBRow(rows pgx.Rows) (hostSoftwareDBRow, error) {
	var row hostSoftwareDBRow
	err := rows.Scan(
		&row.TitleID,
		&row.TitleName,
		&row.DisplayName,
		&row.Source,
		&row.ExtensionFor,
		&row.SoftwareID,
		&row.Version,
		&row.BundleIdentifier,
		&row.LastOpenedAt,
		&row.InstalledPath,
		&row.TeamIdentifier,
		&row.CDHashSHA256,
		&row.ExecutableSHA256,
		&row.ExecutablePath,
	)
	return row, err
}

func cleanHostSoftwareListParams(params HostSoftwareListParams) HostSoftwareListParams {
	params.ListParams = store.CleanListParams(params.ListParams)
	params.SoftwareSources = store.SplitListValues(params.SoftwareSources)
	return params
}

func (s *SoftwareStore) hostSoftwareTitleIDs(
	ctx context.Context,
	whereSQL string,
	orderSQL string,
	args []any,
	params HostSoftwareListParams,
) ([]int64, error) {
	limitIndex := len(args) + 1
	queryArgs := append(append([]any{}, args...), int32(params.PerPage), int32((params.Page-1)*params.PerPage))
	rows, err := s.db.Pool().Query(ctx, `
SELECT
	st.id,
	MIN(lower(st.name)) AS order_name,
	MIN(lower(s.version)) AS order_version,
	MAX(hs.last_opened_at) AS order_last_opened_at,
	MIN(lower(st.source)) AS order_source
FROM host_software hs
JOIN software s ON s.id = hs.software_id
JOIN software_titles st ON st.id = s.title_id
`+whereSQL+`
GROUP BY st.id
`+orderSQL+`
LIMIT $`+strconv.Itoa(limitIndex)+` OFFSET $`+strconv.Itoa(limitIndex+1), queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		var orderName string
		var orderVersion string
		var orderLastOpenedAt *time.Time
		var orderSource string
		if err := rows.Scan(&id, &orderName, &orderVersion, &orderLastOpenedAt, &orderSource); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func hostSoftwareWhere(params HostSoftwareListParams, args []any) (string, []any) {
	clauses := []string{"hs.host_id = $1"}
	if params.Q != "" {
		args = append(args, "%"+params.Q+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, `(
			st.name ILIKE `+placeholder+`
			OR st.display_name ILIKE `+placeholder+`
			OR st.source ILIKE `+placeholder+`
			OR st.extension_for ILIKE `+placeholder+`
			OR st.bundle_identifier ILIKE `+placeholder+`
			OR s.version ILIKE `+placeholder+`
			OR s.bundle_identifier ILIKE `+placeholder+`
			OR EXISTS (
				SELECT 1
				FROM host_software_installed_paths paths
				WHERE paths.host_id = hs.host_id
					AND paths.software_id = hs.software_id
					AND paths.installed_path ILIKE `+placeholder+`
			)
		)`)
	}
	if len(params.SoftwareSources) > 0 {
		args = append(args, params.SoftwareSources)
		clauses = append(clauses, fmt.Sprintf("st.source = ANY($%d::text[])", len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func hostSoftwareOrder(orderKey string, direction string) (string, error) {
	return store.OrderBy(
		store.CleanListParams(store.ListParams{OrderKey: orderKey, OrderDirection: direction}),
		map[string]store.OrderExpr{
			"name":           {SQL: "order_name"},
			"version":        {SQL: "order_version"},
			"source":         {SQL: "order_source"},
			"last_opened_at": {SQL: "order_last_opened_at", NullsLast: true},
		},
		[]store.OrderExpr{{SQL: "order_name"}, {SQL: "st.id"}},
	)
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

const hostSoftwareSQL = `
SELECT
	st.id,
	st.name,
	st.display_name,
	st.source,
	st.extension_for,
	s.id,
	s.version,
	s.bundle_identifier,
	hs.last_opened_at,
	COALESCE(paths.installed_path, ''),
	COALESCE(paths.team_identifier, ''),
	COALESCE(paths.cdhash_sha256, ''),
	COALESCE(paths.executable_sha256, ''),
	COALESCE(paths.executable_path, '')
FROM host_software hs
JOIN software s ON s.id = hs.software_id
JOIN software_titles st ON st.id = s.title_id
LEFT JOIN host_software_installed_paths paths
	ON paths.host_id = hs.host_id AND paths.software_id = hs.software_id
WHERE hs.host_id = $1
	AND st.id = ANY($2::bigint[])
ORDER BY array_position($2::bigint[], st.id), lower(s.version), paths.installed_path`
