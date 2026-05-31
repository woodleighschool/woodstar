package munki

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// ErrNotFound reports that a requested Munki repository object does not exist.
var ErrNotFound = errors.New("munki resource not found")

// ErrStorageUnavailable reports that an artifact exists but has no usable storage backend.
var ErrStorageUnavailable = errors.New("munki artifact storage unavailable")

type hostResolver interface {
	GetByHardwareSerial(context.Context, string) (*hosts.Host, error)
}

type releaseResolver interface {
	EffectiveReleasesForHost(context.Context, int64) ([]EffectiveRelease, error)
}

type artifactResolver interface {
	GetArtifactByLocation(context.Context, ArtifactKind, string) (*Artifact, error)
}

type artifactPresigner interface {
	PresignGet(context.Context, Artifact) (string, error)
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
	releases  releaseResolver
	artifacts artifactResolver
	presigner artifactPresigner
	publicURL string
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

// WithPublicURL sets the absolute base URL used in rendered Munki metadata.
func WithPublicURL(publicURL string) ServiceOption {
	return func(s *Service) {
		s.publicURL = strings.TrimRight(strings.TrimSpace(publicURL), "/")
	}
}

// NewService returns the default Munki repository renderer.
func NewService(hosts hostResolver, releases releaseResolver, options ...ServiceOption) *Service {
	s := &Service{hosts: hosts, releases: releases}
	if artifacts, ok := releases.(artifactResolver); ok {
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
	releases, err := s.effectiveReleases(ctx, client.ID)
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
		FeaturedItems:     []string{},
	}
	for _, release := range releases {
		addManifestRelease(&manifest, release)
	}
	return encodePlist(manifest)
}

// Catalog returns a Munki catalog plist for name.
func (s *Service) Catalog(ctx context.Context, client ClientHost, name string) ([]byte, error) {
	if name != "production" || !validResourceName(name) {
		return nil, ErrNotFound
	}
	releases, err := s.effectiveReleases(ctx, client.ID)
	if err != nil {
		return nil, err
	}
	items, err := s.catalogItems(releases)
	if err != nil {
		return nil, err
	}
	return encodePlist(items)
}

// ArtifactRedirect returns a storage-backed URL for a stable Woodstar artifact URL.
func (s *Service) ArtifactRedirect(
	ctx context.Context,
	_ ClientHost,
	kind ArtifactKind,
	location string,
) (string, error) {
	if s.artifacts == nil {
		return "", ErrNotFound
	}
	location = strings.TrimSpace(location)
	if !validArtifactKind(kind) || !validResourcePath(location) {
		return "", ErrNotFound
	}
	artifact, err := s.artifacts.GetArtifactByLocation(ctx, kind, location)
	if errors.Is(err, dbutil.ErrNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if s.presigner == nil {
		return "", ErrStorageUnavailable
	}
	storageURL, err := s.presigner.PresignGet(ctx, *artifact)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(storageURL) == "" {
		return "", ErrStorageUnavailable
	}
	return storageURL, nil
}

func (s *Service) effectiveReleases(ctx context.Context, hostID int64) ([]EffectiveRelease, error) {
	if s.releases == nil {
		return nil, nil
	}
	releases, err := s.releases.EffectiveReleasesForHost(ctx, hostID)
	if err != nil {
		return nil, err
	}
	return resolveEffectiveReleases(releases), nil
}

func addManifestRelease(manifest *renderedManifest, release EffectiveRelease) {
	name := strings.TrimSpace(release.Release.Name)
	if name == "" {
		return
	}
	switch release.Intent {
	case IntentEnsureInstalled:
		manifest.ManagedInstalls = appendUnique(manifest.ManagedInstalls, name)
	case IntentEnsureAbsent:
		manifest.ManagedUninstalls = appendUnique(manifest.ManagedUninstalls, name)
	case IntentUpdateIfPresent:
		manifest.ManagedUpdates = appendUnique(manifest.ManagedUpdates, name)
	case IntentOptional:
		manifest.OptionalInstalls = appendUnique(manifest.OptionalInstalls, name)
	case IntentFeatured:
		manifest.OptionalInstalls = appendUnique(manifest.OptionalInstalls, name)
		manifest.FeaturedItems = appendUnique(manifest.FeaturedItems, name)
	}
}

func resolveEffectiveReleases(releases []EffectiveRelease) []EffectiveRelease {
	resolved := make([]EffectiveRelease, 0, len(releases))
	positions := make(map[string]int, len(releases))
	for _, release := range releases {
		name := strings.TrimSpace(release.Release.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		position, exists := positions[key]
		if !exists {
			positions[key] = len(resolved)
			resolved = append(resolved, release)
			continue
		}
		if betterEffectiveRelease(release, resolved[position]) {
			resolved[position] = release
		}
	}
	return resolved
}

func betterEffectiveRelease(candidate, current EffectiveRelease) bool {
	if candidate.Intent != current.Intent {
		return assignmentIntentRank(candidate.Intent) > assignmentIntentRank(current.Intent)
	}
	if candidate.scopeRank != current.scopeRank {
		return candidate.scopeRank > current.scopeRank
	}
	if candidate.Release.ID != current.Release.ID {
		return candidate.Release.ID > current.Release.ID
	}
	return candidate.AssignmentID > current.AssignmentID
}

func assignmentIntentRank(intent AssignmentIntent) int {
	switch intent {
	case IntentEnsureAbsent:
		return 50
	case IntentEnsureInstalled:
		return 40
	case IntentUpdateIfPresent:
		return 30
	case IntentFeatured:
		return 20
	case IntentOptional:
		return 10
	default:
		return 0
	}
}

func (s *Service) catalogItems(releases []EffectiveRelease) ([]map[string]any, error) {
	items := make([]map[string]any, 0, len(releases))
	seen := make(map[int64]bool, len(releases))
	for _, release := range releases {
		if seen[release.Release.ID] {
			continue
		}
		seen[release.Release.ID] = true
		var item map[string]any
		if err := json.Unmarshal(release.Release.Pkginfo, &item); err != nil {
			return nil, fmt.Errorf("render munki pkginfo %d: %w", release.Release.ID, err)
		}
		delete(item, "PackageCompleteURL")
		delete(item, "PackageURL")
		if release.Release.InstallerArtifactID != nil {
			if release.Release.InstallerArtifactLocation != "" {
				item["installer_item_location"] = release.Release.InstallerArtifactLocation
				item["PackageCompleteURL"] = s.artifactURL(
					ArtifactKindPackage,
					release.Release.InstallerArtifactLocation,
				)
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

func (s *Service) artifactURL(kind ArtifactKind, location string) string {
	path := artifactPath(kind, location)
	if s.publicURL == "" {
		return path
	}
	return s.publicURL + path
}

func artifactPath(kind ArtifactKind, location string) string {
	prefix := "/munki/pkgs/"
	if kind == ArtifactKindIcon {
		prefix = "/munki/icons/"
	}
	return prefix + escapePath(location)
}

func escapePath(location string) string {
	segments := strings.Split(location, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
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
	FeaturedItems     []string `plist:"featured_items"`
}
