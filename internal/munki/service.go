package munki

import (
	"context"
	"errors"
	"slices"
	"strings"

	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const catalogName = "woodstar"

// ErrNotFound reports that a requested Munki repository object does not exist.
var ErrNotFound = errors.New("munki resource not found")

type hostResolver interface {
	GetByHardwareSerial(context.Context, string) (*hosts.Host, error)
}

type effectivePackageResolver interface {
	EffectivePackagesForHost(context.Context, int64) ([]munkisoftware.EffectivePackage, error)
}

type packageResolver interface {
	ListRepositoryPackages(context.Context) ([]packages.Package, error)
	ListRepositoryIconObjectIDs(context.Context) ([]int64, error)
	PackagesByID(context.Context, []int64) ([]packages.Package, error)
	RepositoryPackagesByIconObjectID(context.Context, int64) ([]packages.Package, error)
}

type objectResolver interface {
	ListByIDs(context.Context, []int64) (map[int64]storage.Object, error)
}

type clientResourcesResolver interface {
	Get(context.Context) (*clientresources.ClientResources, error)
}

// RepositoryService renders the Munki client-facing repository surface.
type RepositoryService struct {
	deps Dependencies
}

// Dependencies are the collaborators the Munki repository renderer needs.
type Dependencies struct {
	Hosts           hostResolver
	Software        effectivePackageResolver
	Packages        packageResolver
	Objects         objectResolver
	ClientResources clientResourcesResolver
}

// NewRepositoryService returns the Munki repository renderer.
func NewRepositoryService(deps Dependencies) *RepositoryService {
	return &RepositoryService{deps: deps}
}

func (s *RepositoryService) resolveManifestHostID(ctx context.Context, serial string) (int64, error) {
	host, err := s.deps.Hosts.GetByHardwareSerial(ctx, serial)
	if errors.Is(err, dbutil.ErrNotFound) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	return host.ID, nil
}

// Manifest returns the Munki manifest for the host serial in name.
func (s *RepositoryService) Manifest(ctx context.Context, name string) ([]byte, error) {
	hostID, err := s.resolveManifestHostID(ctx, name)
	if err != nil {
		return nil, err
	}
	pkgs, err := s.effectivePackages(ctx, hostID)
	if err != nil {
		return nil, err
	}
	manifest := renderedManifest{
		Catalogs:          []string{catalogName},
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
func (s *RepositoryService) Catalog(ctx context.Context, name string) ([]byte, error) {
	if name != catalogName {
		return nil, ErrNotFound
	}
	pkgs, err := s.deps.Packages.ListRepositoryPackages(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.catalogItems(ctx, pkgs)
	if err != nil {
		return nil, err
	}
	return encodePlist(items)
}

// IconHashes returns the available catalog icon hashes keyed by repository filename.
func (s *RepositoryService) IconHashes(ctx context.Context) ([]byte, error) {
	iconIDs, err := s.deps.Packages.ListRepositoryIconObjectIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(iconIDs) == 0 {
		return encodePlist(map[string]string{})
	}
	objects, err := s.deps.Objects.ListByIDs(ctx, iconIDs)
	if err != nil {
		return nil, err
	}

	hashes := make(map[string]string)
	for _, id := range iconIDs {
		obj, ok := objects[id]
		if !ok || !obj.Available() {
			continue
		}
		hashes[packages.IconName(obj)] = obj.SHA256Value()
	}
	return encodePlist(hashes)
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

// ResolvePackageFile resolves a package installer Munki path to the package
// identity and storage key for serving. The identity lets the delivery path mint
// a distribution grant; the key serves Woodstar-direct.
func (s *RepositoryService) ResolvePackageFile(
	ctx context.Context,
	key string,
) (PackageInstaller, error) {
	if key == "" {
		return PackageInstaller{}, ErrNotFound
	}
	packageID, ok := packages.ParseInstallerItemLocation(key)
	if !ok {
		return PackageInstaller{}, ErrNotFound
	}
	pkgs, err := s.deps.Packages.PackagesByID(ctx, []int64{packageID})
	if err != nil {
		return PackageInstaller{}, err
	}
	if len(pkgs) == 0 {
		return PackageInstaller{}, ErrNotFound
	}
	pkg := pkgs[0]
	if !pkg.Eligible || pkg.InstallerType == packages.InstallerTypeNoPkg {
		return PackageInstaller{}, ErrNotFound
	}
	objects, err := s.objectsForPackages(ctx, []packages.Package{pkg})
	if err != nil {
		return PackageInstaller{}, err
	}
	obj := objectByID(objects, pkg.InstallerObjectID)
	if obj == nil || !obj.Available() || packages.InstallerItemLocation(pkg, *obj) != key {
		return PackageInstaller{}, ErrNotFound
	}
	return PackageInstaller{
		PackageID:             pkg.ID,
		InstallerItemLocation: key,
		Key:                   obj.Key(),
		SHA256:                obj.SHA256Value(),
		SizeBytes:             obj.SizeBytesValue(),
	}, nil
}

// ResolveIconFile resolves a software icon name to the private object key for
// serving.
func (s *RepositoryService) ResolveIconFile(
	ctx context.Context,
	key string,
) (string, error) {
	if key == "" {
		return "", ErrNotFound
	}
	iconObjectID, ok := packages.ParseIconName(key)
	if !ok {
		return "", ErrNotFound
	}
	pkgs, err := s.deps.Packages.RepositoryPackagesByIconObjectID(ctx, iconObjectID)
	if err != nil {
		return "", err
	}
	if len(pkgs) == 0 {
		return "", ErrNotFound
	}
	objects, err := s.deps.Objects.ListByIDs(ctx, []int64{iconObjectID})
	if err != nil {
		return "", err
	}
	obj, ok := objects[iconObjectID]
	if !ok || !obj.Available() || packages.IconName(obj) != key {
		return "", ErrNotFound
	}
	return obj.Key(), nil
}

// ResolveClientResources resolves a configured archive for Munki's host-specific
// request or its site_default.zip fallback.
func (s *RepositoryService) ResolveClientResources(ctx context.Context, name string) (string, error) {
	if name != "site_default.zip" {
		serial, ok := strings.CutSuffix(name, ".zip")
		if !ok || serial == "" || strings.Contains(serial, "/") {
			return "", ErrNotFound
		}
		if _, err := s.resolveManifestHostID(ctx, serial); err != nil {
			return "", err
		}
	}
	resource, err := s.deps.ClientResources.Get(ctx)
	if errors.Is(err, dbutil.ErrNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	objects, err := s.deps.Objects.ListByIDs(ctx, []int64{resource.ArchiveObjectID})
	if err != nil {
		return "", err
	}
	archive, ok := objects[resource.ArchiveObjectID]
	if !ok || archive.Prefix != clientresources.ArchiveObjectPrefix || !archive.Available() {
		return "", ErrNotFound
	}
	return archive.Key(), nil
}

func (s *RepositoryService) effectivePackages(
	ctx context.Context,
	hostID int64,
) ([]munkisoftware.EffectivePackage, error) {
	return s.deps.Software.EffectivePackagesForHost(ctx, hostID)
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
		return packages.MunkiVersionedSoftwareName(pkg.Package.SoftwareName, pkg.Package.Version)
	}
	return pkg.Package.SoftwareName
}

func (s *RepositoryService) catalogItems(
	ctx context.Context,
	pkgs []packages.Package,
) ([]any, error) {
	objects, err := s.objectsForPackages(ctx, pkgs)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(pkgs))
	for _, pkg := range pkgs {
		items = append(items, packages.Pkginfo(pkg, packageObjects(pkg, objects)))
	}
	return items, nil
}

func (s *RepositoryService) objectsForPackages(
	ctx context.Context,
	pkgs []packages.Package,
) (map[int64]storage.Object, error) {
	ids := make([]int64, 0, len(pkgs)*2)
	for _, pkg := range pkgs {
		ids = appendObjectID(ids, pkg.InstallerObjectID)
		ids = appendObjectID(ids, pkg.SoftwareIconObjectID)
	}
	if len(ids) == 0 {
		return map[int64]storage.Object{}, nil
	}
	return s.deps.Objects.ListByIDs(ctx, ids)
}

func packageObjects(
	pkg packages.Package,
	objects map[int64]storage.Object,
) packages.PkginfoObjects {
	return packages.PkginfoObjects{
		Installer: objectByID(objects, pkg.InstallerObjectID),
		Icon:      objectByID(objects, pkg.SoftwareIconObjectID),
	}
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
