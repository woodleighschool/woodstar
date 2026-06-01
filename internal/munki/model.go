package munki

import (
	"encoding/json"
	"errors"
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

// InstallerType describes the package installer mode Woodstar exposes in
// normal authoring flows. InstallerTypePkg is Woodstar's default package mode
// and is omitted from rendered Munki pkginfo.
type InstallerType string

const (
	InstallerTypePkg                 InstallerType = "pkg"
	InstallerTypeNoPkg               InstallerType = "nopkg"
	InstallerTypeCopyFromDMG         InstallerType = "copy_from_dmg"
	InstallerTypeAppDMG              InstallerType = "appdmg"
	InstallerTypeProfile             InstallerType = "profile"
	InstallerTypeAppleUpdateMetadata InstallerType = "apple_update_metadata"
	InstallerTypeStartOSInstall      InstallerType = "startosinstall"
	InstallerTypeStageOSInstaller    InstallerType = "stage_os_installer"
	InstallerTypeAdobeCCPInstaller   InstallerType = "AdobeCCPInstaller"
	InstallerTypeAdobeCS5AAMEE       InstallerType = "AdobeCS5AAMEEPackage"
	InstallerTypeAdobeCS5Installer   InstallerType = "AdobeCS5Installer"
	InstallerTypeAdobeCS5Patch       InstallerType = "AdobeCS5PatchInstaller"
	InstallerTypeAdobeUberInstaller  InstallerType = "AdobeUberInstaller"
	InstallerTypeAdobeSetup          InstallerType = "AdobeSetup"
	InstallerTypeAdobeAcrobatUpdater InstallerType = "AdobeAcrobatUpdater"
)

var installerTypeValues = []InstallerType{
	InstallerTypePkg,
	InstallerTypeNoPkg,
	InstallerTypeCopyFromDMG,
	InstallerTypeAppDMG,
	InstallerTypeProfile,
	InstallerTypeAppleUpdateMetadata,
	InstallerTypeStartOSInstall,
	InstallerTypeStageOSInstaller,
	InstallerTypeAdobeCCPInstaller,
	InstallerTypeAdobeCS5AAMEE,
	InstallerTypeAdobeCS5Installer,
	InstallerTypeAdobeCS5Patch,
	InstallerTypeAdobeUberInstaller,
	InstallerTypeAdobeSetup,
	InstallerTypeAdobeAcrobatUpdater,
}

func (InstallerType) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(installerTypeValues...)
}

// RestartAction describes Munki's RestartAction values.
type RestartAction string

const (
	RestartActionNone             RestartAction = "None"
	RestartActionRequireLogout    RestartAction = "RequireLogout"
	RestartActionRecommendRestart RestartAction = "RecommendRestart"
	RestartActionRequireRestart   RestartAction = "RequireRestart"
	RestartActionRequireShutdown  RestartAction = "RequireShutdown"
)

var restartActionValues = []RestartAction{
	RestartActionNone,
	RestartActionRequireLogout,
	RestartActionRecommendRestart,
	RestartActionRequireRestart,
	RestartActionRequireShutdown,
}

func (RestartAction) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(restartActionValues...)
}

// DeploymentAction describes the automatic Munki manifest section for an assignment.
type DeploymentAction string

const (
	DeploymentActionInstall         DeploymentAction = "install"
	DeploymentActionRemove          DeploymentAction = "remove"
	DeploymentActionUpdateIfPresent DeploymentAction = "update_if_present"
	DeploymentActionNone            DeploymentAction = "none"
)

var deploymentActionValues = []DeploymentAction{
	DeploymentActionInstall,
	DeploymentActionRemove,
	DeploymentActionUpdateIfPresent,
	DeploymentActionNone,
}

func (DeploymentAction) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(deploymentActionValues...)
}

// SelfServiceMode describes how Munki should present an assignment in Managed Software Center.
type SelfServiceMode string

const (
	SelfServiceHidden    SelfServiceMode = "hidden"
	SelfServiceAvailable SelfServiceMode = "available"
	SelfServiceFeatured  SelfServiceMode = "featured"
	SelfServiceDefault   SelfServiceMode = "default"
)

var selfServiceModeValues = []SelfServiceMode{
	SelfServiceHidden,
	SelfServiceAvailable,
	SelfServiceFeatured,
	SelfServiceDefault,
}

func (SelfServiceMode) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(selfServiceModeValues...)
}

// PackageSelection describes whether an assignment follows latest compatible pkginfos or pins one pkginfo.
type PackageSelection string

const (
	PackageSelectionLatestEligible PackageSelection = "latest_eligible"
	PackageSelectionSpecific       PackageSelection = "specific_package"
)

var packageSelectionValues = []PackageSelection{
	PackageSelectionLatestEligible,
	PackageSelectionSpecific,
}

func (PackageSelection) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(packageSelectionValues...)
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

// PackageMutation is the editable shape for a Munki package version.
type PackageMutation struct {
	SoftwareID             int64           `json:"software_id"`
	Name                   string          `json:"name"`
	Version                string          `json:"version"`
	DisplayName            string          `json:"display_name,omitempty"`
	Description            string          `json:"description,omitempty"`
	Category               string          `json:"category,omitempty"`
	Developer              string          `json:"developer,omitempty"`
	InstallerType          InstallerType   `json:"installer_type,omitempty"`
	UnattendedInstall      bool            `json:"unattended_install,omitempty"`
	UnattendedUninstall    bool            `json:"unattended_uninstall,omitempty"`
	Uninstallable          bool            `json:"uninstallable,omitempty"`
	UninstallMethod        string          `json:"uninstall_method,omitempty"`
	RestartAction          RestartAction   `json:"restart_action,omitempty"`
	MinimumMunkiVersion    string          `json:"minimum_munki_version,omitempty"`
	MinimumOSVersion       string          `json:"minimum_os_version,omitempty"`
	MaximumOSVersion       string          `json:"maximum_os_version,omitempty"`
	SupportedArchitectures []string        `json:"supported_architectures,omitempty"`
	BlockingApplications   []string        `json:"blocking_applications,omitempty"`
	Requires               []string        `json:"requires,omitempty"`
	UpdateFor              []string        `json:"update_for,omitempty"`
	OnDemand               bool            `json:"on_demand,omitempty"`
	Precache               bool            `json:"precache,omitempty"`
	IconName               string          `json:"icon_name,omitempty"`
	IconHash               string          `json:"icon_hash,omitempty"`
	ExtraPkginfo           json.RawMessage `json:"extra_pkginfo,omitempty"`
	InstallerArtifactID    *int64          `json:"installer_artifact_id,omitempty"`
	IconArtifactID         *int64          `json:"icon_artifact_id,omitempty"`
	Eligible               bool            `json:"eligible"`
}

// PackageImportMutation imports one existing Munki pkginfo item as a Woodstar
// package row.
type PackageImportMutation struct {
	SoftwareID          int64           `json:"software_id,omitempty"`
	Pkginfo             json.RawMessage `json:"pkginfo"`
	InstallerArtifactID *int64          `json:"installer_artifact_id,omitempty"`
	IconArtifactID      *int64          `json:"icon_artifact_id,omitempty"`
	Eligible            *bool           `json:"eligible,omitempty"`
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
	InstallerType             InstallerType   `json:"installer_type"`
	UnattendedInstall         bool            `json:"unattended_install"`
	UnattendedUninstall       bool            `json:"unattended_uninstall"`
	Uninstallable             bool            `json:"uninstallable"`
	UninstallMethod           string          `json:"uninstall_method"`
	RestartAction             RestartAction   `json:"restart_action,omitempty"`
	MinimumMunkiVersion       string          `json:"minimum_munki_version"`
	MinimumOSVersion          string          `json:"minimum_os_version"`
	MaximumOSVersion          string          `json:"maximum_os_version"`
	SupportedArchitectures    []string        `json:"supported_architectures"`
	BlockingApplications      []string        `json:"blocking_applications"`
	Requires                  []string        `json:"requires"`
	UpdateFor                 []string        `json:"update_for"`
	OnDemand                  bool            `json:"on_demand"`
	Precache                  bool            `json:"precache"`
	IconName                  string          `json:"icon_name"`
	IconHash                  string          `json:"icon_hash"`
	ExtraPkginfo              json.RawMessage `json:"extra_pkginfo,omitempty"`
	Pkginfo                   json.RawMessage `json:"pkginfo,omitempty"`
	InstallerArtifactID       *int64          `json:"installer_artifact_id,omitempty"`
	InstallerArtifactLocation string          `json:"installer_artifact_location,omitempty"`
	IconArtifactID            *int64          `json:"icon_artifact_id,omitempty"`
	IconArtifactLocation      string          `json:"icon_artifact_location,omitempty"`
	IconURL                   string          `json:"icon_url,omitempty"`
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

// ArtifactUploadURL is a temporary object-storage upload target.
type ArtifactUploadURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ArtifactObject is the storage-side view of an uploaded artifact.
type ArtifactObject struct {
	ContentType string
	SizeBytes   int64
	SHA256      string
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

// DeploymentMutation is the input shape for applying software to a concrete Munki scope.
type DeploymentMutation struct {
	SoftwareID       int64            `json:"software_id"`
	Action           DeploymentAction `json:"action"`
	SelfService      SelfServiceMode  `json:"self_service"`
	PackageSelection PackageSelection `json:"package_selection"`
	PinnedPackageID  *int64           `json:"pinned_package_id,omitempty"`
	AllHosts         bool             `json:"all_hosts"`
	IncludeLabelIDs  []int64          `json:"include_label_ids,omitempty"`
	ExcludeLabelIDs  []int64          `json:"exclude_label_ids,omitempty"`
	IncludeHostIDs   []int64          `json:"include_host_ids,omitempty"`
	ExcludeHostIDs   []int64          `json:"exclude_host_ids,omitempty"`
}

// Deployment links one Munki software title, assignment behavior, and concrete include/exclude scope.
type Deployment struct {
	ID                   int64            `json:"id"`
	SoftwareID           int64            `json:"software_id"`
	SoftwareDisplayName  string           `json:"software_display_name"`
	Action               DeploymentAction `json:"action"`
	SelfService          SelfServiceMode  `json:"self_service"`
	PackageSelection     PackageSelection `json:"package_selection"`
	PinnedPackageID      *int64           `json:"pinned_package_id,omitempty"`
	PinnedPackageName    string           `json:"pinned_package_name,omitempty"`
	PinnedPackageVersion string           `json:"pinned_package_version,omitempty"`
	Position             int32            `json:"position"`
	AllHosts             bool             `json:"all_hosts"`
	IncludeLabelIDs      []int64          `json:"include_label_ids"`
	ExcludeLabelIDs      []int64          `json:"exclude_label_ids"`
	IncludeHostIDs       []int64          `json:"include_host_ids"`
	ExcludeHostIDs       []int64          `json:"exclude_host_ids"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

// EffectivePackage is a host-resolved Munki package ready for manifest/catalog rendering.
type EffectivePackage struct {
	DeploymentID     int64
	SoftwareID       int64
	Action           DeploymentAction
	SelfService      SelfServiceMode
	PackageSelection PackageSelection
	PinnedPackageID  *int64
	Position         int32
	Package          Package
	scopeRank        int
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
	if m.IconArtifactID != nil && *m.IconArtifactID <= 0 {
		return fmt.Errorf("%w: icon_artifact_id must be positive", dbutil.ErrInvalidInput)
	}
	if !validInstallerType(m.InstallerType) {
		return fmt.Errorf("%w: unsupported installer_type %q", dbutil.ErrInvalidInput, m.InstallerType)
	}
	if !validRestartAction(m.RestartAction) {
		return fmt.Errorf("%w: unsupported restart_action %q", dbutil.ErrInvalidInput, m.RestartAction)
	}
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
	if len(m.ExtraPkginfo) > 0 && !json.Valid(m.ExtraPkginfo) {
		return fmt.Errorf("%w: extra_pkginfo must be valid JSON", dbutil.ErrInvalidInput)
	}
	if len(m.ExtraPkginfo) > 0 {
		var extra map[string]any
		if err := json.Unmarshal(m.ExtraPkginfo, &extra); err != nil {
			return fmt.Errorf("%w: extra_pkginfo must be a JSON object", dbutil.ErrInvalidInput)
		}
		if extra == nil {
			return fmt.Errorf("%w: extra_pkginfo must be a JSON object", dbutil.ErrInvalidInput)
		}
	}
	return nil
}

func (m PackageImportMutation) Validate() error {
	if m.SoftwareID < 0 {
		return fmt.Errorf("%w: software_id must not be negative", dbutil.ErrInvalidInput)
	}
	if m.InstallerArtifactID != nil && *m.InstallerArtifactID <= 0 {
		return fmt.Errorf("%w: installer_artifact_id must be positive", dbutil.ErrInvalidInput)
	}
	if m.IconArtifactID != nil && *m.IconArtifactID <= 0 {
		return fmt.Errorf("%w: icon_artifact_id must be positive", dbutil.ErrInvalidInput)
	}
	if len(m.Pkginfo) == 0 || !json.Valid(m.Pkginfo) {
		return fmt.Errorf("%w: pkginfo must be a JSON object", dbutil.ErrInvalidInput)
	}
	var item map[string]any
	if err := json.Unmarshal(m.Pkginfo, &item); err != nil {
		return fmt.Errorf("%w: pkginfo must be a JSON object", dbutil.ErrInvalidInput)
	}
	if item == nil {
		return fmt.Errorf("%w: pkginfo must be a JSON object", dbutil.ErrInvalidInput)
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
	if m.SoftwareID <= 0 {
		return fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	if !validDeploymentAction(m.Action) {
		return fmt.Errorf("%w: unsupported deployment action %q", dbutil.ErrInvalidInput, m.Action)
	}
	if !validSelfServiceMode(m.SelfService) {
		return fmt.Errorf("%w: unsupported self_service %q", dbutil.ErrInvalidInput, m.SelfService)
	}
	if !validPackageSelection(m.PackageSelection) {
		return fmt.Errorf(
			"%w: unsupported package_selection %q",
			dbutil.ErrInvalidInput,
			m.PackageSelection,
		)
	}
	switch m.PackageSelection {
	case PackageSelectionLatestEligible:
		if m.PinnedPackageID != nil {
			return fmt.Errorf(
				"%w: pinned_package_id must be empty for latest_eligible selection",
				dbutil.ErrInvalidInput,
			)
		}
	case PackageSelectionSpecific:
		if m.PinnedPackageID == nil || *m.PinnedPackageID <= 0 {
			return fmt.Errorf("%w: pinned_package_id is required", dbutil.ErrInvalidInput)
		}
	}
	if m.Action == DeploymentActionRemove && m.SelfService != SelfServiceHidden {
		return fmt.Errorf("%w: remove assignments cannot be shown in Self Service", dbutil.ErrInvalidInput)
	}
	if !m.AllHosts && len(m.IncludeLabelIDs) == 0 && len(m.IncludeHostIDs) == 0 {
		return fmt.Errorf("%w: deployment scope is required", dbutil.ErrInvalidInput)
	}
	return nil
}

func validDeploymentAction(action DeploymentAction) bool {
	return slices.Contains(deploymentActionValues, action)
}

func validSelfServiceMode(mode SelfServiceMode) bool {
	return slices.Contains(selfServiceModeValues, mode)
}

func validPackageSelection(selection PackageSelection) bool {
	return slices.Contains(packageSelectionValues, selection)
}

func validInstallerType(installerType InstallerType) bool {
	return installerType == "" || slices.Contains(installerTypeValues, installerType)
}

func validRestartAction(restartAction RestartAction) bool {
	return restartAction == "" || slices.Contains(restartActionValues, restartAction)
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
	item, err := packageExtraPkginfo(pkg.ExtraPkginfo)
	if err != nil {
		return nil, err
	}
	item["name"] = pkg.Name
	item["version"] = pkg.Version

	addPkginfoString(item, "display_name", pkg.DisplayName)
	addPkginfoString(item, "description", pkg.Description)
	addPkginfoString(item, "category", pkg.Category)
	addPkginfoString(item, "developer", pkg.Developer)
	if pkg.InstallerType != "" && pkg.InstallerType != InstallerTypePkg {
		item["installer_type"] = pkg.InstallerType
	} else {
		delete(item, "installer_type")
	}
	addPkginfoString(item, "uninstall_method", pkg.UninstallMethod)
	if pkg.RestartAction != "" && pkg.RestartAction != RestartActionNone {
		item["RestartAction"] = pkg.RestartAction
	} else {
		delete(item, "RestartAction")
	}
	addPkginfoString(item, "minimum_munki_version", pkg.MinimumMunkiVersion)
	addPkginfoString(item, "minimum_os_version", pkg.MinimumOSVersion)
	addPkginfoString(item, "maximum_os_version", pkg.MaximumOSVersion)
	addPkginfoStrings(item, "supported_architectures", pkg.SupportedArchitectures)
	addPkginfoStrings(item, "blocking_applications", pkg.BlockingApplications)
	addPkginfoStrings(item, "requires", pkg.Requires)
	addPkginfoStrings(item, "update_for", pkg.UpdateFor)
	addPkginfoBool(item, "unattended_install", pkg.UnattendedInstall)
	addPkginfoBool(item, "unattended_uninstall", pkg.UnattendedUninstall)
	addPkginfoBool(item, "uninstallable", pkg.Uninstallable)
	addPkginfoBool(item, "OnDemand", pkg.OnDemand)
	addPkginfoBool(item, "precache", pkg.Precache)
	addPkginfoString(item, "icon_name", pkg.IconName)
	addPkginfoString(item, "icon_hash", pkg.IconHash)

	raw, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func packageExtraPkginfo(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New("pkginfo extra data must be a JSON object")
	}
	stripOwnedPkginfoKeys(item)
	return item, nil
}

func cleanExtraPkginfo(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return raw
	}
	stripOwnedPkginfoKeys(object)
	if len(object) == 0 {
		return json.RawMessage(`{}`)
	}
	cleaned, err := json.Marshal(object)
	if err != nil {
		return raw
	}
	return cleaned
}

func stripOwnedPkginfoKeys(item map[string]any) {
	for key := range importedPkginfoKeys {
		delete(item, key)
	}
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
