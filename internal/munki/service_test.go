package munki_test

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestResolvePackageFileRequiresEffectivePackage(t *testing.T) {
	installerID := int64(42)
	store := servicePackageStore{
		packages: []munkisoftware.EffectivePackage{
			{
				TargetID:   1,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package: packages.Package{
					ID:                10,
					SoftwareName:      "GoogleChrome",
					InstallerType:     packages.InstallerTypePkg,
					InstallerObjectID: &installerID,
				},
			},
		},
	}
	objects := serviceObjectStore{objects: map[int64]storage.Object{
		installerID: {ID: installerID, Prefix: packages.ObjectPrefix, Filename: "GoogleChrome.pkg"},
	}}
	service := munki.NewRepositoryService(munki.Dependencies{Packages: store, Objects: objects})
	client := munki.ClientHost{ID: 1, Serial: "C02MUNKI"}

	installer, err := service.ResolvePackageFile(context.Background(), client, "packages/10/installer/GoogleChrome.pkg")
	if err != nil {
		t.Fatalf("ResolvePackageFile allowed package: %v", err)
	}
	if installer.Key != "munki/packages/42/GoogleChrome.pkg" {
		t.Fatalf("key = %q, want the installer key", installer.Key)
	}
	if installer.PackageID != 10 {
		t.Fatalf("package id = %d, want 10", installer.PackageID)
	}

	_, err = service.ResolvePackageFile(context.Background(), client, "munki/packages/99/Blocked.pkg")
	if !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("blocked key error = %v, want ErrNotFound", err)
	}
}

func TestManifestRequiresClientIdentifierName(t *testing.T) {
	service := munki.NewRepositoryService(munki.Dependencies{
		Packages: servicePackageStore{},
	})
	client := munki.ClientHost{ID: 1, Serial: "C02MUNKI"}

	if _, err := service.Manifest(context.Background(), client, "C02OTHER"); !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("Manifest wrong name error = %v, want ErrNotFound", err)
	}
	if _, err := service.Manifest(context.Background(), client, "C02MUNKI"); err != nil {
		t.Fatalf("Manifest client name error = %v, want nil", err)
	}
}

func TestCatalogRequiresProductionName(t *testing.T) {
	service := munki.NewRepositoryService(munki.Dependencies{
		Packages: servicePackageStore{},
		Objects:  serviceObjectStore{},
	})
	client := munki.ClientHost{ID: 1, Serial: "C02MUNKI"}

	if _, err := service.Catalog(context.Background(), client, "testing"); !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("Catalog wrong name error = %v, want ErrNotFound", err)
	}
	if _, err := service.Catalog(context.Background(), client, "production"); err != nil {
		t.Fatalf("Catalog production error = %v, want nil", err)
	}
}

type serviceObjectStore struct {
	objects map[int64]storage.Object
}

func (s serviceObjectStore) ListByIDs(_ context.Context, ids []int64) (map[int64]storage.Object, error) {
	out := make(map[int64]storage.Object, len(ids))
	for _, id := range ids {
		if obj, ok := s.objects[id]; ok {
			out[id] = obj
		}
	}
	return out, nil
}

type servicePackageStore struct {
	packages []munkisoftware.EffectivePackage
}

func (s servicePackageStore) EffectivePackagesForHost(
	_ context.Context,
	hostID int64,
) ([]munkisoftware.EffectivePackage, error) {
	if hostID != 1 {
		return nil, nil
	}
	return s.packages, nil
}
