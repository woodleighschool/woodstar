package munki

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
)

// ArtifactKind describes how Munki clients consume an artifact.
type ArtifactKind string

const (
	// ArtifactKindPackage is an installer package or disk image.
	ArtifactKindPackage ArtifactKind = "package"

	// ArtifactKindIcon is an icon referenced by rendered pkginfo.
	ArtifactKindIcon ArtifactKind = "icon"
)

var artifactKindValues = []ArtifactKind{
	ArtifactKindPackage,
	ArtifactKindIcon,
}

func (ArtifactKind) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(artifactKindValues...)
}

// DeploymentIntent describes how Munki should enforce or present a deployed package.
type DeploymentIntent string

const (
	// IntentEnsureInstalled puts the package in managed_installs.
	IntentEnsureInstalled DeploymentIntent = "ensure_installed"

	// IntentEnsureAbsent puts the package in managed_uninstalls.
	IntentEnsureAbsent DeploymentIntent = "ensure_absent"

	// IntentUpdateIfPresent puts the package in managed_updates.
	IntentUpdateIfPresent DeploymentIntent = "update_if_present"

	// IntentOptional puts the package in optional_installs.
	IntentOptional DeploymentIntent = "optional"

	// IntentFeatured puts the package in optional_installs and featured_items.
	IntentFeatured DeploymentIntent = "featured"
)

var deploymentIntentValues = []DeploymentIntent{
	IntentEnsureInstalled,
	IntentEnsureAbsent,
	IntentUpdateIfPresent,
	IntentOptional,
	IntentFeatured,
}

func (DeploymentIntent) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(deploymentIntentValues...)
}

// SoftwareTitleMutation is the input shape for creating or updating a Munki software title.
type SoftwareTitleMutation struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Developer   string `json:"developer,omitempty"`
}

// SoftwareTitle is Woodstar-managed metadata for a Munki software item.
type SoftwareTitle struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Developer   string    `json:"developer"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PackageMetadata holds editable Munki pkginfo fields that are not Woodstar
// software identity or deployment state.
type PackageMetadata struct {
	InstallerType          string   `json:"installer_type,omitempty"`
	UnattendedInstall      bool     `json:"unattended_install,omitempty"`
	UnattendedUninstall    bool     `json:"unattended_uninstall,omitempty"`
	Uninstallable          bool     `json:"uninstallable,omitempty"`
	UninstallMethod        string   `json:"uninstall_method,omitempty"`
	RestartAction          string   `json:"restart_action,omitempty"`
	MinimumMunkiVersion    string   `json:"minimum_munki_version,omitempty"`
	MinimumOSVersion       string   `json:"minimum_os_version,omitempty"`
	MaximumOSVersion       string   `json:"maximum_os_version,omitempty"`
	SupportedArchitectures []string `json:"supported_architectures,omitempty"`
	BlockingApplications   []string `json:"blocking_applications,omitempty"`
	Requires               []string `json:"requires,omitempty"`
	UpdateFor              []string `json:"update_for,omitempty"`
	OnDemand               bool     `json:"on_demand,omitempty"`
	Precache               bool     `json:"precache,omitempty"`
}

// PackageMutation is the editable shape for a Munki package version.
type PackageMutation struct {
	SoftwareID          int64           `json:"software_id"`
	Name                string          `json:"name"`
	Version             string          `json:"version"`
	DisplayName         string          `json:"display_name,omitempty"`
	Description         string          `json:"description,omitempty"`
	Category            string          `json:"category,omitempty"`
	Developer           string          `json:"developer,omitempty"`
	Metadata            PackageMetadata `json:"metadata"`
	InstallerArtifactID *int64          `json:"installer_artifact_id,omitempty"`
	Eligible            bool            `json:"eligible"`
}

// Package is one Munki pkginfo item available for deployment.
type Package struct {
	ID                        int64           `json:"id"`
	SoftwareID                int64           `json:"software_id"`
	SoftwareName              string          `json:"software_name"`
	SoftwareDisplayName       string          `json:"software_display_name"`
	Name                      string          `json:"name"`
	Version                   string          `json:"version"`
	DisplayName               string          `json:"display_name"`
	Description               string          `json:"description"`
	Category                  string          `json:"category"`
	Developer                 string          `json:"developer"`
	Metadata                  PackageMetadata `json:"metadata"`
	Pkginfo                   json.RawMessage `json:"pkginfo,omitempty"`
	InstallerArtifactID       *int64          `json:"installer_artifact_id,omitempty"`
	InstallerArtifactLocation string          `json:"installer_artifact_location,omitempty"`
	Eligible                  bool            `json:"eligible"`
	CreatedAt                 time.Time       `json:"created_at"`
	UpdatedAt                 time.Time       `json:"updated_at"`
}

// ArtifactMutation is the input shape for registering an existing Munki artifact.
type ArtifactMutation struct {
	Kind        ArtifactKind `json:"kind"`
	DisplayName string       `json:"display_name,omitempty"`
	Location    string       `json:"location"`
	ContentType string       `json:"content_type,omitempty"`
	SizeBytes   int64        `json:"size_bytes"`
	SHA256      string       `json:"sha256"`
	StorageKey  string       `json:"storage_key"`
}

// Artifact references one object stored in Munki's artifact backend.
type Artifact struct {
	ID          int64        `json:"id"`
	Kind        ArtifactKind `json:"kind"`
	DisplayName string       `json:"display_name"`
	Location    string       `json:"location"`
	ContentType string       `json:"content_type"`
	SizeBytes   int64        `json:"size_bytes"`
	SHA256      string       `json:"sha256"`
	StorageKey  string       `json:"storage_key"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// DeploymentMutation is the input shape for applying a package to a concrete Munki scope.
type DeploymentMutation struct {
	PackageID       int64            `json:"package_id"`
	Intent          DeploymentIntent `json:"intent"`
	AllHosts        bool             `json:"all_hosts"`
	IncludeLabelIDs []int64          `json:"include_label_ids,omitempty"`
	ExcludeLabelIDs []int64          `json:"exclude_label_ids,omitempty"`
	IncludeHostIDs  []int64          `json:"include_host_ids,omitempty"`
	ExcludeHostIDs  []int64          `json:"exclude_host_ids,omitempty"`
}

// Deployment links one Munki package, one intent, and concrete include/exclude scope.
type Deployment struct {
	ID                  int64            `json:"id"`
	PackageID           int64            `json:"package_id"`
	PackageName         string           `json:"package_name"`
	PackageVersion      string           `json:"package_version"`
	SoftwareID          int64            `json:"software_id"`
	SoftwareDisplayName string           `json:"software_display_name"`
	Intent              DeploymentIntent `json:"intent"`
	Position            int32            `json:"position"`
	AllHosts            bool             `json:"all_hosts"`
	IncludeLabelIDs     []int64          `json:"include_label_ids"`
	ExcludeLabelIDs     []int64          `json:"exclude_label_ids"`
	IncludeHostIDs      []int64          `json:"include_host_ids"`
	ExcludeHostIDs      []int64          `json:"exclude_host_ids"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

// EffectivePackage is a host-resolved Munki package ready for manifest/catalog rendering.
type EffectivePackage struct {
	DeploymentID int64
	Intent       DeploymentIntent
	Position     int32
	Package      Package
	scopeRank    int
}

type SoftwareTitleDetail struct {
	SoftwareTitle
	Packages    []Package    `json:"packages"`
	Deployments []Deployment `json:"deployments"`
}

type PackageListParams struct {
	dbutil.ListParams
	SoftwareID int64
}

type DeploymentListParams struct {
	dbutil.ListParams
	SoftwareID int64
}

// HostStatusObservation is Munki state observed for an existing host.
type HostStatusObservation struct {
	HostID          int64
	Version         string
	ManifestName    string
	Success         *bool
	Errors          []string
	Warnings        []string
	ProblemInstalls []string
	RunStartedAt    string
	RunEndedAt      string
}

// HostItem is one Munki-managed item observed on a host.
type HostItem struct {
	HostID           int64     `json:"-"`
	Name             string    `json:"name"`
	Installed        bool      `json:"installed"`
	InstalledVersion string    `json:"installed_version"`
	RunEndedAt       string    `json:"run_ended_at,omitempty"`
	LastSeenAt       time.Time `json:"last_seen_at"`
}

// HostState is the Munki sub-object attached to host detail responses.
type HostState struct {
	Version         string     `json:"version"`
	ManifestName    string     `json:"manifest_name"`
	Success         *bool      `json:"success,omitempty"`
	Errors          []string   `json:"errors"`
	Warnings        []string   `json:"warnings"`
	ProblemInstalls []string   `json:"problem_installs"`
	RunStartedAt    string     `json:"run_started_at,omitempty"`
	RunEndedAt      string     `json:"run_ended_at,omitempty"`
	LastSeenAt      time.Time  `json:"last_seen_at"`
	Items           []HostItem `json:"items"`
}

func (m SoftwareTitleMutation) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	return nil
}

func (m PackageMutation) Validate() error {
	if m.SoftwareID <= 0 {
		return fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("%w: version is required", dbutil.ErrInvalidInput)
	}
	if m.InstallerArtifactID != nil && *m.InstallerArtifactID <= 0 {
		return fmt.Errorf("%w: installer_artifact_id must be positive", dbutil.ErrInvalidInput)
	}
	return m.Metadata.Validate()
}

func (m PackageMetadata) Validate() error {
	for _, arch := range m.SupportedArchitectures {
		switch strings.TrimSpace(arch) {
		case "arm64", "x86_64":
		default:
			return fmt.Errorf(
				"%w: supported_architectures contains unsupported architecture %q",
				dbutil.ErrInvalidInput,
				arch,
			)
		}
	}
	return nil
}

func (m ArtifactMutation) Validate() error {
	if !validArtifactKind(m.Kind) {
		return fmt.Errorf("%w: unsupported artifact kind %q", dbutil.ErrInvalidInput, m.Kind)
	}
	if !validArtifactLocation(m.Location) {
		return fmt.Errorf("%w: location is required and must be a relative Munki path", dbutil.ErrInvalidInput)
	}
	if m.SizeBytes < 0 {
		return fmt.Errorf("%w: size_bytes must not be negative", dbutil.ErrInvalidInput)
	}
	if !validSHA256(m.SHA256) {
		return fmt.Errorf("%w: sha256 must be 64 lowercase hex characters", dbutil.ErrInvalidInput)
	}
	if strings.TrimSpace(m.StorageKey) == "" || strings.HasPrefix(strings.TrimSpace(m.StorageKey), "/") {
		return fmt.Errorf("%w: storage_key is required and must be relative", dbutil.ErrInvalidInput)
	}
	return nil
}

func (m DeploymentMutation) Validate() error {
	if m.PackageID <= 0 {
		return fmt.Errorf("%w: package_id is required", dbutil.ErrInvalidInput)
	}
	if !validDeploymentIntent(m.Intent) {
		return fmt.Errorf("%w: unsupported deployment intent %q", dbutil.ErrInvalidInput, m.Intent)
	}
	if !m.AllHosts && len(m.IncludeLabelIDs) == 0 && len(m.IncludeHostIDs) == 0 {
		return fmt.Errorf("%w: deployment scope is required", dbutil.ErrInvalidInput)
	}
	return nil
}

func validDeploymentIntent(intent DeploymentIntent) bool {
	return slices.Contains(deploymentIntentValues, intent)
}

func validArtifactKind(kind ArtifactKind) bool {
	return slices.Contains(artifactKindValues, kind)
}

func validArtifactLocation(location string) bool {
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

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func packagePkginfo(pkg Package) (json.RawMessage, error) {
	item := map[string]any{
		"name":    pkg.Name,
		"version": pkg.Version,
	}
	addPkginfoString(item, "display_name", pkg.DisplayName)
	addPkginfoString(item, "description", pkg.Description)
	addPkginfoString(item, "category", pkg.Category)
	addPkginfoString(item, "developer", pkg.Developer)
	addPkginfoString(item, "installer_type", pkg.Metadata.InstallerType)
	addPkginfoString(item, "uninstall_method", pkg.Metadata.UninstallMethod)
	addPkginfoString(item, "RestartAction", pkg.Metadata.RestartAction)
	addPkginfoString(item, "minimum_munki_version", pkg.Metadata.MinimumMunkiVersion)
	addPkginfoString(item, "minimum_os_version", pkg.Metadata.MinimumOSVersion)
	addPkginfoString(item, "maximum_os_version", pkg.Metadata.MaximumOSVersion)
	addPkginfoStrings(item, "supported_architectures", pkg.Metadata.SupportedArchitectures)
	addPkginfoStrings(item, "blocking_applications", pkg.Metadata.BlockingApplications)
	addPkginfoStrings(item, "requires", pkg.Metadata.Requires)
	addPkginfoStrings(item, "update_for", pkg.Metadata.UpdateFor)
	addPkginfoBool(item, "unattended_install", pkg.Metadata.UnattendedInstall)
	addPkginfoBool(item, "unattended_uninstall", pkg.Metadata.UnattendedUninstall)
	addPkginfoBool(item, "uninstallable", pkg.Metadata.Uninstallable)
	addPkginfoBool(item, "OnDemand", pkg.Metadata.OnDemand)
	addPkginfoBool(item, "precache", pkg.Metadata.Precache)

	raw, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func addPkginfoString(item map[string]any, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		item[key] = value
	}
}

func addPkginfoStrings(item map[string]any, key string, values []string) {
	values = cleanStringList(values)
	if len(values) > 0 {
		item[key] = values
	}
}

func addPkginfoBool(item map[string]any, key string, value bool) {
	if value {
		item[key] = true
	}
}

func cleanStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
