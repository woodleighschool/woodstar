package munki_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"howett.net/plist"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

func TestResolvePackageFileUsesEmbeddedPackageID(t *testing.T) {
	installerID := int64(42)
	installerSize := int64(1024)
	installerSHA := strings.Repeat("a", 64)
	availableAt := time.Now()
	store := servicePackageStore{
		packages: []munkisoftware.EffectivePackage{{Package: packages.Package{ID: 10}}},
		repositoryPackages: []packages.Package{{
			ID:       10,
			Software: packages.PackageSoftware{ID: 10, Name: "GoogleChrome"},
		}},
		packagesByID: map[int64]packages.Package{10: {
			ID:                10,
			Software:          packages.PackageSoftware{Name: "GoogleChrome"},
			InstallerType:     packages.InstallerTypePkg,
			InstallerObjectID: &installerID,
		}},
	}
	objects := serviceObjectStore{objects: map[int64]bloby.Object{
		installerID: {
			ID:          installerID,
			Prefix:      packages.ObjectPrefix,
			Filename:    "GoogleChrome.pkg",
			ContentType: "application/octet-stream",
			SizeBytes:   &installerSize,
			SHA256:      &installerSHA,
			AvailableAt: &availableAt,
		},
	}}
	service := munki.NewRepositoryService(munki.Dependencies{Software: store, Packages: store, Objects: objects})

	installer, err := service.ResolvePackageFile(context.Background(), 1, "packages/10/installer/GoogleChrome.pkg")
	if err != nil {
		t.Fatalf("ResolvePackageFile allowed package: %v", err)
	}
	if installer.Object.ID != installerID || installer.Object.Prefix != packages.ObjectPrefix ||
		installer.Object.Filename != "GoogleChrome.pkg" {
		t.Fatalf("installer object = %+v, want the package installer", installer.Object)
	}
	if installer.PackageID != 10 {
		t.Fatalf("package id = %d, want 10", installer.PackageID)
	}
	if installer.Object.ContentType != "application/octet-stream" {
		t.Fatalf("content type = %q, want application/octet-stream", installer.Object.ContentType)
	}

	_, err = service.ResolvePackageFile(context.Background(), 1, "munki/packages/99/Blocked.pkg")
	if !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("blocked key error = %v, want ErrNotFound", err)
	}
}

func TestResolveIconFileUsesEmbeddedObjectID(t *testing.T) {
	iconID := int64(42)
	availableAt := time.Now()
	store := servicePackageStore{
		packages: []munkisoftware.EffectivePackage{{Package: packages.Package{ID: 10}}},
		repositoryPackages: []packages.Package{{
			ID:            10,
			Software:      packages.PackageSoftware{Name: "GoogleChrome", IconObjectID: &iconID},
			InstallerType: packages.InstallerTypeNoPkg,
		}},
	}
	objects := serviceObjectStore{objects: map[int64]bloby.Object{
		iconID: {
			ID:          iconID,
			Prefix:      "munki/icons",
			Filename:    "GoogleChrome.png",
			ContentType: "image/png",
			AvailableAt: &availableAt,
		},
	}}
	service := munki.NewRepositoryService(munki.Dependencies{Software: store, Packages: store, Objects: objects})

	file, err := service.ResolveIconFile(context.Background(), 1, "42-GoogleChrome.png")
	if err != nil {
		t.Fatalf("ResolveIconFile allowed icon: %v", err)
	}
	if file.ID != iconID || file.Prefix != "munki/icons" ||
		file.Filename != "GoogleChrome.png" || file.ContentType != "image/png" {
		t.Fatalf("file = %+v, want canonical icon storage metadata", file)
	}

	_, err = service.ResolveIconFile(context.Background(), 1, "42-Other.png")
	if !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("mismatched icon error = %v, want ErrNotFound", err)
	}
}

func TestRepositoryProjectionExpandsRelationsTransitively(t *testing.T) {
	root := repositoryTestPackage(10, 1)
	root.Requires = []packages.PackageReference{{SoftwareID: 2}}
	dependency := repositoryTestPackage(20, 2)
	dependency.Requires = []packages.PackageReference{{SoftwareID: 3, PackageID: 31}}

	assertRepositoryCatalogIDs(t, []munkisoftware.EffectivePackage{{Package: root}}, []packages.Package{
		repositoryTestPackage(99, 9),
		dependency,
		repositoryTestPackage(31, 3),
		root,
	}, []int64{20, 31, 10})
}

func TestRepositoryProjectionIncludesReverseUpdatePackages(t *testing.T) {
	root := repositoryTestPackage(10, 1)
	update := repositoryTestPackage(40, 4)
	update.UpdateFor = []packages.PackageReference{{SoftwareID: root.Software.ID}}

	assertRepositoryCatalogIDs(t, []munkisoftware.EffectivePackage{{Package: root}}, []packages.Package{
		repositoryTestPackage(99, 9),
		update,
		root,
	}, []int64{40, 10})
}

func TestRepositoryProjectionExcludesReverseUpdatePinnedToAnotherPackage(t *testing.T) {
	root := repositoryTestPackage(10, 1)
	otherVersion := repositoryTestPackage(11, root.Software.ID)
	update := repositoryTestPackage(40, 4)
	update.UpdateFor = []packages.PackageReference{{
		SoftwareID: root.Software.ID,
		PackageID:  otherVersion.ID,
	}}

	assertRepositoryCatalogIDs(t, []munkisoftware.EffectivePackage{{Package: root}}, []packages.Package{
		otherVersion,
		update,
		root,
	}, []int64{10})
}

func TestRepositoryProjectionHonorsPinnedReferences(t *testing.T) {
	root := repositoryTestPackage(10, 1)
	root.Requires = []packages.PackageReference{{SoftwareID: 3, PackageID: 31}}

	assertRepositoryCatalogIDs(t, []munkisoftware.EffectivePackage{{Package: root}}, []packages.Package{
		repositoryTestPackage(30, 3),
		repositoryTestPackage(31, 3),
		root,
	}, []int64{31, 10})
}

func TestRepositoryProjectionTerminatesCyclesAndPreservesRepositoryOrder(t *testing.T) {
	root := repositoryTestPackage(10, 1)
	root.Requires = []packages.PackageReference{{SoftwareID: 2}}
	dependency := repositoryTestPackage(20, 2)
	dependency.Requires = []packages.PackageReference{{SoftwareID: 1, PackageID: root.ID}}

	assertRepositoryCatalogIDs(t, []munkisoftware.EffectivePackage{{Package: root}}, []packages.Package{
		dependency,
		root,
	}, []int64{20, 10})
}

func TestRepositoryResourceResolutionRejectsPackagesAndIconsOutsideProjection(t *testing.T) {
	allowedIconID := int64(42)
	blockedIconID := int64(43)
	allowed := repositoryTestPackage(10, 1)
	allowed.Software.IconObjectID = &allowedIconID
	blocked := repositoryTestPackage(20, 2)
	blocked.Software.IconObjectID = &blockedIconID
	installerID := int64(44)
	blocked.InstallerObjectID = &installerID

	service := munki.NewRepositoryService(munki.Dependencies{
		Software: servicePackageStore{packages: []munkisoftware.EffectivePackage{{Package: allowed}}},
		Packages: servicePackageStore{
			repositoryPackages: []packages.Package{allowed, blocked},
			packagesByID:       map[int64]packages.Package{blocked.ID: blocked},
		},
		Objects: serviceObjectStore{},
	})

	if _, err := service.ResolvePackageFile(context.Background(), 1, "packages/20/installer/blocked.pkg"); !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("ResolvePackageFile outside projection error = %v, want ErrNotFound", err)
	}
	if _, err := service.ResolveIconFile(context.Background(), 1, "43-blocked.png"); !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("ResolveIconFile outside projection error = %v, want ErrNotFound", err)
	}
}

func repositoryTestPackage(id, softwareID int64) packages.Package {
	return packages.Package{
		ID:            id,
		Software:      packages.PackageSoftware{ID: softwareID, Name: fmt.Sprintf("package-%d", id)},
		Version:       "1.0",
		InstallerType: packages.InstallerTypeNoPkg,
	}
}

func assertRepositoryCatalogIDs(
	t *testing.T,
	effective []munkisoftware.EffectivePackage,
	repository []packages.Package,
	want []int64,
) {
	t.Helper()
	service := munki.NewRepositoryService(munki.Dependencies{
		Software: servicePackageStore{packages: effective},
		Packages: servicePackageStore{repositoryPackages: repository},
		Objects:  serviceObjectStore{},
	})
	body, err := service.Catalog(context.Background(), 1, "woodstar")
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	var items []struct {
		Name string `plist:"name"`
	}
	if _, err := plist.Unmarshal(body, &items); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	got := make([]int64, 0, len(items))
	for _, item := range items {
		var id int64
		if _, err := fmt.Sscanf(item.Name, "package-%d", &id); err != nil {
			t.Fatalf("catalog package name %q: %v", item.Name, err)
		}
		got = append(got, id)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("catalog package IDs = %v, want %v", got, want)
	}
}

func TestResolveClientResourcesUsesResolvedHost(t *testing.T) {
	availableAt := time.Now()
	resource := &clientresources.ClientResources{
		ID:              1,
		ArchiveObjectID: 9,
	}
	archive := bloby.Object{
		ID:          9,
		Prefix:      clientresources.ArchiveObjectPrefix,
		Filename:    "site_default.zip",
		ContentType: "application/zip",
		AvailableAt: &availableAt,
	}
	service := munki.NewRepositoryService(munki.Dependencies{
		ClientResources: serviceClientResourcesStore{resource: resource},
		Objects:         serviceObjectStore{objects: map[int64]bloby.Object{archive.ID: archive}},
	})

	for _, name := range []string{"C02MUNKI.zip", "site_default.zip", "nested/C02MUNKI.zip"} {
		file, err := service.ResolveClientResources(context.Background(), 1, name)
		if err != nil {
			t.Fatalf("ResolveClientResources(%q): %v", name, err)
		}
		if file.ID != archive.ID || file.Prefix != clientresources.ArchiveObjectPrefix ||
			file.Filename != archive.Filename || file.ContentType != "application/zip" {
			t.Fatalf("ResolveClientResources(%q) file = %+v", name, file)
		}
	}
}

func TestResolveClientResourcesMapsUnconfiguredToNotFound(t *testing.T) {
	service := munki.NewRepositoryService(munki.Dependencies{
		ClientResources: serviceClientResourcesStore{err: fault.ErrNotFound},
	})
	if _, err := service.ResolveClientResources(
		context.Background(),
		1,
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
		Software: servicePackageStore{packages: []munkisoftware.EffectivePackage{
			{
				Actions: []munkisoftware.Action{munkisoftware.ActionDefaultInstalls},
				Package: packages.Package{
					Software: packages.PackageSoftware{ID: 1, Name: "DefaultApp"},
					Version:  "1.0",
				},
			},
			{
				Actions: []munkisoftware.Action{munkisoftware.ActionFeaturedItems},
				Package: packages.Package{
					Software: packages.PackageSoftware{ID: 2, Name: "FeaturedApp"},
					Version:  "1.0",
				},
			},
		}},
	})

	body, err := service.Manifest(context.Background(), 42)
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
	if !slices.Equal(manifest.DefaultInstalls, []string{"DefaultApp"}) {
		t.Fatalf("default_installs = %v, want [DefaultApp]", manifest.DefaultInstalls)
	}
	if !slices.Equal(manifest.FeaturedItems, []string{"FeaturedApp"}) {
		t.Fatalf("featured_items = %v, want [FeaturedApp]", manifest.FeaturedItems)
	}
}

func TestManifestUsesResolvedHostID(t *testing.T) {
	var gotHostID int64
	service := munki.NewRepositoryService(munki.Dependencies{
		Software: servicePackageStore{requestedHostID: &gotHostID},
	})

	if _, err := service.Manifest(context.Background(), 42); err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	if gotHostID != 42 {
		t.Fatalf("effective packages host ID = %d, want 42", gotHostID)
	}
}

func TestCatalogRequiresWoodstarName(t *testing.T) {
	service := munki.NewRepositoryService(munki.Dependencies{
		Software: servicePackageStore{},
		Packages: servicePackageStore{},
		Objects:  serviceObjectStore{},
	})

	if _, err := service.Catalog(context.Background(), 1, "testing"); !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("Catalog wrong name error = %v, want ErrNotFound", err)
	}
	if _, err := service.Catalog(context.Background(), 1, "woodstar"); err != nil {
		t.Fatalf("Catalog woodstar error = %v, want nil", err)
	}
}

func TestIconHashesIncludesAvailableRepositoryIcons(t *testing.T) {
	iconID := int64(42)
	availableAt := time.Now()
	hash := strings.Repeat("a", 64)
	var requestedObjectIDs []int64
	service := munki.NewRepositoryService(munki.Dependencies{
		Software: servicePackageStore{packages: []munkisoftware.EffectivePackage{{
			Package: packages.Package{ID: 10},
		}}},
		Packages: servicePackageStore{repositoryPackages: []packages.Package{{
			ID:       10,
			Software: packages.PackageSoftware{ID: 10, Name: "GoogleChrome", IconObjectID: &iconID},
		}}},
		Objects: serviceObjectStore{
			objects: map[int64]bloby.Object{
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

	body, err := service.IconHashes(context.Background(), 1)
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
	objects      map[int64]bloby.Object
	requestedIDs *[]int64
}

type serviceClientResourcesStore struct {
	resource *clientresources.ClientResources
	err      error
}

func (s serviceClientResourcesStore) GetByID(
	context.Context,
	int64,
) (*clientresources.ClientResources, error) {
	return s.resource, s.err
}

func (s serviceObjectStore) ListByIDs(_ context.Context, ids []int64) (map[int64]bloby.Object, error) {
	if s.requestedIDs != nil {
		*s.requestedIDs = append(*s.requestedIDs, ids...)
	}
	out := make(map[int64]bloby.Object, len(ids))
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
	packagesByID       map[int64]packages.Package
	listRepositoryErr  error
	requestedHostID    *int64
}

func (s servicePackageStore) EffectivePackagesForHost(
	_ context.Context,
	hostID int64,
) ([]munkisoftware.EffectivePackage, error) {
	if s.requestedHostID != nil {
		*s.requestedHostID = hostID
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
