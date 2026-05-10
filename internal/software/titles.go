package software

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/store"
)

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
		return nil, store.ErrNotFound
	}
	if err := s.loadSoftwareTitleVersions(ctx, titles); err != nil {
		return nil, err
	}
	return &titles[0], nil
}

func cleanSoftwareTitleListParams(params SoftwareTitleListParams) SoftwareTitleListParams {
	params.ListParams = store.CleanListParams(params.ListParams)
	params.SoftwareSources = store.SplitListValues(params.SoftwareSources)
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
	return store.OrderBy(
		store.CleanListParams(store.ListParams{OrderKey: orderKey, OrderDirection: direction}),
		map[string]store.OrderExpr{
			"name":              {SQL: "lower(st.name)"},
			"source":            {SQL: "lower(st.source)"},
			"hosts_count":       {SQL: "hosts_count"},
			"versions_count":    {SQL: "versions_count"},
			"counts_updated_at": {SQL: "counts_updated_at", NullsLast: true},
		},
		[]store.OrderExpr{{SQL: "lower(st.name)"}, {SQL: "st.id"}},
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
