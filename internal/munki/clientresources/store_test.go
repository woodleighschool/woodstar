package clientresources

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestStoreUpsertAndDeleteQueueUnreferencedObjects(t *testing.T) {
	db, ctx := dbtest.Open(t)
	objects := storage.NewObjectStore(db, nil)
	store := NewStore(db, objects)
	banner := createAvailableObject(t, ctx, db, objects, BannerObjectPrefix, "banner.png", "image/png")
	firstArchive := createAvailableObject(t, ctx, db, objects, ArchiveObjectPrefix, archiveFilename, "application/zip")

	first, err := store.Upsert(ctx, storedMutation{
		Mutation: Mutation{
			BannerObjectID:  banner.ID,
			BannerAlignment: BannerAlignmentLeft,
			Links:           []Link{},
			FooterLinks:     []Link{},
		},
		ArchiveObjectID: firstArchive.ID,
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if first.BannerObjectID != banner.ID || first.ArchiveObjectID != firstArchive.ID {
		t.Fatalf(
			"Upsert objects = banner %d archive %d",
			first.BannerObjectID,
			first.ArchiveObjectID,
		)
	}
	if first.BannerAlignment != BannerAlignmentLeft {
		t.Fatalf("banner alignment = %q, want %q", first.BannerAlignment, BannerAlignmentLeft)
	}

	secondArchive := createAvailableObject(t, ctx, db, objects, ArchiveObjectPrefix, archiveFilename, "application/zip")
	second, err := store.Upsert(ctx, storedMutation{
		Mutation: Mutation{
			BannerObjectID:  banner.ID,
			BannerAlignment: BannerAlignmentCenter,
			Links:           []Link{},
			FooterLinks:     []Link{},
		},
		ArchiveObjectID: secondArchive.ID,
	})
	if err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	if second.ArchiveObjectID != secondArchive.ID {
		t.Fatalf("archive id = %d, want %d", second.ArchiveObjectID, secondArchive.ID)
	}
	if second.BannerAlignment != BannerAlignmentCenter {
		t.Fatalf("banner alignment = %q, want %q", second.BannerAlignment, BannerAlignmentCenter)
	}
	replacedArchive, err := objects.GetByID(ctx, firstArchive.ID)
	if err != nil {
		t.Fatalf("get old archive: %v", err)
	}
	if replacedArchive.DeletionRequestedAt == nil {
		t.Fatal("old archive was not queued for deletion")
	}
	retainedBanner, err := objects.GetByID(ctx, banner.ID)
	if err != nil {
		t.Fatalf("get retained banner: %v", err)
	}
	if retainedBanner.DeletionRequestedAt != nil {
		t.Fatal("retained banner was queued for deletion")
	}

	if err := store.Delete(ctx); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("Get after Delete error = %v, want ErrNotFound", err)
	}
	for _, objectID := range []int64{banner.ID, secondArchive.ID} {
		object, err := objects.GetByID(ctx, objectID)
		if err != nil {
			t.Fatalf("get object %d: %v", objectID, err)
		}
		if object.DeletionRequestedAt == nil {
			t.Fatalf("object %d was not queued for deletion", objectID)
		}
	}
}

func createAvailableObject(
	t *testing.T,
	ctx context.Context,
	db *database.DB,
	objects *storage.ObjectStore,
	prefix string,
	filename string,
	contentType string,
) *storage.Object {
	t.Helper()
	var objectID int64
	if err := db.Pool().QueryRow(ctx, `
INSERT INTO storage_objects (
    prefix, filename, content_type, size_bytes, sha256, available_at
) VALUES ($1, $2, $3, 1, $4, now())
RETURNING id`, prefix, filename, contentType, strings.Repeat("a", 64)).Scan(&objectID); err != nil {
		t.Fatalf("insert available object: %v", err)
	}
	object, err := objects.GetByID(ctx, objectID)
	if err != nil {
		t.Fatalf("get available object: %v", err)
	}
	return object
}
