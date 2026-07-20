// Package packages models and persists Munki installers and package metadata.
package packages

import (
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
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
	return openapischema.StringEnum(installerTypeValues...)
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
	return openapischema.StringEnum(restartActionValues...)
}

// UninstallMethod describes the Munki uninstall modes Woodstar exposes.
// Absence means no uninstall mechanism is configured.
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
	return openapischema.StringEnum(uninstallMethodValues...)
}

// PackageReferenceMutation selects Woodstar-authored software and optionally one package version.
type PackageReferenceMutation struct {
	SoftwareID int64 `json:"software_id"          validate:"gt=0"  minimum:"1"`
	PackageID  int64 `json:"package_id,omitempty" validate:"gte=0" minimum:"0"`
}

// PackageReference describes a hydrated Munki package relation.
type PackageReference struct {
	SoftwareID     int64  `json:"software_id"               validate:"gt=0"  minimum:"1"`
	PackageID      int64  `json:"package_id,omitempty"      validate:"gte=0" minimum:"0"`
	SoftwareName   string `json:"software_name"`
	PackageVersion string `json:"package_version,omitempty"`
}

// PackageInstallerEnvironmentVariable is one environment variable passed to a Munki installer process.
type PackageInstallerEnvironmentVariable struct {
	Name  string `json:"name"  validate:"required,notblank" minLength:"1"`
	Value string `json:"value"`
}

// PackageInstallerChoice is one Munki installer choice change entry.
type PackageInstallerChoice struct {
	ChoiceIdentifier string `json:"choice_identifier,omitempty" validate:"required,notblank" minLength:"1"`
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
	return openapischema.StringEnum(installItemTypeValues...)
}

// PackageInstallItem is one Munki installs array entry.
type PackageInstallItem struct {
	Type                  PackageInstallItemType `json:"type"                              validate:"required,oneof=application bundle plist file"`
	Path                  string                 `json:"path"                              validate:"required,notblank"                            minLength:"1"`
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
	PackageID     string `json:"package_id"               validate:"required,notblank" minLength:"1"`
	Version       string `json:"version,omitempty"`
	Name          string `json:"name,omitempty"`
	InstalledSize int64  `json:"installed_size,omitempty" validate:"gte=0"                           minimum:"0"`
	Optional      bool   `json:"optional,omitempty"`
}

// PackageItemToCopy is one Munki items_to_copy entry.
type PackageItemToCopy struct {
	SourceItem      string `json:"source_item"                validate:"required,notblank" minLength:"1"`
	DestinationPath string `json:"destination_path"           validate:"required,notblank" minLength:"1"`
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
	Version                  string                                `json:"version"                                          minLength:"1" validate:"required,notblank"`
	InstallerType            InstallerType                         `json:"installer_type,omitempty"                                       validate:"omitempty,oneof=pkg nopkg copy_from_dmg"`
	UnattendedInstall        bool                                  `json:"unattended_install,omitempty"`
	UnattendedUninstall      bool                                  `json:"unattended_uninstall,omitempty"`
	Uninstallable            bool                                  `json:"uninstallable,omitempty"`
	UninstallMethod          UninstallMethod                       `json:"uninstall_method,omitempty"                                     validate:"omitempty,oneof=removepackages remove_copied_items uninstall_script"`
	RestartAction            RestartAction                         `json:"restart_action,omitempty"                                       validate:"omitempty,oneof=RequireLogout RecommendRestart RequireRestart RequireShutdown"`
	MinimumMunkiVersion      string                                `json:"minimum_munki_version,omitempty"`
	MinimumOSVersion         string                                `json:"minimum_os_version,omitempty"`
	MaximumOSVersion         string                                `json:"maximum_os_version,omitempty"`
	SupportedArchitectures   []string                              `json:"supported_architectures,omitempty"                              validate:"dive,oneof=arm64 x86_64"`
	BlockingApplications     []string                              `json:"blocking_applications,omitempty"                                validate:"dive,required,notblank"`
	BlockingApplicationsNone bool                                  `json:"blocking_applications_none,omitempty"`
	InstallableCondition     string                                `json:"installable_condition,omitempty"`
	BlockingAppsManualQuit   bool                                  `json:"blocking_applications_manual_quit_only,omitempty"`
	BlockingAppsQuitScript   string                                `json:"blocking_applications_quit_script,omitempty"`
	Requires                 []PackageReferenceMutation            `json:"requires,omitempty"                                             validate:"dive"`
	UpdateFor                []PackageReferenceMutation            `json:"update_for,omitempty"                                           validate:"dive"`
	OnDemand                 bool                                  `json:"on_demand,omitempty"`
	Precache                 bool                                  `json:"precache,omitempty"`
	Autoremove               bool                                  `json:"autoremove,omitempty"`
	AppleItem                bool                                  `json:"apple_item,omitempty"`
	SuppressBundleRelocation bool                                  `json:"suppress_bundle_relocation,omitempty"`
	ForceInstallAfterDate    *time.Time                            `json:"force_install_after_date,omitempty"`
	InstalledSize            int64                                 `json:"installed_size,omitempty"                                       validate:"gte=0"                                                                         minimum:"0"`
	PackagePath              string                                `json:"package_path,omitempty"`
	InstallerChoicesXML      []PackageInstallerChoice              `json:"installer_choices_xml,omitempty"                                validate:"dive"`
	InstallerEnvironment     []PackageInstallerEnvironmentVariable `json:"installer_environment,omitempty"                                validate:"dive"`
	Installs                 []PackageInstallItem                  `json:"installs,omitempty"                                             validate:"dive"`
	Receipts                 []PackageReceipt                      `json:"receipts,omitempty"                                             validate:"dive"`
	ItemsToCopy              []PackageItemToCopy                   `json:"items_to_copy,omitempty"                                        validate:"dive"`
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
	InstallerObjectID        *int64                                `json:"installer_object_id,omitempty"                                  validate:"omitempty,gt=0"                                                                minimum:"1"`
}

// PackageCreateMutation creates one package version under selected Munki software.
type PackageCreateMutation struct {
	PackageMutation

	SoftwareID int64 `json:"software_id" minimum:"1" validate:"gt=0"`
}

// InstallerFile is the Munki-facing view of a package-owned installer object.
type InstallerFile struct {
	Filename              string `json:"filename"`
	InstallerItemLocation string `json:"installer_item_location"`
	SizeBytes             int64  `json:"size_bytes"`
	SHA256                string `json:"sha256"`
}

// PackageSoftware is the parent software attached to a package version.
type PackageSoftware struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	DisplayName  *string `json:"display_name,omitempty"`
	Description  string  `json:"description"`
	Category     string  `json:"category"`
	Developer    string  `json:"developer"`
	IconObjectID *int64  `json:"-"`
	IconURL      string  `json:"icon_url,omitempty"`
}

// Package is one Woodstar-authored Munki package version available for targeting.
type Package struct {
	ID                       int64                                 `json:"id"`
	Software                 PackageSoftware                       `json:"software"`
	Version                  string                                `json:"version"`
	InstallerType            InstallerType                         `json:"installer_type"`
	UnattendedInstall        bool                                  `json:"unattended_install"`
	UnattendedUninstall      bool                                  `json:"unattended_uninstall"`
	Uninstallable            bool                                  `json:"uninstallable"`
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
	CreatedAt                time.Time                             `json:"created_at"`
	UpdatedAt                time.Time                             `json:"updated_at"`
}

type PackageListParams struct {
	dbutil.ListParams

	InstallerTypes []string `validate:"dive,oneof=pkg nopkg copy_from_dmg"`
	SoftwareID     int64    `validate:"gte=0"`
}

func (p *PackageListParams) normalize() {
	p.ListParams = dbutil.NormalizeListParams(p.ListParams)
	p.InstallerTypes = dbutil.NormalizeListValues(p.InstallerTypes)
}

func (p *PackageListParams) validate() error {
	if err := validation.Struct(p); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func (m *PackageMutation) validate() error {
	if err := validation.Struct(m); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return m.validateRelations()
}

func (m PackageCreateMutation) validate() error {
	if err := validation.Struct(m); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return m.validateRelations()
}

func (m *PackageMutation) validateRelations() error {
	hasInstaller := m.InstallerObjectID != nil
	switch m.InstallerType {
	case InstallerTypeNoPkg:
		if hasInstaller {
			return fmt.Errorf("%w: nopkg must not reference installer_object_id", dbutil.ErrInvalidInput)
		}
	case InstallerTypePkg, InstallerTypeCopyFromDMG:
		if !hasInstaller {
			return fmt.Errorf("%w: %s requires installer_object_id", dbutil.ErrInvalidInput, m.InstallerType)
		}
	}
	if m.InstallerType == InstallerTypeCopyFromDMG && len(m.ItemsToCopy) == 0 {
		return fmt.Errorf("%w: copy_from_dmg requires items_to_copy entries", dbutil.ErrInvalidInput)
	}
	if m.Uninstallable {
		if m.UninstallMethod == "" {
			return fmt.Errorf("%w: uninstallable requires uninstall_method", dbutil.ErrInvalidInput)
		}
		switch m.UninstallMethod {
		case UninstallMethodRemovePackages:
			if len(m.Receipts) == 0 {
				return fmt.Errorf("%w: removepackages requires receipts", dbutil.ErrInvalidInput)
			}
		case UninstallMethodRemoveCopiedItems:
			if len(m.ItemsToCopy) == 0 {
				return fmt.Errorf("%w: remove_copied_items requires items_to_copy entries", dbutil.ErrInvalidInput)
			}
		case UninstallMethodUninstallScript:
			if strings.TrimSpace(m.UninstallScript) == "" {
				return fmt.Errorf("%w: uninstall_script requires uninstall_script", dbutil.ErrInvalidInput)
			}
		}
	}

	environmentNames := make(map[string]struct{}, len(m.InstallerEnvironment))
	for _, variable := range m.InstallerEnvironment {
		name := strings.TrimSpace(variable.Name)
		if _, exists := environmentNames[name]; exists {
			return fmt.Errorf(
				"%w: installer_environment contains duplicate name %q",
				dbutil.ErrInvalidInput,
				name,
			)
		}
		environmentNames[name] = struct{}{}
	}
	if m.BlockingApplicationsNone && len(m.BlockingApplications) > 0 {
		return fmt.Errorf(
			"%w: blocking_applications_none cannot be set with blocking_applications entries",
			dbutil.ErrInvalidInput,
		)
	}
	return nil
}

func (m *PackageMutation) normalize() {
	m.Version = strings.TrimSpace(m.Version)
	m.InstallerType = InstallerType(strings.TrimSpace(string(m.InstallerType)))
	m.UninstallMethod = UninstallMethod(strings.TrimSpace(string(m.UninstallMethod)))
	m.RestartAction = RestartAction(strings.TrimSpace(string(m.RestartAction)))
	m.MinimumMunkiVersion = strings.TrimSpace(m.MinimumMunkiVersion)
	m.MinimumOSVersion = strings.TrimSpace(m.MinimumOSVersion)
	m.MaximumOSVersion = strings.TrimSpace(m.MaximumOSVersion)
	m.PackagePath = strings.TrimSpace(m.PackagePath)
	for i := range m.SupportedArchitectures {
		m.SupportedArchitectures[i] = strings.TrimSpace(m.SupportedArchitectures[i])
	}
	for i := range m.BlockingApplications {
		m.BlockingApplications[i] = strings.TrimSpace(m.BlockingApplications[i])
	}
	for i := range m.InstallerChoicesXML {
		m.InstallerChoicesXML[i].ChoiceIdentifier = strings.TrimSpace(m.InstallerChoicesXML[i].ChoiceIdentifier)
	}
	for i := range m.InstallerEnvironment {
		m.InstallerEnvironment[i].Name = strings.TrimSpace(m.InstallerEnvironment[i].Name)
	}
	for i := range m.Installs {
		m.Installs[i].Type = PackageInstallItemType(strings.TrimSpace(string(m.Installs[i].Type)))
		m.Installs[i].Path = strings.TrimSpace(m.Installs[i].Path)
	}
	for i := range m.Receipts {
		m.Receipts[i].PackageID = strings.TrimSpace(m.Receipts[i].PackageID)
	}
	for i := range m.ItemsToCopy {
		m.ItemsToCopy[i].SourceItem = strings.TrimSpace(m.ItemsToCopy[i].SourceItem)
		m.ItemsToCopy[i].DestinationPath = strings.TrimSpace(m.ItemsToCopy[i].DestinationPath)
	}
}
