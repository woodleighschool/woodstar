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

	// ArtifactKindIcon is an icon referenced by Munki catalogs.
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
// normal authoring flows. InstallerTypePkg is Woodstar's default package mode.
type InstallerType string

const (
	InstallerTypePkg                 InstallerType = "pkg"
	InstallerTypeNoPkg               InstallerType = "nopkg"
	InstallerTypeCopyFromDMG         InstallerType = "copy_from_dmg"
	InstallerTypeProfile             InstallerType = "profile"
	InstallerTypeAppleUpdateMetadata InstallerType = "apple_update_metadata"
	InstallerTypeStartOSInstall      InstallerType = "startosinstall"
	InstallerTypeStageOSInstaller    InstallerType = "stage_os_installer"
)

var installerTypeValues = []InstallerType{
	InstallerTypePkg,
	InstallerTypeNoPkg,
	InstallerTypeCopyFromDMG,
	InstallerTypeProfile,
	InstallerTypeAppleUpdateMetadata,
	InstallerTypeStartOSInstall,
	InstallerTypeStageOSInstaller,
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

// UninstallMethod describes Woodstar's typed Munki uninstall modes.
type UninstallMethod string

const (
	UninstallMethodNone              UninstallMethod = "none"
	UninstallMethodRemovePackages    UninstallMethod = "removepackages"
	UninstallMethodRemoveCopiedItems UninstallMethod = "remove_copied_items"
	UninstallMethodRemoveProfile     UninstallMethod = "remove_profile"
	UninstallMethodRemoveApp         UninstallMethod = "remove_app"
	UninstallMethodUninstallScript   UninstallMethod = "uninstall_script"
	UninstallMethodUninstallPackage  UninstallMethod = "uninstall_package"
	UninstallMethodCustom            UninstallMethod = "custom"
)

var uninstallMethodValues = []UninstallMethod{
	UninstallMethodNone,
	UninstallMethodRemovePackages,
	UninstallMethodRemoveCopiedItems,
	UninstallMethodRemoveProfile,
	UninstallMethodRemoveApp,
	UninstallMethodUninstallScript,
	UninstallMethodUninstallPackage,
	UninstallMethodCustom,
}

func (UninstallMethod) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(uninstallMethodValues...)
}

// AssignmentAction describes the managed Munki manifest section for an assignment.
type AssignmentAction string

const (
	AssignmentActionInstall         AssignmentAction = "install"
	AssignmentActionRemove          AssignmentAction = "remove"
	AssignmentActionUpdateIfPresent AssignmentAction = "update_if_present"
	AssignmentActionNone            AssignmentAction = "none"
)

var assignmentActionValues = []AssignmentAction{
	AssignmentActionInstall,
	AssignmentActionRemove,
	AssignmentActionUpdateIfPresent,
	AssignmentActionNone,
}

func (AssignmentAction) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(assignmentActionValues...)
}

type AssignmentEffect string

const (
	AssignmentEffectInclude AssignmentEffect = "include"
	AssignmentEffectExclude AssignmentEffect = "exclude"
)

var assignmentEffectValues = []AssignmentEffect{
	AssignmentEffectInclude,
	AssignmentEffectExclude,
}

func (AssignmentEffect) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(assignmentEffectValues...)
}

// PackageSelection describes whether an assignment follows the latest eligible
// package or pins one package version.
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
	Name           string `json:"name"`
	DisplayName    string `json:"display_name,omitempty"`
	Description    string `json:"description,omitempty"`
	Category       string `json:"category,omitempty"`
	Developer      string `json:"developer,omitempty"`
	IconName       string `json:"icon_name,omitempty"`
	IconHash       string `json:"icon_hash,omitempty"`
	IconArtifactID *int64 `json:"icon_artifact_id,omitempty"`
}

// SoftwareTitle is Woodstar-managed metadata for a Munki software item.
type SoftwareTitle struct {
	ID                   int64     `json:"id"`
	Name                 string    `json:"name"`
	DisplayName          string    `json:"display_name"`
	Description          string    `json:"description"`
	Category             string    `json:"category"`
	Developer            string    `json:"developer"`
	IconName             string    `json:"icon_name"`
	IconHash             string    `json:"icon_hash"`
	IconArtifactID       *int64    `json:"icon_artifact_id,omitempty"`
	IconArtifactLocation string    `json:"icon_artifact_location,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// PackageReference points to either another Woodstar-authored package or a
// literal Munki item name.
type PackageReference struct {
	PackageID      *int64 `json:"package_id,omitempty"`
	Name           string `json:"name,omitempty"`
	PackageName    string `json:"package_name,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
}

// PackageInstallerEnvironmentVariable is one environment variable passed to a
// Munki installer process.
type PackageInstallerEnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PackageInstallItemType describes the Munki installs item matcher shape.
type PackageInstallItemType string

const (
	// PackageInstallItemApplication matches an application bundle.
	PackageInstallItemApplication PackageInstallItemType = "application"
	// PackageInstallItemBundle matches a generic bundle.
	PackageInstallItemBundle PackageInstallItemType = "bundle"
	// PackageInstallItemPlist matches a property list.
	PackageInstallItemPlist PackageInstallItemType = "plist"
	// PackageInstallItemFile matches a filesystem path.
	PackageInstallItemFile PackageInstallItemType = "file"
)

var packageInstallItemTypeValues = []PackageInstallItemType{
	PackageInstallItemApplication,
	PackageInstallItemBundle,
	PackageInstallItemPlist,
	PackageInstallItemFile,
}

func (PackageInstallItemType) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(packageInstallItemTypeValues...)
}

// PackageInstallItem is one Munki installs array entry.
type PackageInstallItem struct {
	Type                  PackageInstallItemType `json:"type"`
	Path                  string                 `json:"path"`
	BundleIdentifier      string                 `json:"bundle_identifier,omitempty"`
	BundleName            string                 `json:"bundle_name,omitempty"`
	BundleShortVersion    string                 `json:"bundle_short_version,omitempty"`
	BundleVersion         string                 `json:"bundle_version,omitempty"`
	VersionComparisonKey  string                 `json:"version_comparison_key,omitempty"`
	MD5Checksum           string                 `json:"md5checksum,omitempty"`
	MinimumOSVersion      string                 `json:"minimum_os_version,omitempty"`
	InstallerItemLocation string                 `json:"installer_item_location,omitempty"`
}

// PackageReceipt is one Munki receipt entry.
type PackageReceipt struct {
	PackageID string `json:"package_id"`
	Version   string `json:"version,omitempty"`
	Optional  bool   `json:"optional,omitempty"`
}

// PackageItemToCopy is one Munki items_to_copy entry.
type PackageItemToCopy struct {
	SourceItem      string `json:"source_item"`
	DestinationPath string `json:"destination_path"`
	DestinationItem string `json:"destination_item,omitempty"`
	User            string `json:"user,omitempty"`
	Group           string `json:"group,omitempty"`
	Mode            string `json:"mode,omitempty"`
}

// PackageAlert is a Munki install or uninstall alert.
type PackageAlert struct {
	Enabled     bool   `json:"enabled"`
	Title       string `json:"title,omitempty"`
	Detail      string `json:"detail,omitempty"`
	OKLabel     string `json:"ok_label,omitempty"`
	CancelLabel string `json:"cancel_label,omitempty"`
}

// PackageMutation is the editable shape for a Munki package version.
type PackageMutation struct {
	SoftwareID               int64                                 `json:"software_id"`
	Name                     string                                `json:"name"`
	Version                  string                                `json:"version"`
	DisplayName              string                                `json:"display_name,omitempty"`
	Description              string                                `json:"description,omitempty"`
	Category                 string                                `json:"category,omitempty"`
	Developer                string                                `json:"developer,omitempty"`
	InstallerType            InstallerType                         `json:"installer_type,omitempty"`
	UnattendedInstall        bool                                  `json:"unattended_install,omitempty"`
	UnattendedUninstall      bool                                  `json:"unattended_uninstall,omitempty"`
	Uninstallable            bool                                  `json:"uninstallable,omitempty"`
	UninstallMethod          UninstallMethod                       `json:"uninstall_method,omitempty"`
	CustomUninstallMethod    string                                `json:"custom_uninstall_method,omitempty"`
	RestartAction            RestartAction                         `json:"restart_action,omitempty"`
	MinimumMunkiVersion      string                                `json:"minimum_munki_version,omitempty"`
	MinimumOSVersion         string                                `json:"minimum_os_version,omitempty"`
	MaximumOSVersion         string                                `json:"maximum_os_version,omitempty"`
	SupportedArchitectures   []string                              `json:"supported_architectures,omitempty"`
	BlockingApplications     []string                              `json:"blocking_applications,omitempty"`
	Requires                 []PackageReference                    `json:"requires,omitempty"`
	UpdateFor                []PackageReference                    `json:"update_for,omitempty"`
	OnDemand                 bool                                  `json:"on_demand,omitempty"`
	Precache                 bool                                  `json:"precache,omitempty"`
	Autoremove               bool                                  `json:"autoremove,omitempty"`
	AppleItem                bool                                  `json:"apple_item,omitempty"`
	SuppressBundleRelocation bool                                  `json:"suppress_bundle_relocation,omitempty"`
	ForceInstallAfterDate    *time.Time                            `json:"force_install_after_date,omitempty"`
	InstalledSize            int64                                 `json:"installed_size,omitempty"`
	PayloadIdentifier        string                                `json:"payload_identifier,omitempty"`
	PackagePath              string                                `json:"package_path,omitempty"`
	InstallerChoicesXML      string                                `json:"installer_choices_xml,omitempty"`
	InstallerEnvironment     []PackageInstallerEnvironmentVariable `json:"installer_environment,omitempty"`
	Installs                 []PackageInstallItem                  `json:"installs,omitempty"`
	Receipts                 []PackageReceipt                      `json:"receipts,omitempty"`
	ItemsToCopy              []PackageItemToCopy                   `json:"items_to_copy,omitempty"`
	Notes                    string                                `json:"notes,omitempty"`
	InstallcheckScript       string                                `json:"installcheck_script,omitempty"`
	UninstallcheckScript     string                                `json:"uninstallcheck_script,omitempty"`
	PreinstallScript         string                                `json:"preinstall_script,omitempty"`
	PostinstallScript        string                                `json:"postinstall_script,omitempty"`
	PreuninstallScript       string                                `json:"preuninstall_script,omitempty"`
	PostuninstallScript      string                                `json:"postuninstall_script,omitempty"`
	UninstallScript          string                                `json:"uninstall_script,omitempty"`
	VersionScript            string                                `json:"version_script,omitempty"`
	PreinstallAlert          PackageAlert                          `json:"preinstall_alert,omitzero"`
	PreuninstallAlert        PackageAlert                          `json:"preuninstall_alert,omitzero"`
	IconName                 string                                `json:"icon_name,omitempty"`
	IconHash                 string                                `json:"icon_hash,omitempty"`
	InstallerArtifactID      *int64                                `json:"installer_artifact_id,omitempty"`
	UninstallerArtifactID    *int64                                `json:"uninstaller_artifact_id,omitempty"`
	IconArtifactID           *int64                                `json:"icon_artifact_id,omitempty"`
	Eligible                 bool                                  `json:"eligible"`
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

// Package is one Woodstar-authored Munki package version available for assignment.
type Package struct {
	ID                           int64                                 `json:"id"`
	SoftwareID                   int64                                 `json:"software_id"`
	SoftwareName                 string                                `json:"software_name"`
	SoftwareDisplayName          string                                `json:"software_display_name"`
	Name                         string                                `json:"name"`
	Version                      string                                `json:"version"`
	DisplayName                  string                                `json:"display_name"`
	Description                  string                                `json:"description"`
	Category                     string                                `json:"category"`
	Developer                    string                                `json:"developer"`
	InstallerType                InstallerType                         `json:"installer_type"`
	UnattendedInstall            bool                                  `json:"unattended_install"`
	UnattendedUninstall          bool                                  `json:"unattended_uninstall"`
	Uninstallable                bool                                  `json:"uninstallable"`
	UninstallMethod              UninstallMethod                       `json:"uninstall_method"`
	CustomUninstallMethod        string                                `json:"custom_uninstall_method"`
	RestartAction                RestartAction                         `json:"restart_action,omitempty"`
	MinimumMunkiVersion          string                                `json:"minimum_munki_version"`
	MinimumOSVersion             string                                `json:"minimum_os_version"`
	MaximumOSVersion             string                                `json:"maximum_os_version"`
	SupportedArchitectures       []string                              `json:"supported_architectures"`
	BlockingApplications         []string                              `json:"blocking_applications"`
	Requires                     []PackageReference                    `json:"requires"`
	UpdateFor                    []PackageReference                    `json:"update_for"`
	OnDemand                     bool                                  `json:"on_demand"`
	Precache                     bool                                  `json:"precache"`
	Autoremove                   bool                                  `json:"autoremove"`
	AppleItem                    bool                                  `json:"apple_item"`
	SuppressBundleRelocation     bool                                  `json:"suppress_bundle_relocation"`
	ForceInstallAfterDate        *time.Time                            `json:"force_install_after_date,omitempty"`
	InstalledSize                int64                                 `json:"installed_size"`
	PayloadIdentifier            string                                `json:"payload_identifier"`
	PackagePath                  string                                `json:"package_path"`
	InstallerChoicesXML          string                                `json:"installer_choices_xml"`
	InstallerEnvironment         []PackageInstallerEnvironmentVariable `json:"installer_environment"`
	Installs                     []PackageInstallItem                  `json:"installs"`
	Receipts                     []PackageReceipt                      `json:"receipts"`
	ItemsToCopy                  []PackageItemToCopy                   `json:"items_to_copy"`
	Notes                        string                                `json:"notes"`
	InstallcheckScript           string                                `json:"installcheck_script"`
	UninstallcheckScript         string                                `json:"uninstallcheck_script"`
	PreinstallScript             string                                `json:"preinstall_script"`
	PostinstallScript            string                                `json:"postinstall_script"`
	PreuninstallScript           string                                `json:"preuninstall_script"`
	PostuninstallScript          string                                `json:"postuninstall_script"`
	UninstallScript              string                                `json:"uninstall_script"`
	VersionScript                string                                `json:"version_script"`
	PreinstallAlert              PackageAlert                          `json:"preinstall_alert"`
	PreuninstallAlert            PackageAlert                          `json:"preuninstall_alert"`
	IconName                     string                                `json:"icon_name"`
	IconHash                     string                                `json:"icon_hash"`
	InstallerArtifactID          *int64                                `json:"installer_artifact_id,omitempty"`
	InstallerArtifactLocation    string                                `json:"installer_artifact_location,omitempty"`
	UninstallerArtifactID        *int64                                `json:"uninstaller_artifact_id,omitempty"`
	UninstallerArtifactLocation  string                                `json:"uninstaller_artifact_location,omitempty"`
	IconArtifactID               *int64                                `json:"icon_artifact_id,omitempty"`
	IconArtifactLocation         string                                `json:"icon_artifact_location,omitempty"`
	SoftwareIconName             string                                `json:"software_icon_name,omitempty"`
	SoftwareIconHash             string                                `json:"software_icon_hash,omitempty"`
	SoftwareIconArtifactID       *int64                                `json:"software_icon_artifact_id,omitempty"`
	SoftwareIconArtifactLocation string                                `json:"software_icon_artifact_location,omitempty"`
	Eligible                     bool                                  `json:"eligible"`
	CreatedAt                    time.Time                             `json:"created_at"`
	UpdatedAt                    time.Time                             `json:"updated_at"`
}

// EffectiveIconArtifactID returns the package icon override, or the software
// title icon when the package does not override it.
func (p Package) EffectiveIconArtifactID() *int64 {
	if p.IconArtifactID != nil || p.IconName != "" || p.IconHash != "" {
		return p.IconArtifactID
	}
	return p.SoftwareIconArtifactID
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

// AssignmentMutation is one ordered label row for Munki desired state.
type AssignmentMutation struct {
	SoftwareID       int64             `json:"software_id"`
	Priority         int32             `json:"priority"`
	LabelID          int64             `json:"label_id"`
	Effect           AssignmentEffect  `json:"effect"`
	Action           *AssignmentAction `json:"action,omitempty"`
	OptionalInstall  bool              `json:"optional_install,omitempty"`
	FeaturedItem     bool              `json:"featured_item,omitempty"`
	PackageSelection *PackageSelection `json:"package_selection,omitempty"`
	PinnedPackageID  *int64            `json:"pinned_package_id,omitempty"`
}

// Assignment links one Munki software title, one target label, and optional include payload.
type Assignment struct {
	ID                   int64             `json:"id"`
	SoftwareID           int64             `json:"software_id"`
	SoftwareDisplayName  string            `json:"software_display_name"`
	Priority             int32             `json:"priority"`
	LabelID              int64             `json:"label_id"`
	Effect               AssignmentEffect  `json:"effect"`
	Action               *AssignmentAction `json:"action,omitempty"`
	OptionalInstall      bool              `json:"optional_install"`
	FeaturedItem         bool              `json:"featured_item"`
	PackageSelection     *PackageSelection `json:"package_selection,omitempty"`
	PinnedPackageID      *int64            `json:"pinned_package_id,omitempty"`
	PinnedPackageName    string            `json:"pinned_package_name,omitempty"`
	PinnedPackageVersion string            `json:"pinned_package_version,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

// EffectivePackage is a host-resolved Munki package ready for manifest/catalog rendering.
type EffectivePackage struct {
	AssignmentID     int64
	SoftwareID       int64
	AssignmentEffect AssignmentEffect
	Action           AssignmentAction
	OptionalInstall  bool
	FeaturedItem     bool
	PackageSelection PackageSelection
	PinnedPackageID  *int64
	Priority         int32
	Package          Package
}

type SoftwareTitleDetail struct {
	SoftwareTitle
	Packages    []Package    `json:"packages"`
	Assignments []Assignment `json:"assignments"`
}

type PackageListParams struct {
	dbutil.ListParams
	SoftwareID int64
}

type AssignmentListParams struct {
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
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("%w: version is required", dbutil.ErrInvalidInput)
	}
	if !validInstallerType(m.InstallerType) {
		return fmt.Errorf("%w: unsupported installer_type %q", dbutil.ErrInvalidInput, m.InstallerType)
	}
	if !validUninstallMethod(m.UninstallMethod) {
		return fmt.Errorf("%w: unsupported uninstall_method %q", dbutil.ErrInvalidInput, m.UninstallMethod)
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
	if err := validatePackageReferences("requires", m.Requires); err != nil {
		return err
	}
	if err := validatePackageReferences("update_for", m.UpdateFor); err != nil {
		return err
	}
	for _, item := range m.Installs {
		if !validPackageInstallItemType(item.Type) {
			return fmt.Errorf("%w: unsupported installs type %q", dbutil.ErrInvalidInput, item.Type)
		}
		if strings.TrimSpace(item.Path) == "" {
			return fmt.Errorf("%w: installs entries require path", dbutil.ErrInvalidInput)
		}
	}
	for _, receipt := range m.Receipts {
		if strings.TrimSpace(receipt.PackageID) == "" {
			return fmt.Errorf("%w: receipts entries require package_id", dbutil.ErrInvalidInput)
		}
	}
	for _, item := range m.ItemsToCopy {
		if strings.TrimSpace(item.SourceItem) == "" || strings.TrimSpace(item.DestinationPath) == "" {
			return fmt.Errorf(
				"%w: items_to_copy entries require source_item and destination_path",
				dbutil.ErrInvalidInput,
			)
		}
	}
	return nil
}

func (m PackageImportMutation) Validate() error {
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
	if !validSHA256(m.SHA256) {
		return fmt.Errorf("%w: sha256 must be 64 lowercase hex characters", dbutil.ErrInvalidInput)
	}
	if strings.TrimSpace(m.StorageKey) == "" || strings.HasPrefix(strings.TrimSpace(m.StorageKey), "/") {
		return fmt.Errorf("%w: storage_key is required and must be relative", dbutil.ErrInvalidInput)
	}
	return nil
}

func (m AssignmentMutation) Validate() error {
	switch m.Effect {
	case AssignmentEffectInclude:
		return m.validateIncludePayload()
	case AssignmentEffectExclude:
		return m.validateExcludePayload()
	default:
		return fmt.Errorf("%w: unsupported assignment effect %q", dbutil.ErrInvalidInput, m.Effect)
	}
}

func validAssignmentAction(action AssignmentAction) bool {
	return slices.Contains(assignmentActionValues, action)
}

func validPackageSelection(selection PackageSelection) bool {
	return slices.Contains(packageSelectionValues, selection)
}

func (m AssignmentMutation) validateIncludePayload() error {
	if m.Action == nil {
		return fmt.Errorf("%w: action is required for include assignments", dbutil.ErrInvalidInput)
	}
	if !validAssignmentAction(*m.Action) {
		return fmt.Errorf("%w: unsupported assignment action %q", dbutil.ErrInvalidInput, *m.Action)
	}
	if m.PackageSelection == nil {
		return fmt.Errorf("%w: package_selection is required for include assignments", dbutil.ErrInvalidInput)
	}
	if !validPackageSelection(*m.PackageSelection) {
		return fmt.Errorf(
			"%w: unsupported package_selection %q",
			dbutil.ErrInvalidInput,
			*m.PackageSelection,
		)
	}
	switch *m.PackageSelection {
	case PackageSelectionLatestEligible:
		if m.PinnedPackageID != nil {
			return fmt.Errorf(
				"%w: pinned_package_id must be empty for latest_eligible selection",
				dbutil.ErrInvalidInput,
			)
		}
	case PackageSelectionSpecific:
		if m.PinnedPackageID == nil {
			return fmt.Errorf("%w: pinned_package_id is required", dbutil.ErrInvalidInput)
		}
	}
	if m.FeaturedItem && !m.OptionalInstall {
		return fmt.Errorf("%w: featured_item requires optional_install", dbutil.ErrInvalidInput)
	}
	if *m.Action == AssignmentActionRemove && (m.OptionalInstall || m.FeaturedItem) {
		return fmt.Errorf(
			"%w: remove assignments cannot be optional_installs or featured_items",
			dbutil.ErrInvalidInput,
		)
	}
	return nil
}

func (m AssignmentMutation) validateExcludePayload() error {
	if m.Action != nil || m.PackageSelection != nil || m.PinnedPackageID != nil || m.OptionalInstall || m.FeaturedItem {
		return fmt.Errorf("%w: exclude assignments cannot carry Munki payload", dbutil.ErrInvalidInput)
	}
	return nil
}

func validInstallerType(installerType InstallerType) bool {
	return installerType == "" || slices.Contains(installerTypeValues, installerType)
}

func validUninstallMethod(uninstallMethod UninstallMethod) bool {
	return uninstallMethod == "" || slices.Contains(uninstallMethodValues, uninstallMethod)
}

func validRestartAction(restartAction RestartAction) bool {
	return restartAction == "" || slices.Contains(restartActionValues, restartAction)
}

func validPackageInstallItemType(itemType PackageInstallItemType) bool {
	return slices.Contains(packageInstallItemTypeValues, itemType)
}

func validatePackageReferences(field string, references []PackageReference) error {
	for _, ref := range references {
		name := strings.TrimSpace(ref.Name)
		if ref.PackageID == nil && name == "" {
			return fmt.Errorf("%w: %s entries require package_id or name", dbutil.ErrInvalidInput, field)
		}
		if ref.PackageID != nil && name != "" {
			return fmt.Errorf("%w: %s entries cannot set both package_id and name", dbutil.ErrInvalidInput, field)
		}
	}
	return nil
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

func packagePkginfo(pkg Package) map[string]any {
	item := make(map[string]any)
	item["name"] = pkg.Name
	item["version"] = pkg.Version

	addPkginfoString(item, "display_name", pkg.DisplayName)
	addPkginfoString(item, "description", pkg.Description)
	addPkginfoString(item, "category", pkg.Category)
	addPkginfoString(item, "developer", pkg.Developer)
	if pkg.InstallerType != "" && pkg.InstallerType != InstallerTypePkg {
		item["installer_type"] = pkg.InstallerType
	}
	addPkginfoUninstallMethod(item, pkg)
	if pkg.RestartAction != "" && pkg.RestartAction != RestartActionNone {
		item["RestartAction"] = pkg.RestartAction
	}
	addPkginfoString(item, "minimum_munki_version", pkg.MinimumMunkiVersion)
	addPkginfoString(item, "minimum_os_version", pkg.MinimumOSVersion)
	addPkginfoString(item, "maximum_os_version", pkg.MaximumOSVersion)
	addPkginfoStrings(item, "supported_architectures", pkg.SupportedArchitectures)
	item["blocking_applications"] = cleanStringList(pkg.BlockingApplications)
	addPkginfoStrings(item, "requires", packageReferenceNames(pkg.Requires))
	addPkginfoStrings(item, "update_for", packageReferenceNames(pkg.UpdateFor))
	addPkginfoBool(item, "unattended_install", pkg.UnattendedInstall)
	addPkginfoBool(item, "unattended_uninstall", pkg.UnattendedUninstall)
	addPkginfoBool(item, "uninstallable", pkg.Uninstallable)
	addPkginfoBool(item, "OnDemand", pkg.OnDemand)
	addPkginfoBool(item, "precache", pkg.Precache)
	addPkginfoBool(item, "autoremove", pkg.Autoremove)
	addPkginfoBool(item, "apple_item", pkg.AppleItem)
	addPkginfoBool(item, "suppress_bundle_relocation", pkg.SuppressBundleRelocation)
	if pkg.ForceInstallAfterDate != nil {
		item["force_install_after_date"] = *pkg.ForceInstallAfterDate
	}
	if pkg.InstalledSize > 0 {
		item["installed_size"] = pkg.InstalledSize
	}
	addPkginfoString(item, "payload_identifier", pkg.PayloadIdentifier)
	addPkginfoString(item, "package_path", pkg.PackagePath)
	addPkginfoString(item, "installer_choices_xml", pkg.InstallerChoicesXML)
	addPkginfoInstallerEnvironment(item, pkg.InstallerEnvironment)
	addPkginfoInstallItems(item, pkg.Installs)
	addPkginfoReceipts(item, pkg.Receipts)
	addPkginfoItemsToCopy(item, pkg.ItemsToCopy)
	addPkginfoString(item, "notes", pkg.Notes)
	addPkginfoString(item, "installcheck_script", pkg.InstallcheckScript)
	addPkginfoString(item, "uninstallcheck_script", pkg.UninstallcheckScript)
	addPkginfoString(item, "preinstall_script", pkg.PreinstallScript)
	addPkginfoString(item, "postinstall_script", pkg.PostinstallScript)
	addPkginfoString(item, "preuninstall_script", pkg.PreuninstallScript)
	addPkginfoString(item, "postuninstall_script", pkg.PostuninstallScript)
	addPkginfoString(item, "uninstall_script", pkg.UninstallScript)
	addPkginfoString(item, "version_script", pkg.VersionScript)
	addPkginfoAlert(item, "preinstall_alert", pkg.PreinstallAlert)
	addPkginfoAlert(item, "preuninstall_alert", pkg.PreuninstallAlert)
	iconName, iconHash := packageIconFields(pkg)
	addPkginfoString(item, "icon_name", iconName)
	addPkginfoString(item, "icon_hash", iconHash)

	return item
}

func packageIconFields(pkg Package) (string, string) {
	if pkg.IconArtifactID != nil || pkg.IconName != "" || pkg.IconHash != "" {
		return pkg.IconName, pkg.IconHash
	}
	return pkg.SoftwareIconName, pkg.SoftwareIconHash
}

func addPkginfoUninstallMethod(item map[string]any, pkg Package) {
	switch pkg.UninstallMethod {
	case "", UninstallMethodNone:
	case UninstallMethodCustom:
		addPkginfoString(item, "uninstall_method", pkg.CustomUninstallMethod)
	default:
		addPkginfoString(item, "uninstall_method", string(pkg.UninstallMethod))
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

func addPkginfoInstallerEnvironment(item map[string]any, values []PackageInstallerEnvironmentVariable) {
	environment := make(map[string]string, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name != "" {
			environment[name] = value.Value
		}
	}
	if len(environment) > 0 {
		item["installer_environment"] = environment
	}
}

func addPkginfoInstallItems(item map[string]any, values []PackageInstallItem) {
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		path := strings.TrimSpace(value.Path)
		if path == "" {
			continue
		}
		record := map[string]any{
			"type": string(value.Type),
			"path": path,
		}
		addPkginfoString(record, "CFBundleIdentifier", value.BundleIdentifier)
		addPkginfoString(record, "CFBundleName", value.BundleName)
		addPkginfoString(record, "CFBundleShortVersionString", value.BundleShortVersion)
		addPkginfoString(record, "CFBundleVersion", value.BundleVersion)
		addPkginfoString(record, "version_comparison_key", value.VersionComparisonKey)
		addPkginfoString(record, "md5checksum", value.MD5Checksum)
		addPkginfoString(record, "minimum_os_version", value.MinimumOSVersion)
		addPkginfoString(record, "installer_item_location", value.InstallerItemLocation)
		out = append(out, record)
	}
	if len(out) > 0 {
		item["installs"] = out
	}
}

func addPkginfoReceipts(item map[string]any, values []PackageReceipt) {
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		packageID := strings.TrimSpace(value.PackageID)
		if packageID == "" {
			continue
		}
		record := map[string]any{pkginfoReceiptPackageIDKey: packageID}
		addPkginfoString(record, "version", value.Version)
		if value.Optional {
			record["optional"] = true
		}
		out = append(out, record)
	}
	if len(out) > 0 {
		item["receipts"] = out
	}
}

func addPkginfoItemsToCopy(item map[string]any, values []PackageItemToCopy) {
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		sourceItem := strings.TrimSpace(value.SourceItem)
		destinationPath := strings.TrimSpace(value.DestinationPath)
		if sourceItem == "" || destinationPath == "" {
			continue
		}
		record := map[string]any{
			"source_item":      sourceItem,
			"destination_path": destinationPath,
		}
		addPkginfoString(record, "destination_item", value.DestinationItem)
		addPkginfoString(record, "user", value.User)
		addPkginfoString(record, "group", value.Group)
		addPkginfoString(record, "mode", value.Mode)
		out = append(out, record)
	}
	if len(out) > 0 {
		item["items_to_copy"] = out
	}
}

func addPkginfoAlert(item map[string]any, key string, alert PackageAlert) {
	if !alert.Enabled {
		return
	}
	record := map[string]any{}
	addPkginfoString(record, "alert_title", alert.Title)
	addPkginfoString(record, "alert_detail", alert.Detail)
	addPkginfoString(record, "ok_label", alert.OKLabel)
	addPkginfoString(record, "cancel_label", alert.CancelLabel)
	if len(record) > 0 {
		item[key] = record
	}
}

func packageReferenceNames(references []PackageReference) []string {
	out := make([]string, 0, len(references))
	for _, ref := range references {
		name := packageReferenceName(ref)
		if name != "" {
			out = append(out, name)
		}
	}
	return cleanStringList(out)
}

func packageReferenceName(ref PackageReference) string {
	if ref.PackageID != nil {
		name := strings.TrimSpace(ref.PackageName)
		version := strings.TrimSpace(ref.PackageVersion)
		if name == "" {
			return ""
		}
		if version == "" {
			return name
		}
		return name + "--" + version
	}
	return strings.TrimSpace(ref.Name)
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
