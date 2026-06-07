package software

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
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

type artifactStore interface {
	GetByID(context.Context, int64) (*artifacts.Artifact, error)
}

type packageStore interface {
	GetByID(context.Context, int64) (*packages.Package, error)
	AttachRelations(context.Context, []packages.Package) ([]packages.Package, error)
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

func (s *Store) Create(ctx context.Context, params SoftwareMutation) (*Software, error) {
	var err error
	params = cleanMutation(params)
	params, err = s.normalizeIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	var softwareID int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		row, err := qtx.CreateMunkiSoftware(ctx, sqlc.CreateMunkiSoftwareParams{
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
		softwareID = row.ID
		return s.replaceTargets(ctx, qtx, softwareID, params.Targets)
	})
	if err != nil {
		return nil, mapMutationError(err)
	}
	return s.GetByID(ctx, softwareID)
}

func (s *Store) Update(ctx context.Context, id int64, params SoftwareMutation) (*Software, error) {
	var err error
	params = cleanMutation(params)
	params, err = s.normalizeIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		row, err := qtx.UpdateMunkiSoftware(ctx, sqlc.UpdateMunkiSoftwareParams{
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
		return s.replaceTargets(ctx, qtx, row.ID, params.Targets)
	})
	if err != nil {
		return nil, mapMutationError(err)
	}
	return s.GetByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Software, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.GetMunkiSoftwareByID(ctx, sqlc.GetMunkiSoftwareByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	software := softwareFromSQLC(row)
	return &software, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return dbutil.ErrNotFound
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		if err := qtx.DeleteMunkiSoftwareTargetsBySoftware(
			ctx,
			sqlc.DeleteMunkiSoftwareTargetsBySoftwareParams{SoftwareID: id},
		); err != nil {
			return err
		}
		_, err := qtx.DeleteMunkiSoftwareByID(ctx, sqlc.DeleteMunkiSoftwareByIDParams{ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		}
		return err
	})
}

// DeleteMany removes multiple software rows. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var deleted int
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		if err := qtx.DeleteMunkiSoftwareTargetsBySoftwareIDs(
			ctx,
			sqlc.DeleteMunkiSoftwareTargetsBySoftwareIDsParams{Ids: ids},
		); err != nil {
			return err
		}
		deletedIDs, err := qtx.DeleteMunkiSoftwareByIDs(ctx, sqlc.DeleteMunkiSoftwareByIDsParams{Ids: ids})
		if err != nil {
			return err
		}
		deleted = len(deletedIDs)
		return nil
	})
	return deleted, err
}

func (s *Store) List(ctx context.Context, params dbutil.ListParams) ([]Software, int, error) {
	params = dbutil.CleanListParams(params)
	where, args := softwareListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL: softwareSelectSQL,
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
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.MunkiSoftware])
	if err != nil {
		return nil, 0, err
	}
	software := make([]Software, len(records))
	for i, row := range records {
		software[i] = softwareFromSQLC(row)
	}
	return software, count, nil
}

func (s *Store) normalizeIcon(ctx context.Context, params SoftwareMutation) (SoftwareMutation, error) {
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

func cleanMutation(params SoftwareMutation) SoftwareMutation {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Category = strings.TrimSpace(params.Category)
	params.Developer = strings.TrimSpace(params.Developer)
	params.IconName = strings.TrimSpace(params.IconName)
	params.IconHash = strings.TrimSpace(params.IconHash)
	params.Targets = normalizeSoftwareTargets(params.Targets)
	return params
}

func softwareFromSQLC(row sqlc.MunkiSoftware) Software {
	return Software{
		ID:             row.ID,
		Name:           row.Name,
		DisplayName:    row.Name,
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

func softwareListWhere(params dbutil.ListParams) (string, []any) {
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

const softwareSelectSQL = `
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
FROM munki_software st`
