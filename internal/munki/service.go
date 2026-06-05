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
	"github.com/woodleighschool/woodstar/internal/munki/assignments"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkistorage "github.com/woodleighschool/woodstar/internal/munki/storage"
)

// ErrNotFound reports that a requested Munki repository object does not exist.
var ErrNotFound = errors.New("munki resource not found")

type hostResolver interface {
	GetByHardwareSerial(context.Context, string) (*hosts.Host, error)
}

type packageResolver interface {
	EffectivePackagesForHost(context.Context, int64) ([]assignments.EffectivePackage, error)
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

// Service renders the Munki client-facing repository surface.
type Service struct {
	hosts     hostResolver
	packages  packageResolver
	artifacts artifactResolver
	presigner artifactPresigner
}

// ServiceOption changes optional Munki repository behavior.
type ServiceOption func(*Service)

// WithArtifactStore lets the service resolve stable artifact locations.
func WithArtifactStore(artifacts artifactResolver) ServiceOption {
	return func(s *Service) {
		s.artifacts = artifacts
	}
}

// WithArtifactPresigner lets the service redirect stable artifact URLs to object storage.
func WithArtifactPresigner(presigner artifactPresigner) ServiceOption {
	return func(s *Service) {
		s.presigner = presigner
	}
}

// NewService returns the default Munki repository renderer.
func NewService(hosts hostResolver, packages packageResolver, options ...ServiceOption) *Service {
	s := &Service{hosts: hosts, packages: packages}
	if artifacts, ok := packages.(artifactResolver); ok {
		s.artifacts = artifacts
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// ResolveClient resolves the Munki request identity to an existing host.
func (s *Service) ResolveClient(ctx context.Context, serial string) (ClientHost, error) {
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
func (s *Service) Manifest(ctx context.Context, client ClientHost, name string) ([]byte, error) {
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
func (s *Service) Catalog(ctx context.Context, client ClientHost, name string) ([]byte, error) {
	if name != "production" || !validResourceName(name) {
		return nil, ErrNotFound
	}
	packages, err := s.effectivePackages(ctx, client.ID)
	if err != nil {
		return nil, err
	}
	items, err := s.catalogItems(packages)
	if err != nil {
		return nil, err
	}
	return encodePlist(items)
}

// ArtifactRedirect returns a storage-backed URL for a stable Woodstar artifact URL.
func (s *Service) ArtifactRedirect(
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
		return "", munkistorage.ErrUnavailable
	}
	storageURL, err := s.presigner.PresignGet(ctx, *artifact)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(storageURL) == "" {
		return "", munkistorage.ErrUnavailable
	}
	return storageURL, nil
}

func (s *Service) clientCanFetchPackageArtifact(
	ctx context.Context,
	hostID int64,
	artifact artifacts.Artifact,
) (bool, error) {
	packages, err := s.effectivePackages(ctx, hostID)
	if err != nil {
		return false, err
	}
	for _, pkg := range packages {
		if pkg.Package.InstallerArtifactID == nil || *pkg.Package.InstallerArtifactID != artifact.ID {
			continue
		}
		if pkg.Package.InstallerArtifactLocation == artifact.Location {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) effectivePackages(
	ctx context.Context,
	hostID int64,
) ([]assignments.EffectivePackage, error) {
	if s.packages == nil {
		return nil, nil
	}
	return s.packages.EffectivePackagesForHost(ctx, hostID)
}

func addManifestPackage(manifest *renderedManifest, pkg assignments.EffectivePackage) {
	name := manifestItemName(pkg)
	if name == "" {
		return
	}
	switch pkg.Action {
	case assignments.AssignmentActionInstall:
		manifest.ManagedInstalls = appendUnique(manifest.ManagedInstalls, name)
	case assignments.AssignmentActionRemove:
		manifest.ManagedUninstalls = appendUnique(manifest.ManagedUninstalls, name)
	case assignments.AssignmentActionUpdateIfPresent:
		manifest.ManagedUpdates = appendUnique(manifest.ManagedUpdates, name)
	case assignments.AssignmentActionNone:
	}
	if pkg.OptionalInstall {
		manifest.OptionalInstalls = appendUnique(manifest.OptionalInstalls, name)
	}
	if pkg.FeaturedItem {
		manifest.FeaturedItems = appendUnique(manifest.FeaturedItems, name)
	}
}

func manifestItemName(pkg assignments.EffectivePackage) string {
	name := strings.TrimSpace(pkg.Package.Name)
	if name == "" {
		return ""
	}
	if pkg.PackageSelection != assignments.PackageSelectionSpecific {
		return name
	}
	version := strings.TrimSpace(pkg.Package.Version)
	if version == "" {
		return name
	}
	return name + "--" + version
}

func (s *Service) catalogItems(effective []assignments.EffectivePackage) ([]map[string]any, error) {
	items := make([]map[string]any, 0, len(effective))
	seen := make(map[int64]bool, len(effective))
	for _, pkg := range effective {
		if seen[pkg.Package.ID] {
			continue
		}
		seen[pkg.Package.ID] = true
		item := packages.Pkginfo(pkg.Package)
		delete(item, "PackageCompleteURL")
		delete(item, "PackageURL")
		delete(item, "installer_item_location")
		if pkg.Package.InstallerArtifactID != nil {
			if pkg.Package.InstallerArtifactLocation != "" {
				item["installer_item_location"] = pkg.Package.InstallerArtifactLocation
			}
		}
		items = append(items, item)
	}
	return items, nil
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
