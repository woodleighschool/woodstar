package software

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) ListTitles(ctx context.Context, params SoftwareTitleListParams) ([]SoftwareTitle, int, error) {
	params.SoftwareSources = dbutil.SplitListValues(params.SoftwareSources)
	whereSQL, args := softwareTitleWhere(params)

	countSQL := `SELECT count(*) FROM software_titles st`
	if whereSQL != "" {
		countSQL += "\n" + whereSQL
	}
	var total int
	if err := s.db.Pool().QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query, args, err := softwareTitleListQuery(params.ListParams, whereSQL, args)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
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

func (s *Store) GetTitle(ctx context.Context, id int64) (*SoftwareTitle, error) {
	query, args, err := softwareTitleListQuery(
		dbutil.ListParams{PageSize: 1},
		"WHERE st.id = $1",
		[]any{id},
	)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	titles, err := scanSoftwareTitles(rows)
	if err != nil {
		return nil, err
	}
	if len(titles) == 0 {
		return nil, dbutil.ErrNotFound
	}
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

func softwareTitleListQuery(params dbutil.ListParams, whereSQL string, args []any) (string, []any, error) {
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
	}.Build()
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
	MAX(hs.last_seen_at) AS counts_updated_at
FROM software_titles st
LEFT JOIN software s ON s.title_id = st.id
LEFT JOIN host_software hs ON hs.software_id = s.id
`

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
		title.Browser = browserFor(title.Source, title.ExtensionFor)
		titles = append(titles, title)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return titles, nil
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

	rows, err := s.db.Pool().Query(ctx, `
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
ORDER BY array_position($1::bigint[], s.title_id), lower(s.version), s.id`, titleIDs)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var titleID int64
		var version SoftwareVersion
		if err := rows.Scan(
			&titleID,
			&version.ID,
			&version.Version,
			&version.BundleIdentifier,
			&version.HostsCount,
		); err != nil {
			return err
		}
		i, ok := titleIndex[titleID]
		if !ok {
			continue
		}
		titles[i].Versions = append(titles[i].Versions, version)
	}
	return rows.Err()
}
