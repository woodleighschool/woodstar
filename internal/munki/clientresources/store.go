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
	BannerObjectID  int64                  `db:"banner_object_id"`
	ArchiveObjectID int64                  `db:"archive_object_id"`
	BannerAlignment BannerAlignment        `db:"banner_alignment"`
	Links           dbutil.JSONSlice[Link] `db:"links"`
	FooterText      string                 `db:"footer_text"`
	FooterLinks     dbutil.JSONSlice[Link] `db:"footer_links"`
	CreatedAt       time.Time              `db:"created_at"`
	UpdatedAt       time.Time              `db:"updated_at"`
}

const clientResourcesSelectSQL = `SELECT
    cr.banner_object_id,
    cr.archive_object_id,
    cr.banner_alignment,
    cr.links,
    cr.footer_text,
    cr.footer_links,
    cr.created_at,
    cr.updated_at
FROM munki_client_resources cr
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

func (s *Store) Upsert(ctx context.Context, mutation storedMutation) (*ClientResources, error) {
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
		if existing != nil {
			replacedObjectIDs = replacedClientResourceObjectIDs(*existing, mutation)
		}

		_, err = tx.Exec(ctx, `
INSERT INTO munki_client_resources (
    singleton,
    banner_object_id,
    archive_object_id,
    banner_alignment,
    links,
    footer_text,
    footer_links
) VALUES (
    TRUE,
    @banner_object_id,
    @archive_object_id,
    @banner_alignment,
    @links::jsonb,
    @footer_text,
    @footer_links::jsonb
)
ON CONFLICT (singleton) DO UPDATE SET
    banner_object_id = EXCLUDED.banner_object_id,
    archive_object_id = EXCLUDED.archive_object_id,
    banner_alignment = EXCLUDED.banner_alignment,
    links = EXCLUDED.links,
    footer_text = EXCLUDED.footer_text,
    footer_links = EXCLUDED.footer_links,
    updated_at = now()`, pgx.NamedArgs{
			"banner_object_id":  mutation.BannerObjectID,
			"archive_object_id": mutation.ArchiveObjectID,
			"banner_alignment":  mutation.BannerAlignment,
			"links":             dbutil.JSONSlice[Link](mutation.Links),
			"footer_text":       mutation.FooterText,
			"footer_links":      dbutil.JSONSlice[Link](mutation.FooterLinks),
		})
		if err != nil {
			return dbutil.MutationError(err)
		}
		resource, err = get(ctx, tx)
		if err != nil {
			return err
		}
		return s.objects.RequestDeletion(ctx, tx, replacedObjectIDs...)
	})
	if err != nil {
		return nil, err
	}
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
		objectIDs = []int64{existing.BannerObjectID, existing.ArchiveObjectID}
		tag, err := tx.Exec(ctx, `DELETE FROM munki_client_resources WHERE singleton`)
		if err != nil {
			return dbutil.MutationError(err)
		}
		if tag.RowsAffected() == 0 {
			return dbutil.ErrNotFound
		}
		return s.objects.RequestDeletion(ctx, tx, objectIDs...)
	})
	if err != nil {
		return err
	}
	return nil
}

func replacedClientResourceObjectIDs(existing ClientResources, replacement storedMutation) []int64 {
	current := map[int64]struct{}{
		replacement.BannerObjectID:  {},
		replacement.ArchiveObjectID: {},
	}
	var replaced []int64
	for _, id := range []int64{existing.BannerObjectID, existing.ArchiveObjectID} {
		if _, retained := current[id]; !retained {
			replaced = append(replaced, id)
		}
	}
	return replaced
}

func clientResourcesFromRow(row clientResourcesRow) ClientResources {
	return ClientResources{
		Mutation: Mutation{
			BannerObjectID:  row.BannerObjectID,
			BannerAlignment: row.BannerAlignment,
			Links:           row.Links,
			FooterText:      row.FooterText,
			FooterLinks:     row.FooterLinks,
		},
		ArchiveObjectID: row.ArchiveObjectID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
