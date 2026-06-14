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
)

// ErrNotFound reports that a requested Munki repository object does not exist.
var ErrNotFound = errors.New("munki resource not found")

type hostResolver interface {
	GetByHardwareSerial(context.Context, string) (*hosts.Host, error)
}

type packageResolver interface {
	EffectivePackagesForHost(context.Context, int64) ([]munkisoftware.EffectivePackage, error)
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
}

// Dependencies are the collaborators the Munki repository renderer needs.
type Dependencies struct {
	Hosts    hostResolver
	Packages packageResolver
}

// NewRepositoryService returns the Munki repository renderer.
func NewRepositoryService(deps Dependencies) *RepositoryService {
	return &RepositoryService{
		hosts:    deps.Hosts,
		packages: deps.Packages,
	}
}

// ResolveClient resolves the Munki request identity to an existing host.
func (s *RepositoryService) ResolveClient(ctx context.Context, serial string) (ClientHost, error) {
	host, err := s.hosts.GetByHardwareSerial(ctx, strings.TrimSpace(serial))
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
	if !validResourcePath(name) {
		return nil, ErrNotFound
	}
	displayName := client.DisplayName
	if displayName == "" {
		displayName = client.Serial
	}
	pkgs, err := s.effectivePackages(ctx, client.ID)
	if err != nil {
		return nil, err
	}
	manifest := renderedManifest{
		Catalogs:          []string{"production"},
		DisplayName:       displayName,
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
	if name != "production" || !validResourceName(name) {
		return nil, ErrNotFound
	}
	pkgs, err := s.effectivePackages(ctx, client.ID)
	if err != nil {
		return nil, err
	}
	return encodePlist(s.catalogItems(pkgs))
}

// ResolvePackageArtifact authorizes a package installer/uninstaller storage key
// for a client and returns it for serving.
func (s *RepositoryService) ResolvePackageArtifact(
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
	for _, pkg := range pkgs {
		if pkg.Package.InstallerObjectLocation == key || pkg.Package.UninstallerObjectLocation == key {
			return key, nil
		}
	}
	return "", ErrNotFound
}

// ResolveIconArtifact authorizes a software icon storage key for a client.
func (s *RepositoryService) ResolveIconArtifact(
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
	for _, pkg := range pkgs {
		if pkg.SoftwareIcon.ObjectLocation == key {
			return key, nil
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
	if name == "" {
		return
	}
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

func (s *RepositoryService) catalogItems(effective []munkisoftware.EffectivePackage) []map[string]any {
	items := make([]map[string]any, 0, len(effective))
	seen := make(map[int64]bool, len(effective))
	for _, pkg := range effective {
		if seen[pkg.Package.ID] {
			continue
		}
		seen[pkg.Package.ID] = true
		item := packages.Pkginfo(pkg.Package, pkg.SoftwareIcon)
		if pkg.Package.InstallerObjectID != nil {
			if pkg.Package.InstallerObjectLocation != "" {
				item["installer_item_location"] = pkg.Package.InstallerObjectLocation
			}
		}
		if pkg.Package.UninstallMethod == packages.UninstallMethodUninstallPackage &&
			pkg.Package.UninstallerObjectID != nil &&
			pkg.Package.UninstallerObjectLocation != "" {
			item["uninstaller_item_location"] = pkg.Package.UninstallerObjectLocation
		}
		items = append(items, item)
	}
	return items
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func validResourceName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && !strings.ContainsAny(name, `/\`)
}

func validResourcePath(location string) bool {
	location = strings.TrimSpace(location)
	if location == "" || strings.HasPrefix(location, "/") || strings.Contains(location, `\`) {
		return false
	}
	for segment := range strings.SplitSeq(location, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func encodePlist(value any) ([]byte, error) {
	return plist.Marshal(value, plist.XMLFormat)
}

type renderedManifest struct {
	Catalogs          []string `plist:"catalogs"`
	DisplayName       string   `plist:"display_name"`
	ManagedInstalls   []string `plist:"managed_installs"`
	ManagedUninstalls []string `plist:"managed_uninstalls"`
	ManagedUpdates    []string `plist:"managed_updates"`
	OptionalInstalls  []string `plist:"optional_installs"`
	DefaultInstalls   []string `plist:"default_installs"`
	FeaturedItems     []string `plist:"featured_items"`
}
