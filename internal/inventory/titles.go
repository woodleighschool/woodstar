package inventory

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) ListTitles(ctx context.Context, params SoftwareTitleListParams) ([]SoftwareTitle, int, error) {
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	params.SoftwareSources = dbutil.NormalizeListValues(params.SoftwareSources)
	whereSQL, args := softwareTitleWhere(params)
	listQuery := softwareTitleListQuery(params.ListParams, whereSQL, args)

	titles, total, err := dbutil.ListWithCount[SoftwareTitle](ctx, s.db.Pool(), listQuery)
	if err != nil {
		return nil, 0, err
	}
	setSoftwareTitleBrowsers(titles)
	if err := s.loadSoftwareTitleVersions(ctx, titles); err != nil {
		return nil, 0, err
	}
	if err := s.loadSoftwareTitleSigningIdentities(ctx, titles); err != nil {
		return nil, 0, err
	}
	return titles, total, nil
}

func (s *Store) GetTitle(ctx context.Context, id int64) (*SoftwareTitle, error) {
	query := softwareTitleSelectSQL + "\nWHERE st.id = $1\nGROUP BY st.id"
	title, err := dbutil.GetOne[SoftwareTitle](ctx, s.db.Pool(), query, id)
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	titles := []SoftwareTitle{title}
	setSoftwareTitleBrowsers(titles)
	if err := s.loadSoftwareTitleVersions(ctx, titles); err != nil {
		return nil, err
	}
	if err := s.loadSoftwareTitleSigningIdentities(ctx, titles); err != nil {
		return nil, err
	}
	return &titles[0], nil
}

func softwareTitleWhere(params SoftwareTitleListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			st.name ILIKE ` + search + `
			OR st.bundle_identifier ILIKE ` + search + `
			OR st.extension_for ILIKE ` + search + `
			OR EXISTS (
				SELECT 1 FROM software s
				WHERE s.title_id = st.id AND s.version ILIKE ` + search + `
			)
			OR EXISTS (
				SELECT 1
				FROM software s
				JOIN host_software_installed_paths paths ON paths.software_id = s.id
				WHERE s.title_id = st.id
					AND (
						paths.team_identifier ILIKE ` + search + `
						OR paths.identifier ILIKE ` + search + `
						OR paths.signing_authority ILIKE ` + search + `
					)
			)
		)`)
	}
	if len(params.SoftwareSources) > 0 {
		where.Add("st.source = ANY(" + where.Arg(params.SoftwareSources) + "::text[])")
	}
	return where.Build()
}

func softwareTitleListQuery(params dbutil.ListParams, whereSQL string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL:  softwareTitleSelectSQL,
		WhereSQL:   whereSQL,
		GroupBySQL: "GROUP BY st.id",
		Args:       args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":        {SQL: "lower(st.name)"},
			"source":      {SQL: "lower(st.source)"},
			"hosts_count": {SQL: "hosts_count"},
			"versions":    {SQL: "versions_count"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(st.name)"}, {SQL: "st.id"}},
		Params:       params,
	}
}

const softwareTitleSelectSQL = `
SELECT
	st.id,
	st.name,
	st.source,
	st.extension_for,
	st.bundle_identifier,
	st.vendor,
	COUNT(DISTINCT hs.host_id)::integer AS hosts_count,
	COUNT(DISTINCT s.id)::integer AS versions_count
FROM software_titles st
LEFT JOIN software s ON s.title_id = st.id
LEFT JOIN host_software hs ON hs.software_id = s.id
`

func setSoftwareTitleBrowsers(titles []SoftwareTitle) {
	for i := range titles {
		titles[i].Browser = browserFor(titles[i].Source, titles[i].ExtensionFor)
	}
}

// browserFor returns the browser name when source indicates a browser
// extension; otherwise empty.
func browserFor(source, extensionFor string) string {
	switch source {
	case SourceChromeExtensions, SourceFirefoxAddons, SourceSafariExtensions:
		return extensionFor
	default:
		return ""
	}
}

type softwareTitleVersionRow struct {
	TitleID          int64  `db:"title_id"`
	ID               int64  `db:"id"`
	Version          string `db:"version"`
	BundleIdentifier string `db:"bundle_identifier"`
	HostsCount       int32  `db:"hosts_count"`
}

const softwareTitleVersionsSQL = `
SELECT
    s.title_id,
    s.id,
    s.version,
    s.bundle_identifier,
    COUNT(DISTINCT hs.host_id)::integer AS hosts_count
FROM software s
LEFT JOIN host_software hs ON hs.software_id = s.id
WHERE s.title_id = ANY($1::bigint[])
GROUP BY s.id
ORDER BY array_position($1::bigint[], s.title_id), lower(s.version), s.id`

func (s *Store) loadSoftwareTitleVersions(ctx context.Context, titles []SoftwareTitle) error {
	if len(titles) == 0 {
		return nil
	}
	titleIDs := make([]int64, len(titles))
	titleIndex := make(map[int64]int, len(titles))
	for i := range titles {
		titleIDs[i] = titles[i].ID
		titleIndex[titles[i].ID] = i
		titles[i].Versions = SoftwareVersionList{
			Items: make([]SoftwareVersion, 0, titles[i].VersionsCount),
			Count: titles[i].VersionsCount,
		}
	}

	qrows, err := s.db.Pool().Query(ctx, softwareTitleVersionsSQL, titleIDs)
	if err != nil {
		return err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[softwareTitleVersionRow])
	if err != nil {
		return err
	}

	for _, row := range rows {
		i, ok := titleIndex[row.TitleID]
		if !ok {
			continue
		}
		titles[i].Versions.Items = append(titles[i].Versions.Items, SoftwareVersion{
			ID:               row.ID,
			Version:          row.Version,
			BundleIdentifier: row.BundleIdentifier,
			HostsCount:       row.HostsCount,
		})
	}
	return nil
}

type softwareTitleSigningIdentityRow struct {
	TitleID        int64    `db:"title_id"`
	Identifier     string   `db:"identifier"`
	TeamIdentifier string   `db:"team_identifier"`
	Authorities    []string `db:"authorities"`
	HostsCount     int32    `db:"hosts_count"`
}

const softwareTitleSigningIdentitiesSQL = `
SELECT
	s.title_id,
	paths.identifier,
	paths.team_identifier,
	COALESCE(
		array_agg(DISTINCT paths.signing_authority ORDER BY paths.signing_authority)
			FILTER (WHERE paths.signing_authority <> ''),
		ARRAY[]::text[]
	) AS authorities,
	COUNT(DISTINCT paths.host_id)::integer AS hosts_count
FROM host_software_installed_paths paths
JOIN software s ON s.id = paths.software_id
WHERE s.title_id = ANY($1::bigint[])
	AND paths.team_identifier <> ''
GROUP BY s.title_id, paths.team_identifier, paths.identifier
ORDER BY
	array_position($1::bigint[], s.title_id),
	lower(paths.team_identifier),
	lower(paths.identifier)`

func (s *Store) loadSoftwareTitleSigningIdentities(ctx context.Context, titles []SoftwareTitle) error {
	if len(titles) == 0 {
		return nil
	}
	titleIDs := make([]int64, len(titles))
	titleIndex := make(map[int64]int, len(titles))
	for i := range titles {
		titleIDs[i] = titles[i].ID
		titleIndex[titles[i].ID] = i
		titles[i].SigningIdentities = SoftwareSigningIdentityList{
			Items: []SoftwareSigningIdentity{},
		}
	}

	qrows, err := s.db.Pool().Query(ctx, softwareTitleSigningIdentitiesSQL, titleIDs)
	if err != nil {
		return err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[softwareTitleSigningIdentityRow])
	if err != nil {
		return err
	}

	for _, row := range rows {
		i, ok := titleIndex[row.TitleID]
		if !ok {
			continue
		}
		titles[i].SigningIdentities.Items = append(
			titles[i].SigningIdentities.Items,
			SoftwareSigningIdentity{
				Identifier:     row.Identifier,
				TeamIdentifier: row.TeamIdentifier,
				Authorities:    row.Authorities,
				HostsCount:     row.HostsCount,
			},
		)
		titles[i].SigningIdentities.Count++
	}
	return nil
}
