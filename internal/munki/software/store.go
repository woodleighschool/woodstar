package software

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type objectStore interface {
	GetByID(context.Context, int64) (*storage.Object, error)
	DeleteUnreferenced(context.Context, ...int64) error
}

type packageStore interface {
	GetByID(context.Context, int64) (*packages.Package, error)
	AttachRelations(context.Context, []packages.Package) ([]packages.Package, error)
}

type Store struct {
	db       *database.DB
	q        *sqlc.Queries
	objects  objectStore
	packages packageStore
}

func NewStore(db *database.DB, objects objectStore, packages packageStore) *Store {
	return &Store{db: db, q: db.Queries(), objects: objects, packages: packages}
}

func (s *Store) Create(ctx context.Context, params Mutation) (*Software, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	params, err := s.normalizeIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	var softwareID int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		row, err := qtx.CreateMunkiSoftware(ctx, sqlc.CreateMunkiSoftwareParams{
			Name:         params.Name,
			Description:  params.Description,
			Category:     params.Category,
			Developer:    params.Developer,
			IconObjectID: params.IconObjectID,
		})
		if err != nil {
			return err
		}
		softwareID = row.ID
		return s.replaceTargets(ctx, qtx, softwareID, params.Targets)
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return s.GetByID(ctx, softwareID)
}

func (s *Store) Update(ctx context.Context, id int64, params Mutation) (*Software, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	params, err := s.normalizeIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	var oldIconObjectID *int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		existing, err := qtx.GetMunkiSoftwareByID(ctx, sqlc.GetMunkiSoftwareByIDParams{ID: id})
		if err != nil {
			return dbutil.GetError(err)
		}
		oldIconObjectID = existing.IconObjectID
		row, err := qtx.UpdateMunkiSoftware(ctx, sqlc.UpdateMunkiSoftwareParams{
			Name:         params.Name,
			Description:  params.Description,
			Category:     params.Category,
			Developer:    params.Developer,
			IconObjectID: params.IconObjectID,
			ID:           id,
		})
		if err != nil {
			return err
		}
		return s.replaceTargets(ctx, qtx, row.ID, params.Targets)
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	if err := s.objects.DeleteUnreferenced(
		ctx,
		replacedObjectIDs(oldIconObjectID, params.IconObjectID)...); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Software, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.GetMunkiSoftwareByID(ctx, sqlc.GetMunkiSoftwareByIDParams{ID: id})
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	software := softwareFromSQLC(row)
	return &software, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return dbutil.ErrNotFound
	}
	var objectIDs []int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		var err error
		objectIDs, err = qtx.ListMunkiSoftwareObjectIDsByIDs(
			ctx,
			sqlc.ListMunkiSoftwareObjectIDsByIDsParams{Ids: []int64{id}},
		)
		if err != nil {
			return err
		}
		if err := qtx.DeleteMunkiSoftwareTargetsBySoftware(
			ctx,
			sqlc.DeleteMunkiSoftwareTargetsBySoftwareParams{SoftwareID: id},
		); err != nil {
			return err
		}
		_, err = qtx.DeleteMunkiSoftwareByID(ctx, sqlc.DeleteMunkiSoftwareByIDParams{ID: id})
		return err
	})
	if err != nil {
		return dbutil.MutationError(err)
	}
	return s.objects.DeleteUnreferenced(ctx, objectIDs...)
}

// DeleteMany removes multiple software rows. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var deleted int
	var objectIDs []int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		rows, err := qtx.ListMunkiSoftwareObjectIDsByIDs(
			ctx,
			sqlc.ListMunkiSoftwareObjectIDsByIDsParams{Ids: ids},
		)
		if err != nil {
			return err
		}
		objectIDs = append(objectIDs, rows...)
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
	if err != nil {
		return deleted, err
	}
	return deleted, s.objects.DeleteUnreferenced(ctx, objectIDs...)
}

func (s *Store) List(ctx context.Context, params dbutil.ListParams) ([]Software, int, error) {
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
	records, count, err := dbutil.ListWithCount[sqlc.MunkiSoftware](ctx, s.db.Pool(), listQuery)
	if err != nil {
		return nil, 0, err
	}
	software := make([]Software, len(records))
	for i, row := range records {
		software[i] = softwareFromSQLC(row)
	}
	return software, count, nil
}

func (s *Store) normalizeIcon(ctx context.Context, params Mutation) (Mutation, error) {
	if params.IconObjectID == nil {
		return params, nil
	}
	obj, err := s.objects.GetByID(ctx, *params.IconObjectID)
	if err != nil {
		return params, err
	}
	if obj.Prefix != munkiupload.IconObjectPrefix {
		return params, fmt.Errorf(
			"%w: icon_object_id must reference an icon",
			dbutil.ErrInvalidInput,
		)
	}
	if !obj.Available() {
		return params, fmt.Errorf("%w: icon_object_id must reference an uploaded icon", dbutil.ErrInvalidInput)
	}
	return params, nil
}

// SetIcon points software at an icon storage object.
func (s *Store) SetIcon(ctx context.Context, softwareID, objectID int64) error {
	obj, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return err
	}
	if obj.Prefix != munkiupload.IconObjectPrefix {
		return fmt.Errorf("%w: icon_object_id must reference an icon", dbutil.ErrInvalidInput)
	}
	if !obj.Available() {
		return fmt.Errorf("%w: icon_object_id must reference an uploaded icon", dbutil.ErrInvalidInput)
	}
	var oldIconObjectID *int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		qtx := s.q.WithTx(tx)
		existing, err := qtx.GetMunkiSoftwareByID(ctx, sqlc.GetMunkiSoftwareByIDParams{ID: softwareID})
		if err != nil {
			return dbutil.GetError(err)
		}
		oldIconObjectID = existing.IconObjectID
		rows, err := qtx.SetMunkiSoftwareIconObject(ctx, sqlc.SetMunkiSoftwareIconObjectParams{
			ID:       softwareID,
			ObjectID: &objectID,
		})
		if err != nil {
			return dbutil.MutationError(err)
		}
		if rows == 0 {
			return dbutil.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.objects.DeleteUnreferenced(ctx, replacedObjectIDs(oldIconObjectID, &objectID)...)
}

func (m Mutation) validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	return nil
}

func replacedObjectIDs(oldID, newID *int64) []int64 {
	if oldID == nil {
		return nil
	}
	if newID != nil && *oldID == *newID {
		return nil
	}
	return []int64{*oldID}
}

func softwareFromSQLC(row sqlc.MunkiSoftware) Software {
	return Software{
		ID:           row.ID,
		Name:         row.Name,
		Description:  row.Description,
		Category:     row.Category,
		Developer:    row.Developer,
		IconObjectID: row.IconObjectID,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
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

const softwareSelectSQL = `
SELECT
	st.id,
	st.name,
	st.description,
	st.category,
	st.developer,
	st.icon_object_id,
	st.created_at,
	st.updated_at
FROM munki_software st`
