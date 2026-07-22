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

func TestStoreCRUDKeepsEffectiveSingleton(t *testing.T) { //nolint:cyclop,funlen,gocognit // Linear CRUD and object lifecycle.
	db, ctx := testdb.Open(t)
	objects := storage.NewObjectStore(db, nil, slog.New(slog.DiscardHandler))
	store := NewStore(db, objects)
	resources, count, err := store.List(ctx, dbutil.ListParams{})
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if count != 0 || len(resources) != 0 {
		t.Fatalf("List empty = %d/%+v, want 0/empty", count, resources)
	}

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

	generatedWrite := clientResourcesWrite{
		builder: &Builder{
			BannerObjectID: banner.ID,
			BannerFit:      BannerFitCover,
			BannerFocalX:   50,
			Links:          []Link{},
			FooterLinks:    []Link{},
		},
		archiveObjectID: generatedArchive.ID,
	}
	if _, err := db.Pool().Exec(ctx, `
INSERT INTO munki_client_resources (archive_object_id, custom)
VALUES ($1, FALSE)`, generatedArchive.ID); err == nil {
		t.Fatal("insert non-custom client resources without builder succeeded")
	}

	generated, err := store.Create(ctx, generatedWrite)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if generated.ID != 1 || generated.ArchiveObjectID != generatedArchive.ID ||
		generated.Custom || generated.Builder == nil {
		t.Fatalf("generated resources = %+v", generated)
	}
	if generated.Builder.BannerObjectID != banner.ID ||
		generated.Builder.BannerFit != BannerFitCover ||
		generated.Builder.BannerFocalX != 50 {
		t.Fatalf("generated builder = %+v", generated.Builder)
	}
	if _, err := store.Create(ctx, generatedWrite); !errors.Is(err, dbutil.ErrAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrAlreadyExists", err)
	}
	if _, err := db.Pool().Exec(ctx, `
INSERT INTO munki_client_resources (id, archive_object_id, custom, banner_object_id)
VALUES (2, $1, FALSE, $2)`, generatedArchive.ID, banner.ID); err == nil {
		t.Fatal("insert client resource ID 2 succeeded")
	}
	resources, count, err = store.List(ctx, dbutil.ListParams{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if count != 1 || len(resources) != 1 || resources[0].ID != 1 {
		t.Fatalf("List = %d/%+v, want only ID 1", count, resources)
	}
	if _, err := store.GetByID(ctx, 2); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("GetByID(2) error = %v, want ErrNotFound", err)
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
	uploaded, err := store.Update(ctx, generated.ID, clientResourcesWrite{
		archiveObjectID: uploadedArchive.ID,
		custom:          true,
	})
	if err != nil {
		t.Fatalf("Update uploaded archive: %v", err)
	}
	if uploaded.ArchiveObjectID != uploadedArchive.ID || !uploaded.Custom || uploaded.Builder == nil {
		t.Fatalf("uploaded resources = %+v", uploaded)
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
	rebuilt, err := store.Update(ctx, generated.ID, clientResourcesWrite{
		builder:         uploaded.Builder,
		archiveObjectID: rebuiltArchive.ID,
	})
	if err != nil {
		t.Fatalf("rebuild resources: %v", err)
	}
	if rebuilt.Custom || rebuilt.Builder == nil || rebuilt.Builder.BannerObjectID != banner.ID {
		t.Fatalf("rebuilt resources = %+v", rebuilt)
	}
	if _, err := objects.GetByID(ctx, uploadedArchive.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get replaced uploaded archive error = %v, want ErrNotFound", err)
	}

	if err := store.Delete(ctx, generated.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.GetByID(ctx, generated.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("GetByID after Delete error = %v, want ErrNotFound", err)
	}
	if _, err := objects.GetByID(ctx, rebuiltArchive.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get undeployed archive error = %v, want ErrNotFound", err)
	}
	if _, err := objects.GetByID(ctx, banner.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get undeployed banner error = %v, want ErrNotFound", err)
	}

	recreatedArchive := createAvailableObject(
		t,
		ctx,
		db,
		objects,
		ArchiveObjectPrefix,
		"recreated.zip",
		"application/zip",
	)
	recreated, err := store.Create(ctx, clientResourcesWrite{
		archiveObjectID: recreatedArchive.ID,
		custom:          true,
	})
	if err != nil {
		t.Fatalf("recreate: %v", err)
	}
	if recreated.ID != 1 {
		t.Fatalf("recreated ID = %d, want 1", recreated.ID)
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
