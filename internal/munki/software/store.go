package software

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// IconObjectPrefix namespaces software icon objects in storage.
const IconObjectPrefix = "munki/icons"

type objectStore interface {
	GetByID(ctx context.Context, objectID int64) (*storage.Object, error)
	Delete(ctx context.Context, objectID int64) error
}

type packageStore interface {
	GetByID(ctx context.Context, packageID int64) (*packages.Package, error)
	PackagesByID(ctx context.Context, packageIDs []int64) ([]packages.Package, error)
}

type Store struct {
	db       *database.DB
	objects  objectStore
	packages packageStore
}

func NewStore(db *database.DB, objects objectStore, packages packageStore) *Store {
	return &Store{db: db, objects: objects, packages: packages}
}

func (s *Store) Create(ctx context.Context, params CreateMutation) (*Software, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, err
	}
	if err := s.validateIcon(ctx, params.IconObjectID); err != nil {
		return nil, err
	}
	write := newSoftwareWrite(params)
	var id int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
INSERT INTO munki_software (
	name,
	display_name,
	description,
	category,
	developer,
	icon_object_id
) VALUES (
	@name,
	@display_name,
	@description,
	@category,
	@developer,
	@icon_object_id
)
RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
			return dbutil.MutationError(err)
		}
		return s.replaceTargets(ctx, tx, id, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, params UpdateMutation) (*Software, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	if err := s.validateIcon(ctx, params.IconObjectID); err != nil {
		return nil, err
	}
	var oldIconObjectID *int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		existing, err := getSoftwareByID(ctx, tx, id)
		if err != nil {
			return err
		}
		params.normalize(existing.Name)
		oldIconObjectID = existing.IconObjectID
		write := newSoftwareUpdateWrite(params)
		write.ID = id
		var updatedID int64
		if err := tx.QueryRow(ctx, `
UPDATE munki_software
SET
	display_name = @display_name,
	description = @description,
	category = @category,
	developer = @developer,
	icon_object_id = @icon_object_id,
	updated_at = now()
WHERE id = @id
RETURNING id`, pgx.StructArgs(write)).Scan(&updatedID); err != nil {
			return dbutil.MutationError(err)
		}
		return s.replaceTargets(ctx, tx, id, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	if err := deleteObjects(ctx, s.objects, replacedObjectID(oldIconObjectID, params.IconObjectID)...); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Software, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	return getSoftwareByID(ctx, s.db.Pool(), id)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
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
	return deleteObjects(ctx, s.objects, objectIDs...)
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
	return deleted, deleteObjects(ctx, s.objects, objectIDs...)
}

func (s *Store) List(ctx context.Context, params dbutil.ListParams) ([]Software, int, error) {
	params = dbutil.NormalizeListParams(params)
	where, args := softwareListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL:    softwareSelectSQL(),
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

func (s *Store) validateIcon(ctx context.Context, objectID *int64) error {
	if objectID == nil {
		return nil
	}
	return s.requireIcon(ctx, *objectID)
}

// SetIcon points software at an icon storage object.
func (s *Store) SetIcon(ctx context.Context, softwareID, objectID int64) error {
	if err := s.requireIcon(ctx, objectID); err != nil {
		return err
	}
	var oldIconObjectID *int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		existing, err := getSoftwareByID(ctx, tx, softwareID)
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
	return deleteObjects(ctx, s.objects, replacedObjectID(oldIconObjectID, &objectID)...)
}

func (s *Store) requireIcon(ctx context.Context, objectID int64) error {
	object, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return err
	}
	if object.Prefix != IconObjectPrefix {
		return fmt.Errorf("%w: icon_object_id must reference an icon", dbutil.ErrInvalidInput)
	}
	if !object.Available() {
		return fmt.Errorf("%w: icon_object_id must reference an uploaded icon", dbutil.ErrInvalidInput)
	}
	if !supportedIconContentType(object.ContentType) {
		return fmt.Errorf("%w: icon must be a PNG, JPEG, WebP, or ICNS image", dbutil.ErrInvalidInput)
	}
	return nil
}

func supportedIconContentType(contentType string) bool {
	detected := mimetype.Lookup(contentType)
	if detected == nil {
		return false
	}
	return detected.Is("image/png") ||
		detected.Is("image/jpeg") ||
		detected.Is("image/webp") ||
		detected.Is("image/x-icns")
}

func deleteObjects(ctx context.Context, objects objectStore, ids ...int64) error {
	for _, id := range ids {
		if err := objects.Delete(ctx, id); err != nil &&
			!errors.Is(err, dbutil.ErrConflict) &&
			!errors.Is(err, dbutil.ErrNotFound) {
			return err
		}
	}
	return nil
}

func replacedObjectID(oldID, newID *int64) []int64 {
	if oldID == nil || newID != nil && *oldID == *newID {
		return nil
	}
	return []int64{*oldID}
}

func softwareOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":         {SQL: "lower(st.name)"},
		"display_name": {SQL: "lower(st.display_name)"},
		"category":     {SQL: "lower(st.category)"},
		"developer":    {SQL: "lower(st.developer)"},
		"updated_at":   {SQL: "st.updated_at"},
	}
}

func softwareListWhere(params dbutil.ListParams) (string, []any) {
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

type softwareWrite struct {
	ID           int64   `db:"id"`
	Name         string  `db:"name"`
	DisplayName  *string `db:"display_name"`
	Description  string  `db:"description"`
	Category     string  `db:"category"`
	Developer    string  `db:"developer"`
	IconObjectID *int64  `db:"icon_object_id"`
}

func newSoftwareWrite(params CreateMutation) softwareWrite {
	return softwareWrite{
		Name:         params.Name,
		DisplayName:  dbutil.NullString(params.DisplayName),
		Description:  params.Description,
		Category:     params.Category,
		Developer:    params.Developer,
		IconObjectID: params.IconObjectID,
	}
}

func newSoftwareUpdateWrite(params UpdateMutation) softwareWrite {
	return softwareWrite{
		DisplayName:  dbutil.NullString(params.DisplayName),
		Description:  params.Description,
		Category:     params.Category,
		Developer:    params.Developer,
		IconObjectID: params.IconObjectID,
	}
}

func softwareSelectSQL() string {
	return `
SELECT
	st.id,
	st.name,
	st.display_name,
	st.description,
	st.category,
	st.developer,
	st.icon_object_id,
	icon_obj.filename AS icon_filename,
	icon_obj.size_bytes AS icon_size_bytes,
	icon_obj.sha256 AS icon_sha256,
	st.created_at,
	st.updated_at
FROM munki_software st
LEFT JOIN storage_objects icon_obj ON icon_obj.id = st.icon_object_id`
}

type softwareRow struct {
	ID            int64     `db:"id"`
	Name          string    `db:"name"`
	DisplayName   *string   `db:"display_name"`
	Description   string    `db:"description"`
	Category      string    `db:"category"`
	Developer     string    `db:"developer"`
	IconObjectID  *int64    `db:"icon_object_id"`
	IconFilename  *string   `db:"icon_filename"`
	IconSizeBytes *int64    `db:"icon_size_bytes"`
	IconSHA256    *string   `db:"icon_sha256"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

func getSoftwareByID(ctx context.Context, q dbutil.Queryer, id int64) (*Software, error) {
	row, err := dbutil.GetOne[softwareRow](ctx, q, softwareSelectSQL()+"\nWHERE st.id = $1", id)
	if err != nil {
		return nil, err
	}
	software := softwareFromRow(row)
	return &software, nil
}

func softwareFromRow(row softwareRow) Software {
	software := Software{
		ID:           row.ID,
		Name:         row.Name,
		DisplayName:  row.DisplayName,
		Description:  row.Description,
		Category:     row.Category,
		Developer:    row.Developer,
		IconObjectID: row.IconObjectID,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
	if row.IconObjectID != nil && row.IconFilename != nil {
		software.IconFile = &IconFile{
			Filename:  *row.IconFilename,
			SizeBytes: valueOrZero(row.IconSizeBytes),
			SHA256:    stringOrEmpty(row.IconSHA256),
		}
	}
	return software
}

func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
