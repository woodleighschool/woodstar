package packages

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/database/dbtest/crudtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestPackageStoreConformance(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db, storage.NewObjectStore(db, nil))

	crudtest.RunConformance(
		t,
		ctx,
		crudtest.Fixtures[Package, PackageCreateMutation, PackageMutation, PackageListParams]{
			Store: store,
			NewValid: func(t *testing.T, ctx context.Context) PackageCreateMutation {
				softwareID := insertSoftware(t, ctx, db, "ConformanceApp")
				return PackageCreateMutation{
					SoftwareID: softwareID,
					PackageMutation: PackageMutation{
						Version:       "1.0.0",
						InstallerType: InstallerTypeNoPkg,
						OnDemand:      true,
						Eligible:      true,
					},
				}
			},
			Mutate: func(_ Package) PackageMutation {
				return PackageMutation{
					Version:       "2.0.0",
					InstallerType: InstallerTypeNoPkg,
					OnDemand:      true,
					Eligible:      true,
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
						Eligible:      true,
					},
				}, true
			},
		},
	)
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
