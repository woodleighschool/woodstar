package munki_test

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestResolvePackageFileUsesRepositoryPackages(t *testing.T) {
	installerID := int64(42)
	store := servicePackageStore{
		repositoryPackages: []packages.Package{
			{
				ID:                10,
				SoftwareName:      "GoogleChrome",
				InstallerType:     packages.InstallerTypePkg,
				InstallerObjectID: &installerID,
			},
		},
	}
	objects := serviceObjectStore{objects: map[int64]storage.Object{
		installerID: {ID: installerID, Prefix: packages.ObjectPrefix, Filename: "GoogleChrome.pkg"},
	}}
	service := munki.NewRepositoryService(munki.Dependencies{Packages: store, Objects: objects})

	installer, err := service.ResolvePackageFile(context.Background(), "packages/10/installer/GoogleChrome.pkg")
	if err != nil {
		t.Fatalf("ResolvePackageFile allowed package: %v", err)
	}
	if installer.Key != "munki/packages/42/GoogleChrome.pkg" {
		t.Fatalf("key = %q, want the installer key", installer.Key)
	}
	if installer.PackageID != 10 {
		t.Fatalf("package id = %d, want 10", installer.PackageID)
	}

	_, err = service.ResolvePackageFile(context.Background(), "munki/packages/99/Blocked.pkg")
	if !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("blocked key error = %v, want ErrNotFound", err)
	}
}

func TestManifestRequiresClientIdentifierName(t *testing.T) {
	service := munki.NewRepositoryService(munki.Dependencies{
		Hosts:    serviceHostStore{host: &hosts.Host{ID: 1, Hardware: hosts.HostHardware{Serial: "C02MUNKI"}}},
		Software: servicePackageStore{},
	})

	if _, err := service.Manifest(context.Background(), "C02OTHER"); !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("Manifest wrong name error = %v, want ErrNotFound", err)
	}
	if _, err := service.Manifest(context.Background(), "C02MUNKI"); err != nil {
		t.Fatalf("Manifest client name error = %v, want nil", err)
	}
}

func TestCatalogRequiresWoodstarName(t *testing.T) {
	service := munki.NewRepositoryService(munki.Dependencies{
		Packages: servicePackageStore{},
		Objects:  serviceObjectStore{},
	})

	if _, err := service.Catalog(context.Background(), "testing"); !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("Catalog wrong name error = %v, want ErrNotFound", err)
	}
	if _, err := service.Catalog(context.Background(), "woodstar"); err != nil {
		t.Fatalf("Catalog woodstar error = %v, want nil", err)
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
	packages           []munkisoftware.EffectivePackage
	repositoryPackages []packages.Package
}

type serviceHostStore struct {
	host *hosts.Host
}

func (s serviceHostStore) GetByHardwareSerial(_ context.Context, serial string) (*hosts.Host, error) {
	if s.host == nil || s.host.Hardware.Serial != serial {
		return nil, dbutil.ErrNotFound
	}
	return s.host, nil
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

func (s servicePackageStore) ListRepositoryPackages(
	_ context.Context,
) ([]packages.Package, error) {
	return s.repositoryPackages, nil
}
