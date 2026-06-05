package munki

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) CreateSoftwareTitle(ctx context.Context, params SoftwareTitleMutation) (*SoftwareTitle, error) {
	var err error
	params = cleanSoftwareTitleMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	params, err = s.normalizeSoftwareTitleIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.CreateMunkiSoftwareTitle(ctx, sqlc.CreateMunkiSoftwareTitleParams{
		Name:           params.Name,
		DisplayName:    params.DisplayName,
		Description:    params.Description,
		Category:       params.Category,
		Developer:      params.Developer,
		IconName:       params.IconName,
		IconHash:       params.IconHash,
		IconArtifactID: params.IconArtifactID,
	})
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	title := softwareTitleFromSQLC(row)
	return &title, nil
}

func (s *Store) UpdateSoftwareTitle(
	ctx context.Context,
	id int64,
	params SoftwareTitleMutation,
) (*SoftwareTitle, error) {
	var err error
	params = cleanSoftwareTitleMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	params, err = s.normalizeSoftwareTitleIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpdateMunkiSoftwareTitle(ctx, sqlc.UpdateMunkiSoftwareTitleParams{
		Name:           params.Name,
		DisplayName:    params.DisplayName,
		Description:    params.Description,
		Category:       params.Category,
		Developer:      params.Developer,
		IconName:       params.IconName,
		IconHash:       params.IconHash,
		IconArtifactID: params.IconArtifactID,
		ID:             id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	title := softwareTitleFromSQLC(row)
	return &title, nil
}

func (s *Store) GetSoftwareTitle(ctx context.Context, id int64) (*SoftwareTitle, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.GetMunkiSoftwareTitleByID(ctx, sqlc.GetMunkiSoftwareTitleByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	title := softwareTitleFromSQLC(row)
	return &title, nil
}

func (s *Store) GetSoftwareTitleByName(ctx context.Context, name string) (*SoftwareTitle, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.GetMunkiSoftwareTitleByName(ctx, sqlc.GetMunkiSoftwareTitleByNameParams{Name: name})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	title := softwareTitleFromSQLC(row)
	return &title, nil
}

func (s *Store) LoadSoftwareTitleDetail(ctx context.Context, id int64) (*SoftwareTitleDetail, error) {
	title, err := s.GetSoftwareTitle(ctx, id)
	if err != nil {
		return nil, err
	}
	packages, _, err := s.ListPackages(ctx, PackageListParams{
		ListParams: dbutil.ListParams{PageSize: 1000},
		SoftwareID: id,
	})
	if err != nil {
		return nil, err
	}
	assignments, _, err := s.ListAssignments(ctx, AssignmentListParams{
		ListParams: dbutil.ListParams{PageSize: 1000, Sort: "priority.asc"},
		SoftwareID: id,
	})
	if err != nil {
		return nil, err
	}
	return &SoftwareTitleDetail{
		SoftwareTitle: *title,
		Packages:      packages,
		Assignments:   assignments,
	}, nil
}

func (s *Store) ListSoftwareTitles(ctx context.Context, params dbutil.ListParams) ([]SoftwareTitle, int, error) {
	params = dbutil.CleanListParams(params)
	where, args := softwareTitleListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL: softwareTitleSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":       {SQL: "lower(COALESCE(NULLIF(st.display_name, ''), st.name))"},
			"category":   {SQL: "lower(st.category)"},
			"developer":  {SQL: "lower(st.developer)"},
			"updated_at": {SQL: "st.updated_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{
			{SQL: "lower(COALESCE(NULLIF(st.display_name, ''), st.name))"},
			{SQL: "lower(st.name)"},
			{SQL: "st.id"},
		},
		Params: params,
	}
	var count int
	countSQL, countArgs := listQuery.BuildCount()
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := listQuery.Build()
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.MunkiSoftwareTitle])
	if err != nil {
		return nil, 0, err
	}
	titles := make([]SoftwareTitle, len(records))
	for i, row := range records {
		titles[i] = softwareTitleFromSQLC(row)
	}
	return titles, count, nil
}

func (s *Store) normalizeSoftwareTitleIcon(
	ctx context.Context,
	params SoftwareTitleMutation,
) (SoftwareTitleMutation, error) {
	if params.IconArtifactID == nil {
		return params, nil
	}
	artifact, err := s.GetArtifact(ctx, *params.IconArtifactID)
	if err != nil {
		return params, err
	}
	if artifact.Kind != ArtifactKindIcon {
		return params, fmt.Errorf(
			"%w: icon_artifact_id must reference an icon artifact",
			dbutil.ErrInvalidInput,
		)
	}
	params.IconName = artifact.Location
	params.IconHash = artifact.SHA256
	return params, nil
}

func cleanSoftwareTitleMutation(params SoftwareTitleMutation) SoftwareTitleMutation {
	params.Name = strings.TrimSpace(params.Name)
	params.DisplayName = strings.TrimSpace(params.DisplayName)
	if params.DisplayName == "" {
		params.DisplayName = params.Name
	}
	params.Description = strings.TrimSpace(params.Description)
	params.Category = strings.TrimSpace(params.Category)
	params.Developer = strings.TrimSpace(params.Developer)
	params.IconName = strings.TrimSpace(params.IconName)
	params.IconHash = strings.TrimSpace(params.IconHash)
	return params
}

func softwareTitleFromSQLC(row sqlc.MunkiSoftwareTitle) SoftwareTitle {
	return SoftwareTitle{
		ID:             row.ID,
		Name:           row.Name,
		DisplayName:    row.DisplayName,
		Description:    row.Description,
		Category:       row.Category,
		Developer:      row.Developer,
		IconName:       row.IconName,
		IconHash:       row.IconHash,
		IconArtifactID: row.IconArtifactID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func softwareTitleListWhere(params dbutil.ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			st.name ILIKE ` + search + `
			OR st.display_name ILIKE ` + search + `
			OR st.description ILIKE ` + search + `
			OR st.category ILIKE ` + search + `
			OR st.developer ILIKE ` + search + `
		)`)
	}
	return where.Build()
}

const softwareTitleSelectSQL = `
SELECT
	st.id,
	st.name,
	st.display_name,
	st.description,
	st.category,
	st.developer,
	st.icon_name,
	st.icon_hash,
	st.icon_artifact_id,
	st.created_at,
	st.updated_at
FROM munki_software_titles st`
