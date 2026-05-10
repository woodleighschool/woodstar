package models

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// HostSoftwareEntry is one installed software version reported by a host.
type HostSoftwareEntry struct {
	Name             string
	Version          string
	Source           string
	BundleIdentifier string
	ExtensionID      string
	ExtensionFor     string
	Vendor           string
	Arch             string
	Release          string
	InstalledPath    string
	TeamIdentifier   string
	CDHashSHA256     string
	ExecutableSHA256 string
	ExecutablePath   string
	LastOpenedAt     *time.Time
}

// SoftwareVersion is one observed version under a software title.
type SoftwareVersion struct {
	ID               int64
	Version          string
	BundleIdentifier string
	HostsCount       int
}

// PathSignatureInformation is code-signing and executable hash data for an installed path.
type PathSignatureInformation struct {
	InstalledPath    string
	TeamIdentifier   string
	CDHashSHA256     string
	ExecutableSHA256 string
	ExecutablePath   string
}

// HostSoftwareInstalledVersion is a host's installed software version and paths.
type HostSoftwareInstalledVersion struct {
	Version              string
	BundleIdentifier     string
	InstalledPaths       []string
	SignatureInformation []PathSignatureInformation
	LastOpenedAt         *time.Time
}

// HostSoftwareRow is software inventory projected for one host.
type HostSoftwareRow struct {
	ID                int64
	Name              string
	DisplayName       string
	Source            string
	ExtensionFor      string
	InstalledVersions []HostSoftwareInstalledVersion
}

// SoftwareTitle is an aggregate software title row.
type SoftwareTitle struct {
	ID               int64
	Name             string
	DisplayName      string
	Source           string
	ExtensionFor     string
	BundleIdentifier string
	Vendor           string
	HostsCount       int
	VersionsCount    int
	CountsUpdatedAt  *time.Time
	Versions         []SoftwareVersion
}

// SoftwareTitleListParams controls software title list filtering and sorting.
type SoftwareTitleListParams struct {
	ListParams

	SoftwareSources []string
}

// HostSoftwareListParams controls software installed on one host.
type HostSoftwareListParams struct {
	ListParams

	SoftwareSources []string
}

// SoftwareStore persists global software titles and host inventory joins.
type SoftwareStore struct {
	db *database.DB
	q  *sqlc.Queries
}

// NewSoftwareStore returns a software store backed by db.
func NewSoftwareStore(db *database.DB) *SoftwareStore {
	return &SoftwareStore{db: db, q: db.Queries()}
}

// ReplaceHostSoftware replaces a host's software snapshot in one transaction.
func (s *SoftwareStore) ReplaceHostSoftware(ctx context.Context, hostID int64, entries []HostSoftwareEntry) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := resetHostSoftware(ctx, q, hostID); err != nil {
			return err
		}
		for _, entry := range entries {
			if err := replaceHostSoftwareEntry(ctx, q, hostID, entry); err != nil {
				return err
			}
		}
		return nil
	})
}

func resetHostSoftware(ctx context.Context, q *sqlc.Queries, hostID int64) error {
	if err := q.DeleteHostSoftwarePaths(ctx, sqlc.DeleteHostSoftwarePathsParams{HostID: hostID}); err != nil {
		return err
	}
	return q.DeleteHostSoftware(ctx, sqlc.DeleteHostSoftwareParams{HostID: hostID})
}

func replaceHostSoftwareEntry(ctx context.Context, q *sqlc.Queries, hostID int64, entry HostSoftwareEntry) error {
	entry = cleanHostSoftwareEntry(entry)
	if entry.Name == "" || entry.Source == "" {
		return nil
	}
	softwareID, err := softwareIDFor(ctx, q, entry)
	if err != nil {
		return err
	}
	if err := q.UpsertHostSoftware(ctx, sqlc.UpsertHostSoftwareParams{
		HostID:       hostID,
		SoftwareID:   softwareID,
		LastOpenedAt: entry.LastOpenedAt,
	}); err != nil {
		return err
	}
	if entry.InstalledPath == "" {
		return nil
	}
	return q.InsertHostSoftwareInstalledPath(ctx, sqlc.InsertHostSoftwareInstalledPathParams{
		HostID:           hostID,
		SoftwareID:       softwareID,
		InstalledPath:    entry.InstalledPath,
		TeamIdentifier:   entry.TeamIdentifier,
		CdhashSha256:     entry.CDHashSHA256,
		ExecutableSha256: entry.ExecutableSHA256,
		ExecutablePath:   entry.ExecutablePath,
	})
}

func softwareIDFor(ctx context.Context, q *sqlc.Queries, entry HostSoftwareEntry) (int64, error) {
	titleID, err := softwareTitleIDFor(ctx, q, entry)
	if err != nil {
		return 0, err
	}
	row, err := q.UpsertSoftware(ctx, sqlc.UpsertSoftwareParams{
		TitleID:          titleID,
		Name:             entry.Name,
		Version:          entry.Version,
		Source:           entry.Source,
		BundleIdentifier: entry.BundleIdentifier,
		ExtensionID:      entry.ExtensionID,
		ExtensionFor:     entry.ExtensionFor,
		Vendor:           entry.Vendor,
		Arch:             entry.Arch,
		Release:          entry.Release,
	})
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

func softwareTitleIDFor(ctx context.Context, q *sqlc.Queries, entry HostSoftwareEntry) (int64, error) {
	displayName := entry.Name
	if entry.BundleIdentifier != "" {
		row, err := q.UpsertSoftwareTitleByBundle(ctx, sqlc.UpsertSoftwareTitleByBundleParams{
			Name:             entry.Name,
			DisplayName:      displayName,
			Source:           entry.Source,
			ExtensionFor:     entry.ExtensionFor,
			BundleIdentifier: entry.BundleIdentifier,
			Vendor:           entry.Vendor,
		})
		if err != nil {
			return 0, err
		}
		return row.ID, nil
	}
	row, err := q.UpsertSoftwareTitleByName(ctx, sqlc.UpsertSoftwareTitleByNameParams{
		Name:             entry.Name,
		DisplayName:      displayName,
		Source:           entry.Source,
		ExtensionFor:     entry.ExtensionFor,
		BundleIdentifier: entry.BundleIdentifier,
		Vendor:           entry.Vendor,
	})
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

// ListTitles returns software titles and the total count matching params.
func (s *SoftwareStore) ListTitles(ctx context.Context, params SoftwareTitleListParams) ([]SoftwareTitle, int, error) {
	params = cleanSoftwareTitleListParams(params)
	whereSQL, args := softwareTitleWhere(params)

	countSQL := `SELECT count(*) FROM software_titles st` + whereSQL
	var total int
	if err := s.db.Pool().QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderSQL, err := softwareTitleOrder(params.OrderKey, params.OrderDirection)
	if err != nil {
		return nil, 0, err
	}
	limitIndex := len(args) + 1
	args = append(args, int32(params.PerPage), int32((params.Page-1)*params.PerPage))
	rows, err := s.db.Pool().Query(ctx, softwareTitleListSQL(whereSQL, orderSQL, limitIndex), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	titles, err := scanSoftwareTitles(rows)
	if err != nil {
		return nil, 0, err
	}
	if err := s.loadSoftwareTitleVersions(ctx, titles); err != nil {
		return nil, 0, err
	}
	return titles, total, nil
}

// GetTitle returns one software title by ID.
func (s *SoftwareStore) GetTitle(ctx context.Context, id int64) (*SoftwareTitle, error) {
	rows, err := s.db.Pool().
		Query(ctx, softwareTitleListSQL(" WHERE st.id = $1", "ORDER BY lower(st.name)", 2), id, int32(1), int32(0))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	titles, err := scanSoftwareTitles(rows)
	if err != nil {
		return nil, err
	}
	if len(titles) == 0 {
		return nil, ErrNotFound
	}
	if err := s.loadSoftwareTitleVersions(ctx, titles); err != nil {
		return nil, err
	}
	return &titles[0], nil
}

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
	titleIDs, err := s.hostSoftwareTitleIDs(
		ctx,
		whereSQL,
		orderSQL,
		args,
		params,
	)
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

func cleanHostSoftwareEntry(entry HostSoftwareEntry) HostSoftwareEntry {
	entry.Name = strings.TrimSpace(entry.Name)
	entry.Version = strings.TrimSpace(entry.Version)
	entry.Source = strings.TrimSpace(entry.Source)
	entry.BundleIdentifier = strings.TrimSpace(entry.BundleIdentifier)
	entry.ExtensionID = strings.TrimSpace(entry.ExtensionID)
	entry.ExtensionFor = strings.TrimSpace(entry.ExtensionFor)
	entry.Vendor = strings.TrimSpace(entry.Vendor)
	entry.Arch = strings.TrimSpace(entry.Arch)
	entry.Release = strings.TrimSpace(entry.Release)
	entry.InstalledPath = strings.TrimSpace(entry.InstalledPath)
	entry.TeamIdentifier = strings.TrimSpace(entry.TeamIdentifier)
	entry.CDHashSHA256 = strings.TrimSpace(entry.CDHashSHA256)
	entry.ExecutableSHA256 = strings.TrimSpace(entry.ExecutableSHA256)
	entry.ExecutablePath = strings.TrimSpace(entry.ExecutablePath)
	return entry
}

func cleanSoftwareTitleListParams(params SoftwareTitleListParams) SoftwareTitleListParams {
	params.ListParams = CleanListParams(params.ListParams)
	params.SoftwareSources = SplitListValues(params.SoftwareSources)
	return params
}

func cleanHostSoftwareListParams(params HostSoftwareListParams) HostSoftwareListParams {
	params.ListParams = CleanListParams(params.ListParams)
	params.SoftwareSources = SplitListValues(params.SoftwareSources)
	return params
}

func softwareTitleWhere(params SoftwareTitleListParams) (string, []any) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0)
	if params.Q != "" {
		args = append(args, "%"+params.Q+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, `(
			st.name ILIKE `+placeholder+`
			OR st.display_name ILIKE `+placeholder+`
			OR st.bundle_identifier ILIKE `+placeholder+`
			OR st.extension_for ILIKE `+placeholder+`
			OR EXISTS (
				SELECT 1 FROM software s
				WHERE s.title_id = st.id AND s.version ILIKE `+placeholder+`
			)
		)`)
	}
	if len(params.SoftwareSources) > 0 {
		args = append(args, params.SoftwareSources)
		clauses = append(clauses, fmt.Sprintf("st.source = ANY($%d::text[])", len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func softwareTitleOrder(orderKey string, direction string) (string, error) {
	return listOrderBy(
		CleanListParams(ListParams{OrderKey: orderKey, OrderDirection: direction}),
		map[string]orderExpr{
			"name":              {SQL: "lower(st.name)"},
			"source":            {SQL: "lower(st.source)"},
			"hosts_count":       {SQL: "hosts_count"},
			"versions_count":    {SQL: "versions_count"},
			"counts_updated_at": {SQL: "counts_updated_at", NullsLast: true},
		},
		[]orderExpr{{SQL: "lower(st.name)"}, {SQL: "st.id"}},
	)
}

func softwareTitleListSQL(whereSQL string, orderSQL string, limitIndex int) string {
	return `
SELECT
	st.id,
	st.name,
	st.display_name,
	st.source,
	st.extension_for,
	st.bundle_identifier,
	st.vendor,
	COUNT(DISTINCT hs.host_id)::integer AS hosts_count,
	COUNT(DISTINCT s.id)::integer AS versions_count,
	MAX(hs.last_seen_at) AS counts_updated_at
FROM software_titles st
LEFT JOIN software s ON s.title_id = st.id
LEFT JOIN host_software hs ON hs.software_id = s.id
` + whereSQL + `
GROUP BY st.id
` + orderSQL + `
LIMIT $` + strconv.Itoa(limitIndex) + ` OFFSET $` + strconv.Itoa(limitIndex+1)
}

func scanSoftwareTitles(rows pgx.Rows) ([]SoftwareTitle, error) {
	titles := make([]SoftwareTitle, 0)
	for rows.Next() {
		var title SoftwareTitle
		if err := rows.Scan(
			&title.ID,
			&title.Name,
			&title.DisplayName,
			&title.Source,
			&title.ExtensionFor,
			&title.BundleIdentifier,
			&title.Vendor,
			&title.HostsCount,
			&title.VersionsCount,
			&title.CountsUpdatedAt,
		); err != nil {
			return nil, err
		}
		titles = append(titles, title)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return titles, nil
}

func (s *SoftwareStore) loadSoftwareTitleVersions(ctx context.Context, titles []SoftwareTitle) error {
	for i := range titles {
		rows, err := s.db.Pool().Query(ctx, `
SELECT
	s.id,
	s.version,
	s.bundle_identifier,
	COUNT(DISTINCT hs.host_id)::integer AS hosts_count
FROM software s
LEFT JOIN host_software hs ON hs.software_id = s.id
WHERE s.title_id = $1
GROUP BY s.id
ORDER BY lower(s.version), s.id`, titles[i].ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var version SoftwareVersion
			if err := rows.Scan(
				&version.ID,
				&version.Version,
				&version.BundleIdentifier,
				&version.HostsCount,
			); err != nil {
				rows.Close()
				return err
			}
			titles[i].Versions = append(titles[i].Versions, version)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}
	return nil
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
	return listOrderBy(
		CleanListParams(ListParams{OrderKey: orderKey, OrderDirection: direction}),
		map[string]orderExpr{
			"name":           {SQL: "order_name"},
			"version":        {SQL: "order_version"},
			"source":         {SQL: "order_source"},
			"last_opened_at": {SQL: "order_last_opened_at", NullsLast: true},
		},
		[]orderExpr{{SQL: "order_name"}, {SQL: "st.id"}},
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
