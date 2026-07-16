package packages

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/database/dbtest/crudtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestPackageStoreConformance(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db, storage.NewObjectStore(db, nil), slog.New(slog.DiscardHandler))

	crudtest.RunConformance(
		t,
		ctx,
		crudtest.Fixtures[Package, PackageCreateMutation, PackageMutation, PackageListParams]{
			Store: store,
			NewValid: func(t *testing.T, ctx context.Context) PackageCreateMutation {
				t.Helper()

				softwareID := insertSoftware(t, ctx, db, "ConformanceApp")
				return PackageCreateMutation{
					SoftwareID: softwareID,
					PackageMutation: PackageMutation{
						Version:       "1.0.0",
						InstallerType: InstallerTypeNoPkg,
						OnDemand:      true,
					},
				}
			},
			Mutate: func(_ Package) PackageMutation {
				return PackageMutation{
					Version:       "2.0.0",
					InstallerType: InstallerTypeNoPkg,
					OnDemand:      true,
				}
			},
			ID:         func(pkg Package) int64 { return pkg.ID },
			ListParams: packageListParams,
			SortKeys:   slices.Sorted(maps.Keys(packageOrderKeys())),
			SearchMatch: func(pkg Package) string {
				return pkg.SoftwareName
			},
			NewInvalid: func() (PackageCreateMutation, bool) {
				return PackageCreateMutation{
					SoftwareID: 1,
					PackageMutation: PackageMutation{
						Version:       "1.0.0",
						InstallerType: InstallerType("bogus"),
					},
				}, true
			},
		},
	)
}

func TestPackageUpdateSucceedsWhenDetachedInstallerCleanupFails(t *testing.T) {
	db, ctx := dbtest.Open(t)
	registry := storage.NewObjectStore(db, nil)
	objects := &failingObjectStore{err: errors.New("storage unavailable")}
	store := NewStore(db, objects, slog.New(slog.DiscardHandler))
	softwareID := insertSoftware(t, ctx, db, "CleanupFailure")
	oldInstaller := createAvailableInstaller(t, ctx, registry, "old.pkg")
	replacement := createAvailableInstaller(t, ctx, registry, "replacement.pkg")

	pkg, err := store.Create(ctx, PackageCreateMutation{
		SoftwareID: softwareID,
		PackageMutation: PackageMutation{
			Version:           "1.0.0",
			InstallerType:     InstallerTypePkg,
			InstallerObjectID: &oldInstaller.ID,
		},
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	updated, err := store.Update(ctx, pkg.ID, PackageMutation{
		Version:           pkg.Version,
		InstallerType:     InstallerTypePkg,
		InstallerObjectID: &replacement.ID,
	})
	if err != nil {
		t.Fatalf("update package after cleanup failure: %v", err)
	}
	if updated.InstallerObjectID == nil || *updated.InstallerObjectID != replacement.ID {
		t.Fatalf("installer object = %v, want %d", updated.InstallerObjectID, replacement.ID)
	}
	if len(objects.deletedIDs) != 1 || objects.deletedIDs[0] != oldInstaller.ID {
		t.Fatalf("cleanup IDs = %v, want [%d]", objects.deletedIDs, oldInstaller.ID)
	}
}

type failingObjectStore struct {
	err        error
	deletedIDs []int64
}

func (s *failingObjectStore) Delete(_ context.Context, id int64) error {
	s.deletedIDs = append(s.deletedIDs, id)
	return s.err
}

func createAvailableInstaller(
	t *testing.T,
	ctx context.Context,
	registry *storage.ObjectStore,
	filename string,
) *storage.Object {
	t.Helper()
	object, err := registry.CreatePending(ctx, ObjectPrefix, filename)
	if err != nil {
		t.Fatalf("create pending installer: %v", err)
	}
	object, err = registry.MarkAvailable(
		ctx,
		object.ID,
		1,
		"application/octet-stream",
		strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatalf("finalize installer: %v", err)
	}
	return object
}

func packageListParams(q, sort string, pageIndex, pageSize int32) PackageListParams {
	return PackageListParams{
		ListParams: dbutil.ListParams{
			Q:         q,
			Sort:      sort,
			PageIndex: pageIndex,
			PageSize:  pageSize,
		},
	}
}

func insertSoftware(t *testing.T, ctx context.Context, db *database.DB, name string) int64 {
	t.Helper()
	var id int64
	err := db.Pool().
		QueryRow(ctx, `INSERT INTO munki_software (name, display_name) VALUES ($1, $1) RETURNING id`, name).
		Scan(&id)
	if err != nil {
		t.Fatalf("insert munki_software: %v", err)
	}
	return id
}
