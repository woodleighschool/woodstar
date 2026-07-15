package clientresources

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestStoreUpsertAndDeleteOwnReferencedObjects(t *testing.T) {
	db, ctx := dbtest.Open(t)
	objects := storage.NewObjectStore(db, nil)
	store := NewStore(db, objects)
	banner := createAvailableObject(t, ctx, objects, BannerObjectPrefix, "banner.png", "image/png")
	firstArchive := createAvailableObject(t, ctx, objects, ArchiveObjectPrefix, archiveFilename, archiveContentType)

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

	secondArchive := createAvailableObject(t, ctx, objects, ArchiveObjectPrefix, archiveFilename, archiveContentType)
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
	if _, err := objects.GetByID(ctx, firstArchive.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("old archive GetByID error = %v, want ErrNotFound", err)
	}

	if err := store.Delete(ctx); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("Get after Delete error = %v, want ErrNotFound", err)
	}
	for _, objectID := range []int64{banner.ID, secondArchive.ID} {
		if _, err := objects.GetByID(ctx, objectID); !errors.Is(err, dbutil.ErrNotFound) {
			t.Fatalf("object %d GetByID error = %v, want ErrNotFound", objectID, err)
		}
	}
}

func createAvailableObject(
	t *testing.T,
	ctx context.Context,
	objects *storage.ObjectStore,
	prefix string,
	filename string,
	contentType string,
) *storage.Object {
	t.Helper()
	object, err := objects.CreatePending(ctx, prefix, filename, contentType)
	if err != nil {
		t.Fatalf("CreatePending: %v", err)
	}
	object, err = objects.Confirm(ctx, object.ID, 1, contentType, strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	return object
}
