package munki

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"howett.net/plist"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

const catalogName = "woodstar"

// ErrNotFound reports that a requested Munki repository object does not exist.
var ErrNotFound = errors.New("munki resource not found")

type effectivePackageResolver interface {
	EffectivePackagesForHost(ctx context.Context, hostID int64) ([]munkisoftware.EffectivePackage, error)
}

type packageResolver interface {
	ListRepositoryPackages(ctx context.Context) ([]packages.Package, error)
	PackagesByID(ctx context.Context, packageIDs []int64) ([]packages.Package, error)
}

type objectResolver interface {
	ListByIDs(ctx context.Context, objectIDs []int64) (map[int64]bloby.Object, error)
}

type clientResourcesResolver interface {
	GetByID(ctx context.Context, id int64) (*clientresources.ClientResources, error)
}

// RepositoryService renders the Munki client-facing repository surface.
type RepositoryService struct {
	deps Dependencies
}

// Dependencies are the collaborators the Munki repository renderer needs.
type Dependencies struct {
	Software        effectivePackageResolver
	Packages        packageResolver
	Objects         objectResolver
	ClientResources clientResourcesResolver
}

// NewRepositoryService returns the Munki repository renderer.
func NewRepositoryService(deps Dependencies) *RepositoryService {
	return &RepositoryService{deps: deps}
}

// Manifest returns the Munki manifest for hostID.
func (s *RepositoryService) Manifest(ctx context.Context, hostID int64) ([]byte, error) {
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

// Catalog returns a Munki catalog plist for hostID and name.
func (s *RepositoryService) Catalog(ctx context.Context, hostID int64, name string) ([]byte, error) {
	if name != catalogName {
		return nil, ErrNotFound
	}
	projection, err := s.projection(ctx, hostID)
	if err != nil {
		return nil, err
	}
	items, err := s.catalogItems(ctx, projection.packages)
	if err != nil {
		return nil, err
	}
	return encodePlist(items)
}

// IconHashes returns the available catalog icon hashes for hostID keyed by repository filename.
func (s *RepositoryService) IconHashes(ctx context.Context, hostID int64) ([]byte, error) {
	projection, err := s.projection(ctx, hostID)
	if err != nil {
		return nil, err
	}
	iconIDs := mapKeys(projection.iconObjectIDs)
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

// PackageInstaller is a package identity and its canonical stored installer.
type PackageInstaller struct {
	PackageID             int64
	InstallerItemLocation string
	Object                bloby.Object
}

// ResolvePackageFile resolves a package installer Munki path to the package
// identity and storage key for serving. The identity lets the delivery path mint
// a distribution grant; the key serves the file directly.
func (s *RepositoryService) ResolvePackageFile(
	ctx context.Context,
	hostID int64,
	key string,
) (PackageInstaller, error) {
	if key == "" {
		return PackageInstaller{}, ErrNotFound
	}
	packageID, ok := packages.ParseInstallerItemLocation(key)
	if !ok {
		return PackageInstaller{}, ErrNotFound
	}
	projection, err := s.projection(ctx, hostID)
	if err != nil {
		return PackageInstaller{}, err
	}
	if _, allowed := projection.packageIDs[packageID]; !allowed {
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
	if pkg.InstallerType == packages.InstallerTypeNoPkg {
		return PackageInstaller{}, ErrNotFound
	}
	objects, err := s.objectsForPackages(ctx, []packages.Package{pkg})
	if err != nil {
		return PackageInstaller{}, err
	}
	obj := objectByID(objects, pkg.InstallerObjectID)
	if obj == nil || !obj.Available() || obj.SizeBytes == nil || obj.SHA256 == nil {
		return PackageInstaller{}, fmt.Errorf("package %d installer object is not finalized", pkg.ID)
	}
	if packages.InstallerItemLocation(pkg, *obj) != key {
		return PackageInstaller{}, ErrNotFound
	}
	return PackageInstaller{
		PackageID:             pkg.ID,
		InstallerItemLocation: key,
		Object:                *obj,
	}, nil
}

// ResolveIconFile resolves a software icon name to the private object key for
// serving.
func (s *RepositoryService) ResolveIconFile(
	ctx context.Context,
	hostID int64,
	key string,
) (bloby.Object, error) {
	if key == "" {
		return bloby.Object{}, ErrNotFound
	}
	iconObjectID, ok := packages.ParseIconName(key)
	if !ok {
		return bloby.Object{}, ErrNotFound
	}
	projection, err := s.projection(ctx, hostID)
	if err != nil {
		return bloby.Object{}, err
	}
	if _, allowed := projection.iconObjectIDs[iconObjectID]; !allowed {
		return bloby.Object{}, ErrNotFound
	}
	objects, err := s.deps.Objects.ListByIDs(ctx, []int64{iconObjectID})
	if err != nil {
		return bloby.Object{}, err
	}
	obj, ok := objects[iconObjectID]
	if !ok || !obj.Available() || packages.IconName(obj) != key {
		return bloby.Object{}, ErrNotFound
	}
	return obj, nil
}

// ResolveClientResources resolves the configured archive for hostID.
func (s *RepositoryService) ResolveClientResources(
	ctx context.Context,
	_ int64,
	_ string,
) (bloby.Object, error) {
	const effectiveClientResourcesID int64 = 1
	resource, err := s.deps.ClientResources.GetByID(ctx, effectiveClientResourcesID)
	if errors.Is(err, fault.ErrNotFound) {
		return bloby.Object{}, ErrNotFound
	}
	if err != nil {
		return bloby.Object{}, err
	}
	objects, err := s.deps.Objects.ListByIDs(ctx, []int64{resource.ArchiveObjectID})
	if err != nil {
		return bloby.Object{}, err
	}
	archive, ok := objects[resource.ArchiveObjectID]
	if !ok || archive.Prefix != clientresources.ArchiveObjectPrefix || !archive.Available() {
		return bloby.Object{}, ErrNotFound
	}
	return archive, nil
}

func (s *RepositoryService) effectivePackages(
	ctx context.Context,
	hostID int64,
) ([]munkisoftware.EffectivePackage, error) {
	return s.deps.Software.EffectivePackagesForHost(ctx, hostID)
}

func (s *RepositoryService) projection(ctx context.Context, hostID int64) (repositoryProjection, error) {
	effective, err := s.effectivePackages(ctx, hostID)
	if err != nil {
		return repositoryProjection{}, err
	}
	repository, err := s.deps.Packages.ListRepositoryPackages(ctx)
	if err != nil {
		return repositoryProjection{}, err
	}
	return buildRepositoryProjection(effective, repository), nil
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
		return packages.MunkiVersionedSoftwareName(pkg.Package.Software.Name, pkg.Package.Version)
	}
	return pkg.Package.Software.Name
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
		item, err := packages.Pkginfo(pkg, packageObjects(pkg, objects))
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *RepositoryService) objectsForPackages(
	ctx context.Context,
	pkgs []packages.Package,
) (map[int64]bloby.Object, error) {
	ids := make([]int64, 0, len(pkgs)*2)
	for _, pkg := range pkgs {
		ids = appendObjectID(ids, pkg.InstallerObjectID)
		ids = appendObjectID(ids, pkg.Software.IconObjectID)
	}
	if len(ids) == 0 {
		return map[int64]bloby.Object{}, nil
	}
	return s.deps.Objects.ListByIDs(ctx, ids)
}

func packageObjects(
	pkg packages.Package,
	objects map[int64]bloby.Object,
) packages.PkginfoObjects {
	return packages.PkginfoObjects{
		Installer: objectByID(objects, pkg.InstallerObjectID),
		Icon:      objectByID(objects, pkg.Software.IconObjectID),
	}
}

func objectByID(objects map[int64]bloby.Object, id *int64) *bloby.Object {
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

func mapKeys(values map[int64]struct{}) []int64 {
	keys := make([]int64, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	slices.Sort(keys)
	return keys
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
