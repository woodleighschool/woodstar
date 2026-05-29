package software

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
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
	row, err := s.q.GetSoftwareTitleSummary(ctx, sqlc.GetSoftwareTitleSummaryParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	titles := []SoftwareTitle{softwareTitleFromSQLC(row)}
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

func softwareTitleFromSQLC(row sqlc.GetSoftwareTitleSummaryRow) SoftwareTitle {
	title := SoftwareTitle{
		ID:               row.ID,
		Name:             row.Name,
		DisplayName:      row.DisplayName,
		Source:           row.Source,
		ExtensionFor:     row.ExtensionFor,
		BundleIdentifier: row.BundleIdentifier,
		Vendor:           row.Vendor,
		HostsCount:       row.HostsCount,
		VersionsCount:    row.VersionsCount,
		CountsUpdatedAt:  timePtr(row.CountsUpdatedAt),
	}
	title.Browser = browserFor(title.Source, title.ExtensionFor)
	return title
}

func timePtr(value any) *time.Time {
	switch t := value.(type) {
	case nil:
		return nil
	case time.Time:
		return &t
	case *time.Time:
		return t
	default:
		return nil
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

	rows, err := s.q.ListSoftwareTitleVersions(ctx, sqlc.ListSoftwareTitleVersionsParams{TitleIds: titleIDs})
	if err != nil {
		return err
	}

	for _, row := range rows {
		var version SoftwareVersion
		version.ID = row.ID
		version.Version = row.Version
		version.BundleIdentifier = row.BundleIdentifier
		version.HostsCount = row.HostsCount
		i, ok := titleIndex[row.TitleID]
		if !ok {
			continue
		}
		titles[i].Versions = append(titles[i].Versions, version)
	}
	return nil
}
