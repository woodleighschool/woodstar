package softwaretitles

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
	"github.com/woodleighschool/woodstar/internal/munki/assignments"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

type artifactStore interface {
	GetByID(context.Context, int64) (*artifacts.Artifact, error)
}

type packageStore interface {
	GetByID(context.Context, int64) (*packages.Package, error)
}

type Store struct {
	db        *database.DB
	q         *sqlc.Queries
	artifacts artifactStore
	packages  packageStore
}

func NewStore(db *database.DB, artifacts artifactStore, packages packageStore) *Store {
	return &Store{db: db, q: db.Queries(), artifacts: artifacts, packages: packages}
}

func (s *Store) Create(ctx context.Context, params SoftwareTitleMutation) (*SoftwareTitle, error) {
	var err error
	params = cleanMutation(params)
	params, err = s.normalizeIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	var titleID int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		row, err := qtx.CreateMunkiSoftwareTitle(ctx, sqlc.CreateMunkiSoftwareTitleParams{
			Name:           params.Name,
			Description:    params.Description,
			Category:       params.Category,
			Developer:      params.Developer,
			IconName:       params.IconName,
			IconHash:       params.IconHash,
			IconArtifactID: params.IconArtifactID,
		})
		if err != nil {
			return err
		}
		titleID = row.ID
		return s.replaceTargets(ctx, qtx, titleID, params.Includes, params.ExcludeLabelIDs)
	})
	if err != nil {
		return nil, mapMutationError(err)
	}
	return s.GetByID(ctx, titleID)
}

func (s *Store) Update(ctx context.Context, id int64, params SoftwareTitleMutation) (*SoftwareTitle, error) {
	var err error
	params = cleanMutation(params)
	params, err = s.normalizeIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		row, err := qtx.UpdateMunkiSoftwareTitle(ctx, sqlc.UpdateMunkiSoftwareTitleParams{
			Name:           params.Name,
			Description:    params.Description,
			Category:       params.Category,
			Developer:      params.Developer,
			IconName:       params.IconName,
			IconHash:       params.IconHash,
			IconArtifactID: params.IconArtifactID,
			ID:             id,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.replaceTargets(ctx, qtx, row.ID, params.Includes, params.ExcludeLabelIDs)
	})
	if err != nil {
		return nil, mapMutationError(err)
	}
	return s.GetByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*SoftwareTitle, error) {
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

func (s *Store) Exists(ctx context.Context, id int64) error {
	_, err := s.GetByID(ctx, id)
	return err
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return dbutil.ErrNotFound
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		if err := qtx.DeleteMunkiAssignmentsBySoftware(
			ctx,
			sqlc.DeleteMunkiAssignmentsBySoftwareParams{SoftwareID: id},
		); err != nil {
			return err
		}
		_, err := qtx.DeleteMunkiSoftwareTitle(ctx, sqlc.DeleteMunkiSoftwareTitleParams{ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		}
		return err
	})
}

// DeleteMany removes multiple software titles. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var deleted int
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		if err := qtx.DeleteMunkiAssignmentsBySoftwareIDs(
			ctx,
			sqlc.DeleteMunkiAssignmentsBySoftwareIDsParams{Ids: ids},
		); err != nil {
			return err
		}
		deletedIDs, err := qtx.DeleteMunkiSoftwareTitles(ctx, sqlc.DeleteMunkiSoftwareTitlesParams{Ids: ids})
		if err != nil {
			return err
		}
		deleted = len(deletedIDs)
		return nil
	})
	return deleted, err
}

func (s *Store) List(ctx context.Context, params dbutil.ListParams) ([]SoftwareTitle, int, error) {
	params = dbutil.CleanListParams(params)
	where, args := softwareTitleListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL: softwareTitleSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":       {SQL: "lower(st.name)"},
			"category":   {SQL: "lower(st.category)"},
			"developer":  {SQL: "lower(st.developer)"},
			"updated_at": {SQL: "st.updated_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{
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

func (s *Store) replaceTargets(
	ctx context.Context,
	qtx *sqlc.Queries,
	softwareID int64,
	includes []assignments.AssignmentIncludeMutation,
	excludeLabelIDs []int64,
) error {
	mutations, err := s.normalizeTargets(ctx, softwareID, includes, excludeLabelIDs)
	if err != nil {
		return err
	}
	if err := validateExcludedLabels(ctx, qtx, excludeLabelIDs); err != nil {
		return err
	}
	if err := qtx.DeleteMunkiAssignmentExcludeLabels(
		ctx,
		sqlc.DeleteMunkiAssignmentExcludeLabelsParams{SoftwareID: softwareID},
	); err != nil {
		return err
	}
	if err := qtx.DeleteMunkiAssignmentsBySoftware(
		ctx,
		sqlc.DeleteMunkiAssignmentsBySoftwareParams{SoftwareID: softwareID},
	); err != nil {
		return err
	}
	for _, mutation := range mutations {
		if _, err := qtx.CreateMunkiAssignment(ctx, createAssignmentParams(mutation)); err != nil {
			return err
		}
	}
	if len(excludeLabelIDs) == 0 {
		return nil
	}
	return qtx.InsertMunkiAssignmentExcludeLabels(
		ctx,
		sqlc.InsertMunkiAssignmentExcludeLabelsParams{SoftwareID: softwareID, LabelIds: excludeLabelIDs},
	)
}

func (s *Store) normalizeTargets(
	ctx context.Context,
	softwareID int64,
	includes []assignments.AssignmentIncludeMutation,
	excludeLabelIDs []int64,
) ([]assignments.AssignmentMutation, error) {
	if err := validateExcludeLabelIDs(excludeLabelIDs); err != nil {
		return nil, err
	}
	excludedLabels := labelIDSet(excludeLabelIDs)
	seenIncludeLabels := make(map[int64]struct{}, len(includes))
	mutations := make([]assignments.AssignmentMutation, len(includes))
	for index, include := range includes {
		mutation := cleanAssignmentMutation(include.Mutation(softwareID, int32(index+1)))
		if err := mutation.Validate(); err != nil {
			return nil, err
		}
		if _, ok := seenIncludeLabels[mutation.LabelID]; ok {
			return nil, fmt.Errorf("%w: duplicate include label_id", dbutil.ErrInvalidInput)
		}
		if _, ok := excludedLabels[mutation.LabelID]; ok {
			return nil, fmt.Errorf("%w: label_id is both included and excluded", dbutil.ErrInvalidInput)
		}
		if err := s.validatePinnedPackage(ctx, softwareID, mutation); err != nil {
			return nil, err
		}
		seenIncludeLabels[mutation.LabelID] = struct{}{}
		mutations[index] = mutation
	}
	return mutations, nil
}

func (s *Store) validatePinnedPackage(
	ctx context.Context,
	softwareID int64,
	mutation assignments.AssignmentMutation,
) error {
	if mutation.PackageSelection != assignments.PackageSelectionSpecific {
		return nil
	}
	pkg, err := s.packages.GetByID(ctx, *mutation.PinnedPackageID)
	if err != nil {
		return err
	}
	if pkg.SoftwareID != softwareID {
		return fmt.Errorf("%w: pinned_package_id must belong to software title", dbutil.ErrInvalidInput)
	}
	return nil
}

func validateExcludeLabelIDs(labelIDs []int64) error {
	seen := make(map[int64]struct{}, len(labelIDs))
	for _, labelID := range labelIDs {
		if labelID <= 0 {
			return fmt.Errorf("%w: exclude_label_ids contains an invalid label_id", dbutil.ErrInvalidInput)
		}
		if _, ok := seen[labelID]; ok {
			return fmt.Errorf("%w: duplicate exclude_label_ids entry", dbutil.ErrInvalidInput)
		}
		seen[labelID] = struct{}{}
	}
	return nil
}

func validateExcludedLabels(ctx context.Context, qtx *sqlc.Queries, labelIDs []int64) error {
	if len(labelIDs) == 0 {
		return nil
	}
	builtinExcludeIDs, err := qtx.ListBuiltinLabelIDs(ctx, sqlc.ListBuiltinLabelIDsParams{LabelIds: labelIDs})
	if err != nil {
		return err
	}
	if len(builtinExcludeIDs) > 0 {
		return fmt.Errorf("%w: builtin labels cannot be excluded from Munki software titles", dbutil.ErrInvalidInput)
	}
	return nil
}

func cleanAssignmentMutation(params assignments.AssignmentMutation) assignments.AssignmentMutation {
	params.Action = assignments.AssignmentAction(strings.TrimSpace(string(params.Action)))
	params.PackageSelection = assignments.PackageSelection(strings.TrimSpace(string(params.PackageSelection)))
	return params
}

func createAssignmentParams(params assignments.AssignmentMutation) sqlc.CreateMunkiAssignmentParams {
	return sqlc.CreateMunkiAssignmentParams{
		SoftwareID:       params.SoftwareID,
		Priority:         params.Priority,
		LabelID:          params.LabelID,
		Action:           sqlc.MunkiAssignmentAction(params.Action),
		OptionalInstall:  params.OptionalInstall,
		FeaturedItem:     params.FeaturedItem,
		PackageSelection: sqlc.MunkiPackageSelection(params.PackageSelection),
		PinnedPackageID:  params.PinnedPackageID,
	}
}

func labelIDSet(labelIDs []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(labelIDs))
	for _, labelID := range labelIDs {
		out[labelID] = struct{}{}
	}
	return out
}

func (s *Store) normalizeIcon(ctx context.Context, params SoftwareTitleMutation) (SoftwareTitleMutation, error) {
	if params.IconArtifactID == nil {
		return params, nil
	}
	artifact, err := s.artifacts.GetByID(ctx, *params.IconArtifactID)
	if err != nil {
		return params, err
	}
	if artifact.Kind != artifacts.ArtifactKindIcon {
		return params, fmt.Errorf(
			"%w: icon_artifact_id must reference an icon artifact",
			dbutil.ErrInvalidInput,
		)
	}
	params.IconName = artifact.Location
	params.IconHash = artifact.SHA256
	return params, nil
}

func cleanMutation(params SoftwareTitleMutation) SoftwareTitleMutation {
	params.Name = strings.TrimSpace(params.Name)
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
			OR st.description ILIKE ` + search + `
			OR st.category ILIKE ` + search + `
			OR st.developer ILIKE ` + search + `
		)`)
	}
	return where.Build()
}

func mapMutationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	switch database.SQLState(err) {
	case pgerrcode.ForeignKeyViolation:
		return dbutil.ErrNotFound
	case pgerrcode.UniqueViolation:
		return dbutil.ErrAlreadyExists
	case pgerrcode.InvalidTextRepresentation,
		pgerrcode.NotNullViolation,
		pgerrcode.CheckViolation:
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return err
}

const softwareTitleSelectSQL = `
SELECT
	st.id,
	st.name,
	st.description,
	st.category,
	st.developer,
	st.icon_name,
	st.icon_hash,
	st.icon_artifact_id,
	st.created_at,
	st.updated_at
FROM munki_software_titles st`
