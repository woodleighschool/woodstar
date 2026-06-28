package packages

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/schema"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// InstallerType describes the Munki installer modes Woodstar exposes for app
// distribution. OS upgrades and profile delivery are outside this package scope.
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
	return schema.StringEnum(installerTypeValues...)
}

// RestartAction describes Munki's RestartAction values.
type RestartAction string

const (
	RestartActionRequireLogout    RestartAction = "RequireLogout"
	RestartActionRecommendRestart RestartAction = "RecommendRestart"
	RestartActionRequireRestart   RestartAction = "RequireRestart"
	RestartActionRequireShutdown  RestartAction = "RequireShutdown"
)

var restartActionValues = []RestartAction{
	RestartActionRequireLogout,
	RestartActionRecommendRestart,
	RestartActionRequireRestart,
	RestartActionRequireShutdown,
}

func (RestartAction) Schema(_ huma.Registry) *huma.Schema {
	return schema.StringEnum(restartActionValues...)
}

// UninstallMethod describes the Munki uninstall modes Woodstar exposes.
// Absence means the package is not uninstallable.
type UninstallMethod string

const (
	UninstallMethodRemovePackages    UninstallMethod = "removepackages"
	UninstallMethodRemoveCopiedItems UninstallMethod = "remove_copied_items"
	UninstallMethodUninstallScript   UninstallMethod = "uninstall_script"
)

var uninstallMethodValues = []UninstallMethod{
	UninstallMethodRemovePackages,
	UninstallMethodRemoveCopiedItems,
	UninstallMethodUninstallScript,
}

func (UninstallMethod) Schema(_ huma.Registry) *huma.Schema {
	return schema.StringEnum(uninstallMethodValues...)
}

// PackageReference points to Woodstar-authored software, optionally pinned to one package version.
type PackageReference struct {
	SoftwareID     int64  `json:"software_id"`
	PackageID      int64  `json:"package_id,omitempty"`
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
	AttributeSetting int32  `json:"attribute_setting"`
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
	return schema.StringEnum(installItemTypeValues...)
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
	MinimumUpdateVersion  string                 `json:"minimum_update_version,omitempty"`
	MD5Checksum           string                 `json:"md5checksum,omitempty"`
	MinimumOSVersion      string                 `json:"minimum_os_version,omitempty"`
	InstallerItemLocation string                 `json:"installer_item_location,omitempty"`
}

// PackageReceipt is one Munki receipt entry.
type PackageReceipt struct {
	PackageID     string `json:"package_id"`
	Version       string `json:"version,omitempty"`
	Name          string `json:"name,omitempty"`
	InstalledSize int64  `json:"installed_size,omitempty"`
	Optional      bool   `json:"optional,omitempty"`
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
	BlockingApplicationsNone bool                                  `json:"blocking_applications_none,omitempty"`
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
	InstallerObjectID        *int64                                `json:"installer_object_id,omitempty"`
	Eligible                 bool                                  `json:"eligible"`
}

// PackageCreateMutation creates one package version under selected Munki software.
type PackageCreateMutation struct {
	PackageMutation

	SoftwareID int64 `json:"software_id" minimum:"1"`
}

// InstallerFile is the Munki-facing view of a package-owned installer object.
type InstallerFile struct {
	Filename              string `json:"filename"`
	InstallerItemLocation string `json:"installer_item_location"`
	SizeBytes             int64  `json:"size_bytes"`
	SHA256                string `json:"sha256"`
}

// Package is one Woodstar-authored Munki package version available for targeting.
type Package struct {
	ID                       int64                                 `json:"id"`
	SoftwareID               int64                                 `json:"software_id"`
	SoftwareName             string                                `json:"software_name"`
	SoftwareDescription      string                                `json:"software_description"`
	SoftwareCategory         string                                `json:"software_category"`
	SoftwareDeveloper        string                                `json:"software_developer"`
	Version                  string                                `json:"version"`
	InstallerType            InstallerType                         `json:"installer_type"`
	UnattendedInstall        bool                                  `json:"unattended_install"`
	UnattendedUninstall      bool                                  `json:"unattended_uninstall"`
	UninstallMethod          UninstallMethod                       `json:"uninstall_method,omitempty"`
	RestartAction            RestartAction                         `json:"restart_action,omitempty"`
	MinimumMunkiVersion      string                                `json:"minimum_munki_version"`
	MinimumOSVersion         string                                `json:"minimum_os_version"`
	MaximumOSVersion         string                                `json:"maximum_os_version"`
	SupportedArchitectures   []string                              `json:"supported_architectures"`
	BlockingApplications     []string                              `json:"blocking_applications"`
	BlockingApplicationsNone bool                                  `json:"blocking_applications_none"`
	InstallableCondition     string                                `json:"installable_condition"`
	BlockingAppsManualQuit   bool                                  `json:"blocking_applications_manual_quit_only"`
	BlockingAppsQuitScript   string                                `json:"blocking_applications_quit_script"`
	Requires                 []PackageReference                    `json:"requires"`
	UpdateFor                []PackageReference                    `json:"update_for"`
	OnDemand                 bool                                  `json:"on_demand"`
	Precache                 bool                                  `json:"precache"`
	Autoremove               bool                                  `json:"autoremove"`
	AppleItem                bool                                  `json:"apple_item"`
	SuppressBundleRelocation bool                                  `json:"suppress_bundle_relocation"`
	ForceInstallAfterDate    *time.Time                            `json:"force_install_after_date,omitempty"`
	InstalledSize            int64                                 `json:"installed_size"`
	InstallerFile            *InstallerFile                        `json:"installer_file,omitempty"`
	PackagePath              string                                `json:"package_path"`
	InstallerChoicesXML      []PackageInstallerChoice              `json:"installer_choices_xml"`
	InstallerEnvironment     []PackageInstallerEnvironmentVariable `json:"installer_environment"`
	Installs                 []PackageInstallItem                  `json:"installs"`
	Receipts                 []PackageReceipt                      `json:"receipts"`
	ItemsToCopy              []PackageItemToCopy                   `json:"items_to_copy"`
	Notes                    string                                `json:"notes"`
	InstallcheckScript       string                                `json:"installcheck_script"`
	UninstallcheckScript     string                                `json:"uninstallcheck_script"`
	PreinstallScript         string                                `json:"preinstall_script"`
	PostinstallScript        string                                `json:"postinstall_script"`
	PreuninstallScript       string                                `json:"preuninstall_script"`
	PostuninstallScript      string                                `json:"postuninstall_script"`
	UninstallScript          string                                `json:"uninstall_script"`
	VersionScript            string                                `json:"version_script"`
	PreinstallAlert          PackageAlert                          `json:"preinstall_alert"`
	PreuninstallAlert        PackageAlert                          `json:"preuninstall_alert"`
	InstallerObjectID        *int64                                `json:"installer_object_id,omitempty"`
	SoftwareIconObjectID     *int64                                `json:"-"`
	Eligible                 bool                                  `json:"eligible"`
	CreatedAt                time.Time                             `json:"created_at"`
	UpdatedAt                time.Time                             `json:"updated_at"`
}

type PackageListParams struct {
	dbutil.ListParams

	InstallerTypes []string
	SoftwareID     int64
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
		switch arch {
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
		if receipt.InstalledSize < 0 {
			return fmt.Errorf("%w: receipts entries require non-negative installed_size", dbutil.ErrInvalidInput)
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
	if m.UninstallMethod == UninstallMethodRemoveCopiedItems && len(m.ItemsToCopy) == 0 {
		return fmt.Errorf("%w: remove_copied_items requires items_to_copy entries", dbutil.ErrInvalidInput)
	}
	for _, variable := range m.InstallerEnvironment {
		if strings.TrimSpace(variable.Name) == "" {
			return fmt.Errorf("%w: installer_environment entries require name", dbutil.ErrInvalidInput)
		}
	}
	for _, app := range m.BlockingApplications {
		if strings.TrimSpace(app) == "" {
			return fmt.Errorf("%w: blocking_applications entries must not be blank", dbutil.ErrInvalidInput)
		}
	}
	if m.BlockingApplicationsNone && len(m.BlockingApplications) > 0 {
		return fmt.Errorf(
			"%w: blocking_applications_none cannot be set with blocking_applications entries",
			dbutil.ErrInvalidInput,
		)
	}
	for _, choice := range m.InstallerChoicesXML {
		if strings.TrimSpace(choice.ChoiceIdentifier) == "" {
			return fmt.Errorf("%w: installer_choices_xml entries require choice_identifier", dbutil.ErrInvalidInput)
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
		if ref.SoftwareID <= 0 {
			return fmt.Errorf("%w: %s entries require software_id", dbutil.ErrInvalidInput, field)
		}
		if ref.PackageID < 0 {
			return fmt.Errorf("%w: %s entries have invalid package_id", dbutil.ErrInvalidInput, field)
		}
	}
	return nil
}
