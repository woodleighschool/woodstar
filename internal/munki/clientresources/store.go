package clientresources

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/postgres"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type Store struct {
	pool    *pgxpool.Pool
	objects *storage.ObjectStore
}

func NewStore(pool *pgxpool.Pool, objects *storage.ObjectStore) *Store {
	return &Store{pool: pool, objects: objects}
}

type clientResourcesRow struct {
	ID              int64                    `db:"id"`
	ArchiveObjectID int64                    `db:"archive_object_id"`
	Custom          bool                     `db:"custom"`
	HasBuilder      bool                     `db:"has_builder"`
	BannerObjectID  int64                    `db:"banner_object_id"`
	BannerFit       BannerFit                `db:"banner_fit"`
	BannerFocalX    int                      `db:"banner_focal_x"`
	Links           postgres.JSONSlice[Link] `db:"links"`
	FooterText      string                   `db:"footer_text"`
	FooterLinks     postgres.JSONSlice[Link] `db:"footer_links"`
	CreatedAt       time.Time                `db:"created_at"`
	UpdatedAt       time.Time                `db:"updated_at"`
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
	params listing.Params,
) ([]ClientResources, int, error) {
	params = listing.Normalize(params)
	query := postgres.ListQuery{
		SelectSQL: clientResourcesSelectSQL,
		OrderKeys: map[string]postgres.OrderExpr{
			"id":         {SQL: "cr.id"},
			"created_at": {SQL: "cr.created_at"},
			"updated_at": {SQL: "cr.updated_at"},
		},
		DefaultOrder: []postgres.OrderExpr{{SQL: "cr.id"}},
		Params:       params,
	}
	rows, count, err := postgres.ListWithCount[clientResourcesRow](ctx, s.pool, query)
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
	return getByID(ctx, s.pool, id)
}

func getByID(ctx context.Context, q postgres.Queryer, id int64) (*ClientResources, error) {
	if id <= 0 {
		return nil, fault.ErrNotFound
	}
	row, err := postgres.GetOne[clientResourcesRow](
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
		return nil, fault.ErrNotFound
	}
	row, err := postgres.GetOne[clientResourcesRow](
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
	if err := s.pool.QueryRow(ctx, `
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
		return nil, postgres.MutationError(err)
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, next clientResourcesWrite) (*ClientResources, error) {
	var replacedObjectIDs []int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
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
			return postgres.MutationError(err)
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
		"links":             postgres.JSONSlice[Link](builder.Links),
		"footer_text":       builder.FooterText,
		"footer_links":      postgres.JSONSlice[Link](builder.FooterLinks),
	}
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	var archiveObjectID int64
	var bannerObjectID *int64
	if err := s.pool.QueryRow(ctx, `
DELETE FROM munki_client_resources
WHERE id = $1
RETURNING archive_object_id, banner_object_id`, id).Scan(&archiveObjectID, &bannerObjectID); err != nil {
		return postgres.MutationError(err)
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
