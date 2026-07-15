package clientresources

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type Store struct {
	db      *database.DB
	objects objectCleaner
}

func NewStore(db *database.DB, objects objectCleaner) *Store {
	return &Store{db: db, objects: objects}
}

type clientResourcesRow struct {
	BannerObjectID  int64           `db:"banner_object_id"`
	ArchiveObjectID int64           `db:"archive_object_id"`
	BannerAlignment BannerAlignment `db:"banner_alignment"`
	Links           linksValue      `db:"links"`
	FooterText      string          `db:"footer_text"`
	FooterLinks     linksValue      `db:"footer_links"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
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
			replacedObjectIDs = append(
				replacedObjectIDs,
				existing.BannerObjectID,
				existing.ArchiveObjectID,
			)
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
			"links":             linksValue(mutation.Links),
			"footer_text":       mutation.FooterText,
			"footer_links":      linksValue(mutation.FooterLinks),
		})
		if err != nil {
			return dbutil.MutationError(err)
		}
		resource, err = get(ctx, tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := cleanupObjects(ctx, s.objects, replacedObjectIDs...); err != nil {
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
		return nil
	})
	if err != nil {
		return err
	}
	return cleanupObjects(ctx, s.objects, objectIDs...)
}

func clientResourcesFromRow(row clientResourcesRow) ClientResources {
	return ClientResources{
		Mutation: Mutation{
			BannerObjectID:  row.BannerObjectID,
			BannerAlignment: row.BannerAlignment,
			Links:           []Link(row.Links),
			FooterText:      row.FooterText,
			FooterLinks:     []Link(row.FooterLinks),
		},
		ArchiveObjectID: row.ArchiveObjectID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
