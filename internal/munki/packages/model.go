package packages

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
)

// InstallerType describes the package installer mode Woodstar exposes in
// normal authoring flows. InstallerTypePkg is Woodstar's default package mode.
type InstallerType string

const (
	InstallerTypePkg         InstallerType = "pkg"
	InstallerTypeNoPkg       InstallerType = "nopkg"
	InstallerTypeCopyFromDMG InstallerType = "copy_from_dmg"
)

var installerTypeValues = []InstallerType{
	InstallerTypePkg,
	InstallerTypeNoPkg,
	InstallerTypeCopyFromDMG,
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
	UninstallMethodUninstallScript   UninstallMethod = "uninstall_script"
	UninstallMethodUninstallPackage  UninstallMethod = "uninstall_package"
)

var uninstallMethodValues = []UninstallMethod{
	UninstallMethodNone,
	UninstallMethodRemovePackages,
	UninstallMethodRemoveCopiedItems,
	UninstallMethodUninstallScript,
	UninstallMethodUninstallPackage,
}

func (UninstallMethod) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(uninstallMethodValues...)
}

// PackageReference points to another Woodstar-authored package.
type PackageReference struct {
	PackageID      int64  `json:"package_id"`
	SoftwareID     int64  `json:"software_id,omitempty"`
	SoftwareName   string `json:"software_name,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
}

// PackageInstallerEnvironmentVariable is one environment variable passed to a Munki installer process.
type PackageInstallerEnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PackageInstallerChoice is one Munki installer choice change entry.
type PackageInstallerChoice struct {
	ChoiceIdentifier string `json:"choice_identifier,omitempty"`
	ChoiceAttribute  string `json:"choice_attribute,omitempty"`
	AttributeSetting int    `json:"attribute_setting"`
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

var installItemTypeValues = []PackageInstallItemType{
	PackageInstallItemApplication,
	PackageInstallItemBundle,
	PackageInstallItemPlist,
	PackageInstallItemFile,
}

func (PackageInstallItemType) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(installItemTypeValues...)
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
	Version                  string                                `json:"version"                                          minLength:"1"`
	InstallerType            InstallerType                         `json:"installer_type,omitempty"`
	UnattendedInstall        bool                                  `json:"unattended_install,omitempty"`
	UnattendedUninstall      bool                                  `json:"unattended_uninstall,omitempty"`
	UninstallMethod          UninstallMethod                       `json:"uninstall_method,omitempty"`
	RestartAction            RestartAction                         `json:"restart_action,omitempty"`
	MinimumMunkiVersion      string                                `json:"minimum_munki_version,omitempty"`
	MinimumOSVersion         string                                `json:"minimum_os_version,omitempty"`
	MaximumOSVersion         string                                `json:"maximum_os_version,omitempty"`
	SupportedArchitectures   []string                              `json:"supported_architectures,omitempty"`
	BlockingApplications     []string                              `json:"blocking_applications,omitempty"`
	InstallableCondition     string                                `json:"installable_condition,omitempty"`
	BlockingAppsManualQuit   bool                                  `json:"blocking_applications_manual_quit_only,omitempty"`
	BlockingAppsQuitScript   string                                `json:"blocking_applications_quit_script,omitempty"`
	Requires                 []PackageReference                    `json:"requires,omitempty"`
	UpdateFor                []PackageReference                    `json:"update_for,omitempty"`
	OnDemand                 bool                                  `json:"on_demand,omitempty"`
	Precache                 bool                                  `json:"precache,omitempty"`
	Autoremove               bool                                  `json:"autoremove,omitempty"`
	AppleItem                bool                                  `json:"apple_item,omitempty"`
	SuppressBundleRelocation bool                                  `json:"suppress_bundle_relocation,omitempty"`
	ForceInstallAfterDate    *time.Time                            `json:"force_install_after_date,omitempty"`
	InstalledSize            int64                                 `json:"installed_size,omitempty"`
	PackagePath              string                                `json:"package_path,omitempty"`
	InstallerChoicesXML      []PackageInstallerChoice              `json:"installer_choices_xml,omitempty"`
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
	InstallerArtifactID      *int64                                `json:"installer_artifact_id,omitempty"`
	UninstallerArtifactID    *int64                                `json:"uninstaller_artifact_id,omitempty"`
	Eligible                 bool                                  `json:"eligible"`
}

// PackageCreateMutation creates one package version under selected Munki software.
type PackageCreateMutation struct {
	SoftwareID int64 `json:"software_id" minimum:"1"`
	PackageMutation
}

// Package is one Woodstar-authored Munki package version available for targeting.
type Package struct {
	ID                          int64                                 `json:"id"`
	SoftwareID                  int64                                 `json:"software_id"`
	SoftwareName                string                                `json:"software_name"`
	SoftwareDescription         string                                `json:"software_description"`
	SoftwareCategory            string                                `json:"software_category"`
	SoftwareDeveloper           string                                `json:"software_developer"`
	Version                     string                                `json:"version"`
	InstallerType               InstallerType                         `json:"installer_type"`
	UnattendedInstall           bool                                  `json:"unattended_install"`
	UnattendedUninstall         bool                                  `json:"unattended_uninstall"`
	UninstallMethod             UninstallMethod                       `json:"uninstall_method"`
	RestartAction               RestartAction                         `json:"restart_action,omitempty"`
	MinimumMunkiVersion         string                                `json:"minimum_munki_version"`
	MinimumOSVersion            string                                `json:"minimum_os_version"`
	MaximumOSVersion            string                                `json:"maximum_os_version"`
	SupportedArchitectures      []string                              `json:"supported_architectures"`
	BlockingApplications        []string                              `json:"blocking_applications"`
	InstallableCondition        string                                `json:"installable_condition"`
	BlockingAppsManualQuit      bool                                  `json:"blocking_applications_manual_quit_only"`
	BlockingAppsQuitScript      string                                `json:"blocking_applications_quit_script"`
	Requires                    []PackageReference                    `json:"requires"`
	UpdateFor                   []PackageReference                    `json:"update_for"`
	OnDemand                    bool                                  `json:"on_demand"`
	Precache                    bool                                  `json:"precache"`
	Autoremove                  bool                                  `json:"autoremove"`
	AppleItem                   bool                                  `json:"apple_item"`
	SuppressBundleRelocation    bool                                  `json:"suppress_bundle_relocation"`
	ForceInstallAfterDate       *time.Time                            `json:"force_install_after_date,omitempty"`
	InstalledSize               int64                                 `json:"installed_size"`
	PackagePath                 string                                `json:"package_path"`
	InstallerChoicesXML         []PackageInstallerChoice              `json:"installer_choices_xml"`
	InstallerEnvironment        []PackageInstallerEnvironmentVariable `json:"installer_environment"`
	Installs                    []PackageInstallItem                  `json:"installs"`
	Receipts                    []PackageReceipt                      `json:"receipts"`
	ItemsToCopy                 []PackageItemToCopy                   `json:"items_to_copy"`
	Notes                       string                                `json:"notes"`
	InstallcheckScript          string                                `json:"installcheck_script"`
	UninstallcheckScript        string                                `json:"uninstallcheck_script"`
	PreinstallScript            string                                `json:"preinstall_script"`
	PostinstallScript           string                                `json:"postinstall_script"`
	PreuninstallScript          string                                `json:"preuninstall_script"`
	PostuninstallScript         string                                `json:"postuninstall_script"`
	UninstallScript             string                                `json:"uninstall_script"`
	VersionScript               string                                `json:"version_script"`
	PreinstallAlert             PackageAlert                          `json:"preinstall_alert"`
	PreuninstallAlert           PackageAlert                          `json:"preuninstall_alert"`
	InstallerArtifactID         *int64                                `json:"installer_artifact_id,omitempty"`
	InstallerArtifactLocation   string                                `json:"installer_artifact_location,omitempty"`
	UninstallerArtifactID       *int64                                `json:"uninstaller_artifact_id,omitempty"`
	UninstallerArtifactLocation string                                `json:"uninstaller_artifact_location,omitempty"`
	Eligible                    bool                                  `json:"eligible"`
	CreatedAt                   time.Time                             `json:"created_at"`
	UpdatedAt                   time.Time                             `json:"updated_at"`
}

// IconRef is parent software icon metadata projected alongside package rows.
type IconRef struct {
	Name             string
	Hash             string
	ArtifactID       *int64
	ArtifactLocation string
}

// PackageRecord is a package row joined with parent software icon context.
type PackageRecord struct {
	Package
	SoftwareIcon IconRef
}

type PackageListParams struct {
	dbutil.ListParams
	SoftwareID int64
}

func (m PackageMutation) Validate() error {
	if err := m.validateEnums(); err != nil {
		return err
	}
	if err := validateArchitectures(m.SupportedArchitectures); err != nil {
		return err
	}
	if err := validateReferences("requires", m.Requires); err != nil {
		return err
	}
	if err := validateReferences("update_for", m.UpdateFor); err != nil {
		return err
	}
	return m.validateCollections()
}

func (m PackageMutation) validateEnums() error {
	if !validInstallerType(m.InstallerType) {
		return fmt.Errorf("%w: unsupported installer_type %q", dbutil.ErrInvalidInput, m.InstallerType)
	}
	if !validUninstallMethod(m.UninstallMethod) {
		return fmt.Errorf("%w: unsupported uninstall_method %q", dbutil.ErrInvalidInput, m.UninstallMethod)
	}
	if !validRestartAction(m.RestartAction) {
		return fmt.Errorf("%w: unsupported restart_action %q", dbutil.ErrInvalidInput, m.RestartAction)
	}
	return nil
}

func validateArchitectures(architectures []string) error {
	for _, arch := range architectures {
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

func (m PackageMutation) validateCollections() error {
	for _, item := range m.Installs {
		if !validInstallItemType(item.Type) {
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

func (m PackageCreateMutation) Validate() error {
	if m.SoftwareID <= 0 {
		return fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	return m.PackageMutation.Validate()
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

func validInstallItemType(itemType PackageInstallItemType) bool {
	return slices.Contains(installItemTypeValues, itemType)
}

func validateReferences(field string, references []PackageReference) error {
	for _, ref := range references {
		if ref.PackageID <= 0 {
			return fmt.Errorf("%w: %s entries require package_id", dbutil.ErrInvalidInput, field)
		}
	}
	return nil
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
