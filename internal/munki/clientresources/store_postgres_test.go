//go:build postgres

package clientresources

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestStoreTransitionsBetweenBuilderAndUploadedArchive(t *testing.T) {
	db, ctx := testdb.Open(t)
	objects := storage.NewObjectStore(db, nil, slog.New(slog.DiscardHandler))
	store := NewStore(db, objects)
	banner := createAvailableObject(t, ctx, db, objects, BannerObjectPrefix, "banner.png", "image/png")
	generatedArchive := createAvailableObject(
		t,
		ctx,
		db,
		objects,
		ArchiveObjectPrefix,
		archiveFilename,
		"application/zip",
	)

	generated, err := store.PublishBuilder(ctx, storedBuilder{
		Builder: Builder{
			BannerObjectID: banner.ID,
			BannerFit:      BannerFitCover,
			BannerFocalX:   50,
			Links:          []Link{},
			FooterLinks:    []Link{},
		},
		ArchiveObjectID: generatedArchive.ID,
	})
	if err != nil {
		t.Fatalf("PublishBuilder: %v", err)
	}
	if generated.ArchiveObjectID != generatedArchive.ID || generated.Custom || generated.Builder == nil {
		t.Fatalf("published builder resources = %+v", generated)
	}
	if generated.Builder.BannerObjectID != banner.ID ||
		generated.Builder.BannerFit != BannerFitCover ||
		generated.Builder.BannerFocalX != 50 {
		t.Fatalf("published builder = %+v", generated.Builder)
	}

	uploadedArchive := createAvailableObject(
		t,
		ctx,
		db,
		objects,
		ArchiveObjectPrefix,
		"school-resources.zip",
		"application/zip",
	)
	uploaded, err := store.PublishArchive(ctx, uploadedArchive.ID)
	if err != nil {
		t.Fatalf("PublishArchive: %v", err)
	}
	if uploaded.ArchiveObjectID != uploadedArchive.ID || !uploaded.Custom || uploaded.Builder == nil {
		t.Fatalf("published uploaded resources = %+v", uploaded)
	}
	if uploaded.Builder.BannerObjectID != banner.ID {
		t.Fatalf("retained builder = %+v, want banner %d", uploaded.Builder, banner.ID)
	}
	if _, err := objects.GetByID(ctx, banner.ID); err != nil {
		t.Fatalf("get retained banner: %v", err)
	}
	if _, err := objects.GetByID(ctx, generatedArchive.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get replaced generated archive error = %v, want ErrNotFound", err)
	}

	rebuiltArchive := createAvailableObject(
		t,
		ctx,
		db,
		objects,
		ArchiveObjectPrefix,
		archiveFilename,
		"application/zip",
	)
	rebuilt, err := store.PublishBuilder(ctx, storedBuilder{
		Builder:         *uploaded.Builder,
		ArchiveObjectID: rebuiltArchive.ID,
	})
	if err != nil {
		t.Fatalf("republish builder: %v", err)
	}
	if rebuilt.Custom || rebuilt.Builder == nil || rebuilt.Builder.BannerObjectID != banner.ID {
		t.Fatalf("republished builder resources = %+v", rebuilt)
	}
	if _, err := objects.GetByID(ctx, uploadedArchive.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get replaced uploaded archive error = %v, want ErrNotFound", err)
	}

	if err := store.Delete(ctx); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("Get after Delete error = %v, want ErrNotFound", err)
	}
	if _, err := objects.GetByID(ctx, rebuiltArchive.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get undeployed archive error = %v, want ErrNotFound", err)
	}
	if _, err := objects.GetByID(ctx, banner.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get undeployed banner error = %v, want ErrNotFound", err)
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
