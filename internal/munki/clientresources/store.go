package clientresources

import (
	"context"
	"errors"
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
    cr.archive_object_id,
    cr.custom,
    b.banner_object_id IS NOT NULL AS has_builder,
    COALESCE(b.banner_object_id, 0) AS banner_object_id,
    COALESCE(b.banner_fit, 'height') AS banner_fit,
    COALESCE(b.banner_focal_x, 0) AS banner_focal_x,
    COALESCE(b.links, '[]'::jsonb) AS links,
    COALESCE(b.footer_text, '') AS footer_text,
    COALESCE(b.footer_links, '[]'::jsonb) AS footer_links,
    cr.created_at,
    cr.updated_at
FROM munki_client_resources cr
LEFT JOIN munki_client_resource_builders b ON b.singleton = cr.singleton
WHERE cr.singleton`

func (s *Store) Get(ctx context.Context) (*ClientResources, error) {
	return get(ctx, s.db.Pool())
}

func get(ctx context.Context, q dbutil.Queryer) (*ClientResources, error) {
	row, err := dbutil.GetOne[clientResourcesRow](ctx, q, clientResourcesSelectSQL)
	if err != nil {
		return nil, err
	}
	resource := clientResourcesFromRow(row)
	return &resource, nil
}

func (s *Store) PublishBuilder(ctx context.Context, builder storedBuilder) (*ClientResources, error) {
	return s.publish(ctx, publication{
		archiveObjectID: builder.ArchiveObjectID,
		builder:         &builder.Builder,
	})
}

func (s *Store) PublishArchive(ctx context.Context, archiveObjectID int64) (*ClientResources, error) {
	return s.publish(ctx, publication{archiveObjectID: archiveObjectID, custom: true})
}

type publication struct {
	archiveObjectID int64
	custom          bool
	builder         *Builder
}

func (s *Store) publish(ctx context.Context, next publication) (*ClientResources, error) {
	var replacedObjectIDs []int64
	var resource *ClientResources
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `LOCK TABLE munki_client_resources IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return err
		}
		existing, err := get(ctx, tx)
		if err != nil && !errors.Is(err, dbutil.ErrNotFound) {
			return err
		}
		replacedObjectIDs = replacedClientResourceObjectIDs(existing, next)

		_, err = tx.Exec(ctx, `
INSERT INTO munki_client_resources (singleton, archive_object_id, custom)
VALUES (TRUE, @archive_object_id, @custom)
ON CONFLICT (singleton) DO UPDATE SET
    archive_object_id = EXCLUDED.archive_object_id,
    custom = EXCLUDED.custom,
    updated_at = now()`, pgx.NamedArgs{
			"archive_object_id": next.archiveObjectID,
			"custom":            next.custom,
		})
		if err != nil {
			return dbutil.MutationError(err)
		}

		if next.builder != nil {
			_, err = tx.Exec(ctx, `
INSERT INTO munki_client_resource_builders (
    singleton,
    banner_object_id,
    banner_fit,
    banner_focal_x,
    links,
    footer_text,
    footer_links
) VALUES (
    TRUE,
    @banner_object_id,
    @banner_fit,
    @banner_focal_x,
    @links::jsonb,
    @footer_text,
    @footer_links::jsonb
)
ON CONFLICT (singleton) DO UPDATE SET
    banner_object_id = EXCLUDED.banner_object_id,
    banner_fit = EXCLUDED.banner_fit,
    banner_focal_x = EXCLUDED.banner_focal_x,
    links = EXCLUDED.links,
    footer_text = EXCLUDED.footer_text,
    footer_links = EXCLUDED.footer_links`, pgx.NamedArgs{
				"banner_object_id": next.builder.BannerObjectID,
				"banner_fit":       next.builder.BannerFit,
				"banner_focal_x":   next.builder.BannerFocalX,
				"links":            dbutil.JSONSlice[Link](next.builder.Links),
				"footer_text":      next.builder.FooterText,
				"footer_links":     dbutil.JSONSlice[Link](next.builder.FooterLinks),
			})
			if err != nil {
				return dbutil.MutationError(err)
			}
		}

		resource, err = get(ctx, tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.objects.DeleteUnreferenced(ctx, replacedObjectIDs...)
	return resource, nil
}

func (s *Store) Delete(ctx context.Context) error {
	var objectIDs []int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `LOCK TABLE munki_client_resources IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return err
		}
		existing, err := get(ctx, tx)
		if err != nil {
			return err
		}
		objectIDs = clientResourceObjectIDs(existing)
		tag, err := tx.Exec(ctx, `DELETE FROM munki_client_resources WHERE singleton`)
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
	s.objects.DeleteUnreferenced(ctx, objectIDs...)
	return nil
}

func replacedClientResourceObjectIDs(
	existing *ClientResources,
	next publication,
) []int64 {
	if existing == nil {
		return nil
	}
	retained := map[int64]struct{}{next.archiveObjectID: {}}
	if next.builder != nil {
		retained[next.builder.BannerObjectID] = struct{}{}
	} else if existing.Builder != nil {
		retained[existing.Builder.BannerObjectID] = struct{}{}
	}
	var replaced []int64
	for _, objectID := range clientResourceObjectIDs(existing) {
		if _, ok := retained[objectID]; !ok {
			replaced = append(replaced, objectID)
		}
	}
	return replaced
}

func clientResourceObjectIDs(resource *ClientResources) []int64 {
	objectIDs := []int64{resource.ArchiveObjectID}
	if resource.Builder != nil {
		objectIDs = append(objectIDs, resource.Builder.BannerObjectID)
	}
	return objectIDs
}

func clientResourcesFromRow(row clientResourcesRow) ClientResources {
	resource := ClientResources{
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
