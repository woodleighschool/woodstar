package munki

import (
	"context"
	"errors"
	"slices"
	"strings"

	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// ErrNotFound reports that a requested Munki repository object does not exist.
var ErrNotFound = errors.New("munki resource not found")

type hostResolver interface {
	GetByHardwareSerial(context.Context, string) (*hosts.Host, error)
}

type packageResolver interface {
	EffectivePackagesForHost(context.Context, int64) ([]munkisoftware.EffectivePackage, error)
}

type objectResolver interface {
	ListByIDs(context.Context, []int64) (map[int64]storage.Object, error)
}

// ClientHost identifies the existing Woodstar host making a Munki request.
type ClientHost struct {
	ID          int64
	Serial      string
	DisplayName string
}

// RepositoryService renders the Munki client-facing repository surface.
type RepositoryService struct {
	hosts    hostResolver
	packages packageResolver
	objects  objectResolver
}

// Dependencies are the collaborators the Munki repository renderer needs.
type Dependencies struct {
	Hosts    hostResolver
	Packages packageResolver
	Objects  objectResolver
}

// NewRepositoryService returns the Munki repository renderer.
func NewRepositoryService(deps Dependencies) *RepositoryService {
	return &RepositoryService{
		hosts:    deps.Hosts,
		packages: deps.Packages,
		objects:  deps.Objects,
	}
}

// ResolveClient resolves the Munki request identity to an existing host.
func (s *RepositoryService) ResolveClient(ctx context.Context, serial string) (ClientHost, error) {
	host, err := s.hosts.GetByHardwareSerial(ctx, serial)
	if errors.Is(err, dbutil.ErrNotFound) {
		return ClientHost{}, ErrNotFound
	}
	if err != nil {
		return ClientHost{}, err
	}
	return ClientHost{
		ID:          host.ID,
		Serial:      host.Hardware.Serial,
		DisplayName: host.DisplayName,
	}, nil
}

// Manifest returns a Munki manifest plist for name.
func (s *RepositoryService) Manifest(ctx context.Context, client ClientHost, name string) ([]byte, error) {
	if strings.TrimSpace(name) != client.Serial {
		return nil, ErrNotFound
	}
	pkgs, err := s.effectivePackages(ctx, client.ID)
	if err != nil {
		return nil, err
	}
	manifest := renderedManifest{
		Catalogs:          []string{"production"},
		ManagedInstalls:   []string{},
		ManagedUninstalls: []string{},
		ManagedUpdates:    []string{},
		OptionalInstalls:  []string{},
		DefaultInstalls:   []string{},
		FeaturedItems:     []string{},
	}
	for _, pkg := range pkgs {
		addManifestPackage(&manifest, pkg)
	}
	return encodePlist(manifest)
}

// Catalog returns a Munki catalog plist for name.
func (s *RepositoryService) Catalog(ctx context.Context, client ClientHost, name string) ([]byte, error) {
	if name != "production" {
		return nil, ErrNotFound
	}
	pkgs, err := s.effectivePackages(ctx, client.ID)
	if err != nil {
		return nil, err
	}
	items, err := s.catalogItems(ctx, pkgs)
	if err != nil {
		return nil, err
	}
	return encodePlist(items)
}

// PackageInstaller is a resolved package installer: the stable package id, the
// Munki repository path, the storage key to serve, and the integrity a
// distribution grant binds to.
type PackageInstaller struct {
	PackageID             int64
	InstallerItemLocation string
	Key                   string
	SHA256                string
	SizeBytes             int64
}

// ResolvePackageFile authorizes a package installer Munki path for a client and
// returns the package identity and storage key for serving. The identity lets
// the delivery path mint a distribution grant; the key serves Woodstar-direct.
func (s *RepositoryService) ResolvePackageFile(
	ctx context.Context,
	client ClientHost,
	key string,
) (PackageInstaller, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return PackageInstaller{}, ErrNotFound
	}
	pkgs, err := s.effectivePackages(ctx, client.ID)
	if err != nil {
		return PackageInstaller{}, err
	}
	objects, err := s.objectsForPackages(ctx, pkgs)
	if err != nil {
		return PackageInstaller{}, err
	}
	for _, pkg := range pkgs {
		if pkg.Package.InstallerType == packages.InstallerTypeNoPkg {
			continue
		}
		obj := objectByID(objects, pkg.Package.InstallerObjectID)
		if obj == nil || packages.InstallerItemLocation(pkg.Package, *obj) != key {
			continue
		}
		return PackageInstaller{
			PackageID:             pkg.Package.ID,
			InstallerItemLocation: key,
			Key:                   obj.Key(),
			SHA256:                objectSHA256(*obj),
			SizeBytes:             objectSize(*obj),
		}, nil
	}
	return PackageInstaller{}, ErrNotFound
}

// ResolveIconFile authorizes a software icon name for a client and returns
// the private object key for serving.
func (s *RepositoryService) ResolveIconFile(
	ctx context.Context,
	client ClientHost,
	key string,
) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", ErrNotFound
	}
	pkgs, err := s.effectivePackages(ctx, client.ID)
	if err != nil {
		return "", err
	}
	objects, err := s.objectsForPackages(ctx, pkgs)
	if err != nil {
		return "", err
	}
	for _, pkg := range pkgs {
		if obj := objectByID(objects, pkg.SoftwareIconObjectID); obj != nil &&
			packages.IconName(*obj) == key {
			return obj.Key(), nil
		}
	}
	return "", ErrNotFound
}

func (s *RepositoryService) effectivePackages(
	ctx context.Context,
	hostID int64,
) ([]munkisoftware.EffectivePackage, error) {
	return s.packages.EffectivePackagesForHost(ctx, hostID)
}

func addManifestPackage(manifest *renderedManifest, pkg munkisoftware.EffectivePackage) {
	name := manifestItemName(pkg)
	for _, action := range pkg.Actions {
		switch action {
		case munkisoftware.ActionManagedInstalls:
			manifest.ManagedInstalls = appendUnique(manifest.ManagedInstalls, name)
		case munkisoftware.ActionManagedUninstalls:
			manifest.ManagedUninstalls = appendUnique(manifest.ManagedUninstalls, name)
		case munkisoftware.ActionManagedUpdates:
			manifest.ManagedUpdates = appendUnique(manifest.ManagedUpdates, name)
		case munkisoftware.ActionOptionalInstalls:
			manifest.OptionalInstalls = appendUnique(manifest.OptionalInstalls, name)
		case munkisoftware.ActionDefaultInstalls:
			manifest.DefaultInstalls = appendUnique(manifest.DefaultInstalls, name)
		case munkisoftware.ActionFeaturedItems:
			manifest.FeaturedItems = appendUnique(manifest.FeaturedItems, name)
		}
	}
}

func manifestItemName(pkg munkisoftware.EffectivePackage) string {
	if pkg.Selector.Strategy == munkisoftware.PackageSpecific {
		return packages.MunkiVersionedName(pkg.Package)
	}
	return packages.MunkiName(pkg.Package)
}

func (s *RepositoryService) catalogItems(
	ctx context.Context,
	effective []munkisoftware.EffectivePackage,
) ([]any, error) {
	objects, err := s.objectsForPackages(ctx, effective)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(effective))
	seen := make(map[int64]bool, len(effective))
	for _, pkg := range effective {
		if seen[pkg.Package.ID] {
			continue
		}
		seen[pkg.Package.ID] = true
		items = append(items, packages.Pkginfo(pkg.Package, packageObjects(pkg, objects)))
	}
	return items, nil
}

func (s *RepositoryService) objectsForPackages(
	ctx context.Context,
	effective []munkisoftware.EffectivePackage,
) (map[int64]storage.Object, error) {
	ids := make([]int64, 0, len(effective)*3)
	for _, pkg := range effective {
		ids = appendObjectID(ids, pkg.Package.InstallerObjectID)
		ids = appendObjectID(ids, pkg.SoftwareIconObjectID)
	}
	if len(ids) == 0 {
		return map[int64]storage.Object{}, nil
	}
	return s.objects.ListByIDs(ctx, ids)
}

func packageObjects(
	pkg munkisoftware.EffectivePackage,
	objects map[int64]storage.Object,
) packages.PkginfoObjects {
	return packages.PkginfoObjects{
		Installer: objectByID(objects, pkg.Package.InstallerObjectID),
		Icon:      objectByID(objects, pkg.SoftwareIconObjectID),
	}
}

func objectSHA256(obj storage.Object) string {
	if obj.SHA256 == nil {
		return ""
	}
	return *obj.SHA256
}

func objectSize(obj storage.Object) int64 {
	if obj.SizeBytes == nil {
		return 0
	}
	return *obj.SizeBytes
}

func objectByID(objects map[int64]storage.Object, id *int64) *storage.Object {
	if id == nil {
		return nil
	}
	obj, ok := objects[*id]
	if !ok {
		return nil
	}
	return &obj
}

func appendObjectID(ids []int64, id *int64) []int64 {
	if id == nil {
		return ids
	}
	return append(ids, *id)
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func encodePlist(value any) ([]byte, error) {
	return plist.Marshal(value, plist.XMLFormat)
}

type renderedManifest struct {
	Catalogs          []string `plist:"catalogs"`
	ManagedInstalls   []string `plist:"managed_installs"`
	ManagedUninstalls []string `plist:"managed_uninstalls"`
	ManagedUpdates    []string `plist:"managed_updates"`
	OptionalInstalls  []string `plist:"optional_installs"`
	DefaultInstalls   []string `plist:"default_installs"`
	FeaturedItems     []string `plist:"featured_items"`
}
