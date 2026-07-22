package clientresources

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type Store struct {
	db      *database.DB
	objects *storage.ObjectStore
}

func NewStore(db *database.DB, objects *storage.ObjectStore) *Store {
	return &Store{db: db, objects: objects}
}

type clientResourcesRow struct {
	ID              int64                  `db:"id"`
	ArchiveObjectID int64                  `db:"archive_object_id"`
	Custom          bool                   `db:"custom"`
	HasBuilder      bool                   `db:"has_builder"`
	BannerObjectID  int64                  `db:"banner_object_id"`
	BannerFit       BannerFit              `db:"banner_fit"`
	BannerFocalX    int                    `db:"banner_focal_x"`
	Links           dbutil.JSONSlice[Link] `db:"links"`
	FooterText      string                 `db:"footer_text"`
	FooterLinks     dbutil.JSONSlice[Link] `db:"footer_links"`
	CreatedAt       time.Time              `db:"created_at"`
	UpdatedAt       time.Time              `db:"updated_at"`
}

const clientResourcesSelectSQL = `SELECT
    cr.id,
    cr.archive_object_id,
    cr.custom,
    cr.banner_object_id IS NOT NULL AS has_builder,
    COALESCE(cr.banner_object_id, 0) AS banner_object_id,
    cr.banner_fit,
    cr.banner_focal_x,
    cr.links,
    cr.footer_text,
    cr.footer_links,
    cr.created_at,
    cr.updated_at
FROM munki_client_resources cr`

func (s *Store) List(
	ctx context.Context,
	params dbutil.ListParams,
) ([]ClientResources, int, error) {
	params = dbutil.NormalizeListParams(params)
	query := dbutil.ListQuery{
		SelectSQL: clientResourcesSelectSQL,
		OrderKeys: map[string]dbutil.OrderExpr{
			"id":         {SQL: "cr.id"},
			"created_at": {SQL: "cr.created_at"},
			"updated_at": {SQL: "cr.updated_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "cr.id"}},
		Params:       params,
	}
	rows, count, err := dbutil.ListWithCount[clientResourcesRow](ctx, s.db.Pool(), query)
	if err != nil {
		return nil, 0, err
	}
	resources := make([]ClientResources, len(rows))
	for i, row := range rows {
		resources[i] = clientResourcesFromRow(row)
	}
	return resources, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*ClientResources, error) {
	return getByID(ctx, s.db.Pool(), id)
}

func getByID(ctx context.Context, q dbutil.Queryer, id int64) (*ClientResources, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := dbutil.GetOne[clientResourcesRow](
		ctx,
		q,
		clientResourcesSelectSQL+"\nWHERE cr.id = $1",
		id,
	)
	if err != nil {
		return nil, err
	}
	resource := clientResourcesFromRow(row)
	return &resource, nil
}

func lockByID(ctx context.Context, tx pgx.Tx, id int64) (*ClientResources, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := dbutil.GetOne[clientResourcesRow](
		ctx,
		tx,
		clientResourcesSelectSQL+"\nWHERE cr.id = $1\nFOR UPDATE OF cr",
		id,
	)
	if err != nil {
		return nil, err
	}
	resource := clientResourcesFromRow(row)
	return &resource, nil
}

type clientResourcesWrite struct {
	archiveObjectID int64
	custom          bool
	builder         *Builder
}

func (s *Store) Create(ctx context.Context, next clientResourcesWrite) (*ClientResources, error) {
	var id int64
	if err := s.db.Pool().QueryRow(ctx, `
INSERT INTO munki_client_resources (
    archive_object_id,
    custom,
    banner_object_id,
    banner_fit,
    banner_focal_x,
    links,
    footer_text,
    footer_links
) VALUES (
    @archive_object_id,
    @custom,
    NULLIF(@banner_object_id, 0),
    @banner_fit,
    @banner_focal_x,
    @links::jsonb,
    @footer_text,
    @footer_links::jsonb
)
RETURNING id`, clientResourcesWriteArgs(next)).Scan(&id); err != nil {
		return nil, dbutil.MutationError(err)
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, next clientResourcesWrite) (*ClientResources, error) {
	var replacedObjectIDs []int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		existing, err := lockByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if next.builder == nil {
			next.builder = existing.Builder
		}
		replacedObjectIDs = replacedClientResourceObjectIDs(existing, next)

		args := clientResourcesWriteArgs(next)
		args["id"] = id
		var updatedID int64
		if err := tx.QueryRow(ctx, `
UPDATE munki_client_resources
SET
    archive_object_id = @archive_object_id,
    custom = @custom,
    banner_object_id = NULLIF(@banner_object_id, 0),
    banner_fit = @banner_fit,
    banner_focal_x = @banner_focal_x,
    links = @links::jsonb,
    footer_text = @footer_text,
    footer_links = @footer_links::jsonb,
    updated_at = now()
WHERE id = @id
RETURNING id`, args).Scan(&updatedID); err != nil {
			return dbutil.MutationError(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	resource, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.objects.DeleteUnreferenced(ctx, replacedObjectIDs...)
	return resource, nil
}

func clientResourcesWriteArgs(next clientResourcesWrite) pgx.NamedArgs {
	builder := Builder{
		BannerFit:   BannerFitHeight,
		Links:       []Link{},
		FooterLinks: []Link{},
	}
	if next.builder != nil {
		builder = *next.builder
	}
	return pgx.NamedArgs{
		"archive_object_id": next.archiveObjectID,
		"custom":            next.custom,
		"banner_object_id":  builder.BannerObjectID,
		"banner_fit":        builder.BannerFit,
		"banner_focal_x":    builder.BannerFocalX,
		"links":             dbutil.JSONSlice[Link](builder.Links),
		"footer_text":       builder.FooterText,
		"footer_links":      dbutil.JSONSlice[Link](builder.FooterLinks),
	}
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	var archiveObjectID int64
	var bannerObjectID *int64
	if err := s.db.Pool().QueryRow(ctx, `
DELETE FROM munki_client_resources
WHERE id = $1
RETURNING archive_object_id, banner_object_id`, id).Scan(&archiveObjectID, &bannerObjectID); err != nil {
		return dbutil.MutationError(err)
	}
	objectIDs := []int64{archiveObjectID}
	if bannerObjectID != nil {
		objectIDs = append(objectIDs, *bannerObjectID)
	}
	s.objects.DeleteUnreferenced(ctx, objectIDs...)
	return nil
}

func replacedClientResourceObjectIDs(
	existing *ClientResources,
	next clientResourcesWrite,
) []int64 {
	var replaced []int64
	if existing.ArchiveObjectID != next.archiveObjectID {
		replaced = append(replaced, existing.ArchiveObjectID)
	}
	if existing.Builder != nil &&
		(next.builder == nil || existing.Builder.BannerObjectID != next.builder.BannerObjectID) {
		replaced = append(replaced, existing.Builder.BannerObjectID)
	}
	return replaced
}

func clientResourcesFromRow(row clientResourcesRow) ClientResources {
	resource := ClientResources{
		ID:              row.ID,
		ArchiveObjectID: row.ArchiveObjectID,
		Custom:          row.Custom,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if row.HasBuilder {
		resource.Builder = &Builder{
			BannerObjectID: row.BannerObjectID,
			BannerFit:      row.BannerFit,
			BannerFocalX:   row.BannerFocalX,
			Links:          row.Links,
			FooterText:     row.FooterText,
			FooterLinks:    row.FooterLinks,
		}
	}
	return resource
}
