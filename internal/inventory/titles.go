package inventory

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) ListTitles(ctx context.Context, params SoftwareTitleListParams) ([]SoftwareTitle, int, error) {
	params.SoftwareSources = dbutil.SplitListValues(params.SoftwareSources)
	whereSQL, args := softwareTitleWhere(params)
	listQuery := softwareTitleListQuery(params.ListParams, whereSQL, args)

	titles, total, err := dbutil.ScanListWithCount(ctx, s.db.Pool(), listQuery, scanSoftwareTitle)
	if err != nil {
		return nil, 0, err
	}
	if err := s.loadSoftwareTitleVersions(ctx, titles); err != nil {
		return nil, 0, err
	}
	return titles, total, nil
}

func (s *Store) GetTitle(ctx context.Context, id int64) (*SoftwareTitle, error) {
	query := softwareTitleSelectSQL + "\nWHERE st.id = $1\nGROUP BY st.id"
	title, err := scanSoftwareTitle(s.db.Pool().QueryRow(ctx, query, id))
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	titles := []SoftwareTitle{title}
	if err := s.loadSoftwareTitleVersions(ctx, titles); err != nil {
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
			OR st.display_name ILIKE ` + search + `
			OR st.bundle_identifier ILIKE ` + search + `
			OR st.extension_for ILIKE ` + search + `
			OR EXISTS (
				SELECT 1 FROM software s
				WHERE s.title_id = st.id AND s.version ILIKE ` + search + `
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
			"name":              {SQL: "lower(st.name)"},
			"source":            {SQL: "lower(st.source)"},
			"hosts_count":       {SQL: "hosts_count"},
			"versions_count":    {SQL: "versions_count"},
			"counts_updated_at": {SQL: "counts_updated_at", NullOrder: dbutil.NullsLast},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(st.name)"}, {SQL: "st.id"}},
		Params:       params,
	}
}

const softwareTitleSelectSQL = `
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
	(CASE WHEN COUNT(hs.last_seen_at) = 0 THEN NULL ELSE MAX(hs.last_seen_at) END)::timestamptz AS counts_updated_at
FROM software_titles st
LEFT JOIN software s ON s.title_id = st.id
LEFT JOIN host_software hs ON hs.software_id = s.id
`

func scanSoftwareTitle(row pgx.Row) (SoftwareTitle, error) {
	var title SoftwareTitle
	err := row.Scan(
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
	)
	title.Browser = browserFor(title.Source, title.ExtensionFor)
	return title, err
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
		titles[i].Versions = append(titles[i].Versions, SoftwareVersion{
			ID:               row.ID,
			Version:          row.Version,
			BundleIdentifier: row.BundleIdentifier,
			HostsCount:       row.HostsCount,
		})
	}
	return nil
}
