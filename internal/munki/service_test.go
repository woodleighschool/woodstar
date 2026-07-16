package munki_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestResolvePackageFileUsesEmbeddedPackageID(t *testing.T) {
	installerID := int64(42)
	availableAt := time.Now()
	store := servicePackageStore{
		packagesByID: map[int64]packages.Package{10: {
			ID:                10,
			SoftwareName:      "GoogleChrome",
			InstallerType:     packages.InstallerTypePkg,
			InstallerObjectID: &installerID,
			Eligible:          true,
		}},
		listRepositoryErr: errors.New("full repository scan should not be used"),
	}
	objects := serviceObjectStore{objects: map[int64]storage.Object{
		installerID: {
			ID:          installerID,
			Prefix:      packages.ObjectPrefix,
			Filename:    "GoogleChrome.pkg",
			ContentType: "application/octet-stream",
			AvailableAt: &availableAt,
		},
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
	if installer.ContentType != "application/octet-stream" {
		t.Fatalf("content type = %q, want application/octet-stream", installer.ContentType)
	}

	_, err = service.ResolvePackageFile(context.Background(), "munki/packages/99/Blocked.pkg")
	if !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("blocked key error = %v, want ErrNotFound", err)
	}
}

func TestResolveIconFileUsesEmbeddedObjectID(t *testing.T) {
	iconID := int64(42)
	availableAt := time.Now()
	store := servicePackageStore{
		packagesByIconObjectID: map[int64][]packages.Package{iconID: {{
			ID:                   10,
			SoftwareName:         "GoogleChrome",
			InstallerType:        packages.InstallerTypeNoPkg,
			SoftwareIconObjectID: &iconID,
			Eligible:             true,
		}}},
		listRepositoryErr: errors.New("full repository scan should not be used"),
	}
	objects := serviceObjectStore{objects: map[int64]storage.Object{
		iconID: {
			ID:          iconID,
			Prefix:      "munki/icons",
			Filename:    "GoogleChrome.png",
			ContentType: "image/png",
			AvailableAt: &availableAt,
		},
	}}
	service := munki.NewRepositoryService(munki.Dependencies{Packages: store, Objects: objects})

	file, err := service.ResolveIconFile(context.Background(), "42-GoogleChrome.png")
	if err != nil {
		t.Fatalf("ResolveIconFile allowed icon: %v", err)
	}
	if file.Key != "munki/icons/42/GoogleChrome.png" || file.ContentType != "image/png" {
		t.Fatalf("file = %+v, want canonical icon storage metadata", file)
	}

	_, err = service.ResolveIconFile(context.Background(), "42-Other.png")
	if !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("mismatched icon error = %v, want ErrNotFound", err)
	}
}

func TestResolveClientResourcesAcceptsKnownHostAndSiteDefault(t *testing.T) {
	availableAt := time.Now()
	resource := &clientresources.ClientResources{
		ArchiveObjectID: 9,
	}
	archive := storage.Object{
		ID:          9,
		Prefix:      clientresources.ArchiveObjectPrefix,
		Filename:    "site_default.zip",
		ContentType: "application/zip",
		AvailableAt: &availableAt,
	}
	service := munki.NewRepositoryService(munki.Dependencies{
		Hosts: serviceHostStore{host: &hosts.Host{
			ID:       1,
			Hardware: hosts.HostHardware{Serial: "C02MUNKI"},
		}},
		ClientResources: serviceClientResourcesStore{resource: resource},
		Objects:         serviceObjectStore{objects: map[int64]storage.Object{archive.ID: archive}},
	})

	for _, name := range []string{"C02MUNKI.zip", "site_default.zip"} {
		file, err := service.ResolveClientResources(context.Background(), name)
		if err != nil {
			t.Fatalf("ResolveClientResources(%q): %v", name, err)
		}
		if file.Key != "munki/clientresources/archives/9/site_default.zip" || file.ContentType != "application/zip" {
			t.Fatalf("ResolveClientResources(%q) file = %+v", name, file)
		}
	}
	for _, name := range []string{"C02OTHER.zip", "nested/C02MUNKI.zip", "not-a-zip"} {
		if _, err := service.ResolveClientResources(context.Background(), name); !errors.Is(err, munki.ErrNotFound) {
			t.Fatalf("ResolveClientResources(%q) error = %v, want ErrNotFound", name, err)
		}
	}
}

func TestResolveClientResourcesMapsUnconfiguredToNotFound(t *testing.T) {
	service := munki.NewRepositoryService(munki.Dependencies{
		ClientResources: serviceClientResourcesStore{err: dbutil.ErrNotFound},
	})
	if _, err := service.ResolveClientResources(
		context.Background(),
		"site_default.zip",
	); !errors.Is(
		err,
		munki.ErrNotFound,
	) {
		t.Fatalf("ResolveClientResources error = %v, want ErrNotFound", err)
	}
}

func TestManifestKeepsFeaturedDefaultAndOptionalActionsSeparate(t *testing.T) {
	service := munki.NewRepositoryService(munki.Dependencies{
		Hosts: serviceHostStore{host: &hosts.Host{ID: 1, Hardware: hosts.HostHardware{Serial: "C02MUNKI"}}},
		Software: servicePackageStore{packages: []munkisoftware.EffectivePackage{
			{
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionDefaultInstalls},
				Package:    packages.Package{SoftwareID: 1, SoftwareName: "DefaultApp", Version: "1.0"},
			},
			{
				SoftwareID: 2,
				Actions:    []munkisoftware.Action{munkisoftware.ActionFeaturedItems},
				Package:    packages.Package{SoftwareID: 2, SoftwareName: "FeaturedApp", Version: "1.0"},
			},
		}},
	})

	body, err := service.Manifest(context.Background(), "C02MUNKI")
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	var manifest struct {
		OptionalInstalls []string `plist:"optional_installs"`
		DefaultInstalls  []string `plist:"default_installs"`
		FeaturedItems    []string `plist:"featured_items"`
	}
	if _, err := plist.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("manifest plist: %v", err)
	}
	if len(manifest.OptionalInstalls) != 0 {
		t.Fatalf("optional_installs = %v, want empty", manifest.OptionalInstalls)
	}
	if !sameStrings(manifest.DefaultInstalls, []string{"DefaultApp"}) {
		t.Fatalf("default_installs = %v, want [DefaultApp]", manifest.DefaultInstalls)
	}
	if !sameStrings(manifest.FeaturedItems, []string{"FeaturedApp"}) {
		t.Fatalf("featured_items = %v, want [FeaturedApp]", manifest.FeaturedItems)
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

func TestIconHashesIncludesAvailableRepositoryIcons(t *testing.T) {
	iconID := int64(42)
	availableAt := time.Now()
	hash := strings.Repeat("a", 64)
	var requestedObjectIDs []int64
	service := munki.NewRepositoryService(munki.Dependencies{
		Packages: servicePackageStore{
			repositoryIconObjectIDs: []int64{iconID},
			listRepositoryErr:       errors.New("full repository scan should not be used"),
		},
		Objects: serviceObjectStore{
			objects: map[int64]storage.Object{
				iconID: {
					ID:          iconID,
					Filename:    "GoogleChrome.png",
					SHA256:      &hash,
					AvailableAt: &availableAt,
				},
			},
			requestedIDs: &requestedObjectIDs,
		},
	})

	body, err := service.IconHashes(context.Background())
	if err != nil {
		t.Fatalf("IconHashes: %v", err)
	}
	var hashes map[string]string
	if _, err := plist.Unmarshal(body, &hashes); err != nil {
		t.Fatalf("icon hashes plist: %v", err)
	}
	if got := hashes["42-GoogleChrome.png"]; got != hash {
		t.Fatalf("icon hash = %q, want %q", got, hash)
	}
	if len(requestedObjectIDs) != 1 || requestedObjectIDs[0] != iconID {
		t.Fatalf("requested object IDs = %v, want [%d]", requestedObjectIDs, iconID)
	}
}

type serviceObjectStore struct {
	objects      map[int64]storage.Object
	requestedIDs *[]int64
}

type serviceClientResourcesStore struct {
	resource *clientresources.ClientResources
	err      error
}

func (s serviceClientResourcesStore) Get(context.Context) (*clientresources.ClientResources, error) {
	return s.resource, s.err
}

func (s serviceObjectStore) ListByIDs(_ context.Context, ids []int64) (map[int64]storage.Object, error) {
	if s.requestedIDs != nil {
		*s.requestedIDs = append(*s.requestedIDs, ids...)
	}
	out := make(map[int64]storage.Object, len(ids))
	for _, id := range ids {
		if obj, ok := s.objects[id]; ok {
			out[id] = obj
		}
	}
	return out, nil
}

type servicePackageStore struct {
	packages                []munkisoftware.EffectivePackage
	repositoryPackages      []packages.Package
	repositoryIconObjectIDs []int64
	packagesByID            map[int64]packages.Package
	packagesByIconObjectID  map[int64][]packages.Package
	listRepositoryErr       error
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
	if s.listRepositoryErr != nil {
		return nil, s.listRepositoryErr
	}
	return s.repositoryPackages, nil
}

func (s servicePackageStore) ListRepositoryIconObjectIDs(context.Context) ([]int64, error) {
	return s.repositoryIconObjectIDs, nil
}

func (s servicePackageStore) PackagesByID(
	_ context.Context,
	ids []int64,
) ([]packages.Package, error) {
	pkgs := make([]packages.Package, 0, len(ids))
	for _, id := range ids {
		if pkg, ok := s.packagesByID[id]; ok {
			pkgs = append(pkgs, pkg)
		}
	}
	return pkgs, nil
}

func (s servicePackageStore) RepositoryPackagesByIconObjectID(
	_ context.Context,
	iconObjectID int64,
) ([]packages.Package, error) {
	return s.packagesByIconObjectID[iconObjectID], nil
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
