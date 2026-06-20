package software

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
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
	objects  objectStore
	packages packageStore
}

func NewStore(db *database.DB, objects objectStore, packages packageStore) *Store {
	return &Store{db: db, objects: objects, packages: packages}
}

func (s *Store) Create(ctx context.Context, params Mutation) (*Software, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	params, err := s.normalizeIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	write := newSoftwareWrite(params)
	var id int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, insertSoftwareSQL, pgx.StructArgs(write)).Scan(&id); err != nil {
			return dbutil.MutationError(err)
		}
		return s.replaceTargets(ctx, tx, id, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
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
		existing, err := dbutil.GetOne[softwareRow](ctx, tx, softwareSelectSQL+"\nWHERE st.id = $1", id)
		if err != nil {
			return err
		}
		oldIconObjectID = existing.IconObjectID
		write := newSoftwareWrite(params)
		write.ID = id
		var updatedID int64
		if err := tx.QueryRow(ctx, updateSoftwareSQL, pgx.StructArgs(write)).Scan(&updatedID); err != nil {
			return dbutil.MutationError(err)
		}
		return s.replaceTargets(ctx, tx, id, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	if err := s.objects.DeleteUnreferenced(
		ctx,
		storage.ReplacedObjectIDs(oldIconObjectID, params.IconObjectID)...); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Software, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := dbutil.GetOne[softwareRow](ctx, s.db.Pool(), softwareSelectSQL+"\nWHERE st.id = $1", id)
	if err != nil {
		return nil, err
	}
	sw := softwareFromRow(row)
	return &sw, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return dbutil.ErrNotFound
	}
	var objectIDs []int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var err error
		objectIDs, err = softwareObjectIDs(ctx, tx, []int64{id})
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM munki_software WHERE id = $1`, id)
		if err != nil {
			return dbutil.DeleteConflict(err, "Munki software is still referenced")
		}
		if tag.RowsAffected() == 0 {
			return dbutil.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return err
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
		var err error
		objectIDs, err = softwareObjectIDs(ctx, tx, ids)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `DELETE FROM munki_software WHERE id = ANY($1::bigint[]) RETURNING id`, ids)
		if err != nil {
			return dbutil.DeleteConflict(err, "Munki software is still referenced")
		}
		deletedIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return dbutil.DeleteConflict(err, "Munki software is still referenced")
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
		SelectSQL:    softwareSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    softwareOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(st.name)"}, {SQL: "st.id"}},
		Params:       params,
	}
	rows, count, err := dbutil.ListWithCount[softwareRow](ctx, s.db.Pool(), listQuery)
	if err != nil {
		return nil, 0, err
	}
	software := make([]Software, len(rows))
	for i, row := range rows {
		software[i] = softwareFromRow(row)
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
		existing, err := dbutil.GetOne[softwareRow](ctx, tx, softwareSelectSQL+"\nWHERE st.id = $1", softwareID)
		if err != nil {
			return err
		}
		oldIconObjectID = existing.IconObjectID
		tag, err := tx.Exec(
			ctx,
			`UPDATE munki_software SET icon_object_id = $2, updated_at = now() WHERE id = $1`,
			softwareID,
			&objectID,
		)
		if err != nil {
			return dbutil.MutationError(err)
		}
		if tag.RowsAffected() == 0 {
			return dbutil.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.objects.DeleteUnreferenced(ctx, storage.ReplacedObjectIDs(oldIconObjectID, &objectID)...)
}

func (m Mutation) validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	return nil
}

func softwareOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":       {SQL: "lower(st.name)"},
		"category":   {SQL: "lower(st.category)"},
		"developer":  {SQL: "lower(st.developer)"},
		"updated_at": {SQL: "st.updated_at"},
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

func softwareObjectIDs(ctx context.Context, q dbutil.Queryer, ids []int64) ([]int64, error) {
	rows, err := q.Query(ctx, `
		SELECT DISTINCT refs.object_id::bigint AS object_id
		FROM munki_software s
		LEFT JOIN munki_packages p ON p.software_id = s.id
		CROSS JOIN LATERAL unnest(
			array_remove(ARRAY[s.icon_object_id, p.installer_object_id], NULL)::bigint[]
		) AS refs(object_id)
		WHERE s.id = ANY($1::bigint[])
		  AND refs.object_id IS NOT NULL`, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

type softwareRow struct {
	ID           int64     `db:"id"`
	Name         string    `db:"name"`
	Description  string    `db:"description"`
	Category     string    `db:"category"`
	Developer    string    `db:"developer"`
	IconObjectID *int64    `db:"icon_object_id"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func softwareFromRow(row softwareRow) Software {
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

type softwareWrite struct {
	ID           int64  `db:"id"`
	Name         string `db:"name"`
	Description  string `db:"description"`
	Category     string `db:"category"`
	Developer    string `db:"developer"`
	IconObjectID *int64 `db:"icon_object_id"`
}

func newSoftwareWrite(params Mutation) softwareWrite {
	return softwareWrite{
		Name:         params.Name,
		Description:  params.Description,
		Category:     params.Category,
		Developer:    params.Developer,
		IconObjectID: params.IconObjectID,
	}
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

const insertSoftwareSQL = `
INSERT INTO munki_software (
	name,
	description,
	category,
	developer,
	icon_object_id
) VALUES (
	@name,
	@description,
	@category,
	@developer,
	@icon_object_id
)
RETURNING id`

const updateSoftwareSQL = `
UPDATE munki_software
SET
	name = @name,
	description = @description,
	category = @category,
	developer = @developer,
	icon_object_id = @icon_object_id,
	updated_at = now()
WHERE id = @id
RETURNING id`
