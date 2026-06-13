package munki

import (
	"context"
	"errors"
	"slices"
	"strings"

	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
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

type artifactResolver interface {
	GetByLocation(context.Context, artifacts.ArtifactKind, string) (*artifacts.Artifact, error)
}

type artifactPresigner interface {
	PresignGet(context.Context, artifacts.Artifact) (string, error)
}

// ClientHost identifies the existing Woodstar host making a Munki request.
type ClientHost struct {
	ID          int64
	Serial      string
	DisplayName string
}

// RepositoryService renders the Munki client-facing repository surface.
type RepositoryService struct {
	hosts     hostResolver
	packages  packageResolver
	artifacts artifactResolver
	presigner artifactPresigner
}

// RepositoryServiceOption changes optional Munki repository behavior.
type RepositoryServiceOption func(*RepositoryService)

// WithArtifactStore lets the service resolve stable artifact locations.
func WithArtifactStore(artifacts artifactResolver) RepositoryServiceOption {
	return func(s *RepositoryService) {
		s.artifacts = artifacts
	}
}

// WithArtifactPresigner lets the service redirect stable artifact URLs to object storage.
func WithArtifactPresigner(presigner artifactPresigner) RepositoryServiceOption {
	return func(s *RepositoryService) {
		s.presigner = presigner
	}
}

// NewRepositoryService returns the default Munki repository renderer.
func NewRepositoryService(
	hosts hostResolver,
	packages packageResolver,
	options ...RepositoryServiceOption,
) *RepositoryService {
	s := &RepositoryService{hosts: hosts, packages: packages}
	for _, option := range options {
		option(s)
	}
	return s
}

// ResolveClient resolves the Munki request identity to an existing host.
func (s *RepositoryService) ResolveClient(ctx context.Context, serial string) (ClientHost, error) {
	if s.hosts == nil {
		return ClientHost{}, ErrNotFound
	}
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
	packages, err := s.effectivePackages(ctx, client.ID)
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
	for _, pkg := range packages {
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

// ArtifactRedirect returns a storage-backed URL for a stable Woodstar artifact URL.
func (s *RepositoryService) ArtifactRedirect(
	ctx context.Context,
	client ClientHost,
	kind artifacts.ArtifactKind,
	location string,
) (string, error) {
	if s.artifacts == nil {
		return "", ErrNotFound
	}
	location = strings.TrimSpace(location)
	if !artifacts.ValidArtifactKind(kind) || !validResourcePath(location) {
		return "", ErrNotFound
	}
	artifact, err := s.artifacts.GetByLocation(ctx, kind, location)
	if errors.Is(err, dbutil.ErrNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if kind == artifacts.ArtifactKindPackage {
		ok, err := s.clientCanFetchPackageArtifact(ctx, client.ID, *artifact)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", ErrNotFound
		}
	}
	if s.presigner == nil {
		return "", artifacts.ErrUnavailable
	}
	storageURL, err := s.presigner.PresignGet(ctx, *artifact)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(storageURL) == "" {
		return "", artifacts.ErrUnavailable
	}
	return storageURL, nil
}

func (s *RepositoryService) clientCanFetchPackageArtifact(
	ctx context.Context,
	hostID int64,
	artifact artifacts.Artifact,
) (bool, error) {
	packages, err := s.effectivePackages(ctx, hostID)
	if err != nil {
		return false, err
	}
	for _, pkg := range packages {
		if pkg.Package.InstallerArtifactID != nil &&
			*pkg.Package.InstallerArtifactID == artifact.ID &&
			pkg.Package.InstallerArtifactLocation == artifact.Location {
			return true, nil
		}
		if pkg.Package.UninstallerArtifactID != nil &&
			*pkg.Package.UninstallerArtifactID == artifact.ID &&
			pkg.Package.UninstallerArtifactLocation == artifact.Location {
			return true, nil
		}
	}
	return false, nil
}

func (s *RepositoryService) effectivePackages(
	ctx context.Context,
	hostID int64,
) ([]munkisoftware.EffectivePackage, error) {
	if s.packages == nil {
		return nil, nil
	}
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
		delete(item, "PackageCompleteURL")
		delete(item, "PackageURL")
		delete(item, "installer_item_location")
		if pkg.Package.InstallerArtifactID != nil {
			if pkg.Package.InstallerArtifactLocation != "" {
				item["installer_item_location"] = pkg.Package.InstallerArtifactLocation
			}
		}
		if pkg.Package.UninstallMethod == packages.UninstallMethodUninstallPackage &&
			pkg.Package.UninstallerArtifactID != nil &&
			pkg.Package.UninstallerArtifactLocation != "" {
			item["uninstaller_item_location"] = pkg.Package.UninstallerArtifactLocation
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
