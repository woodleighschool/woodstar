package packages

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/storage"
)

// MunkiName returns the stable software identity Woodstar gives Munki.
func MunkiName(pkg Package) string {
	return MunkiSoftwareName(pkg.SoftwareID)
}

// MunkiSoftwareName returns the stable pkginfo name for a Woodstar software item.
func MunkiSoftwareName(softwareID int64) string {
	if softwareID <= 0 {
		return ""
	}
	return strconv.FormatInt(softwareID, 10)
}

// MunkiVersionedName returns Munki's manifest syntax for a specific package version.
func MunkiVersionedName(pkg Package) string {
	return MunkiVersionedSoftwareName(pkg.SoftwareID, pkg.Version)
}

// MunkiVersionedSoftwareName returns Munki's name--version syntax for a specific package version.
func MunkiVersionedSoftwareName(softwareID int64, packageVersion string) string {
	name := MunkiSoftwareName(softwareID)
	version := strings.TrimSpace(packageVersion)
	if name == "" || version == "" {
		return name
	}
	return name + "--" + version
}

// Pkginfo renders the Munki pkginfo shape for a Woodstar package.
func Pkginfo(pkg Package, objects PkginfoObjects) any {
	return munkiPkginfoFromPackage(pkg, objects)
}

// PkginfoObjects are storage objects needed only while rendering Munki pkginfo.
type PkginfoObjects struct {
	Installer *storage.Object
	Icon      *storage.Object
}

type munkiPkginfo struct {
	Name                     string                    `plist:"name"`
	Version                  string                    `plist:"version"`
	DisplayName              string                    `plist:"display_name,omitempty"`
	Description              string                    `plist:"description,omitempty"`
	Category                 string                    `plist:"category,omitempty"`
	Developer                string                    `plist:"developer,omitempty"`
	InstallerType            InstallerType             `plist:"installer_type,omitempty"`
	UninstallMethod          UninstallMethod           `plist:"uninstall_method,omitempty"`
	RestartAction            RestartAction             `plist:"RestartAction,omitempty"`
	MinimumMunkiVersion      string                    `plist:"minimum_munki_version,omitempty"`
	MinimumOSVersion         string                    `plist:"minimum_os_version,omitempty"`
	MaximumOSVersion         string                    `plist:"maximum_os_version,omitempty"`
	SupportedArchitectures   []string                  `plist:"supported_architectures,omitempty"`
	BlockingApplications     *[]string                 `plist:"blocking_applications,omitempty"`
	InstallableCondition     string                    `plist:"installable_condition,omitempty"`
	BlockingAppsManualQuit   bool                      `plist:"blocking_applications_manual_quit_only,omitempty"`
	BlockingAppsQuitScript   string                    `plist:"blocking_applications_quit_script,omitempty"`
	Requires                 []string                  `plist:"requires,omitempty"`
	UpdateFor                []string                  `plist:"update_for,omitempty"`
	UnattendedInstall        bool                      `plist:"unattended_install,omitempty"`
	UnattendedUninstall      bool                      `plist:"unattended_uninstall,omitempty"`
	Uninstallable            bool                      `plist:"uninstallable,omitempty"`
	OnDemand                 bool                      `plist:"OnDemand,omitempty"`
	Precache                 bool                      `plist:"precache,omitempty"`
	Autoremove               bool                      `plist:"autoremove,omitempty"`
	AppleItem                bool                      `plist:"apple_item,omitempty"`
	SuppressBundleRelocation bool                      `plist:"suppress_bundle_relocation,omitempty"`
	ForceInstallAfterDate    *time.Time                `plist:"force_install_after_date,omitempty"`
	InstalledSize            int64                     `plist:"installed_size,omitempty"`
	InstallerItemLocation    string                    `plist:"installer_item_location,omitempty"`
	InstallerItemHash        string                    `plist:"installer_item_hash,omitempty"`
	InstallerItemSize        int64                     `plist:"installer_item_size,omitempty"`
	PackagePath              string                    `plist:"package_path,omitempty"`
	InstallerChoicesXML      []munkiPkginfoChoice      `plist:"installer_choices_xml,omitempty"`
	InstallerEnvironment     map[string]string         `plist:"installer_environment,omitempty"`
	Installs                 []munkiPkginfoInstallItem `plist:"installs,omitempty"`
	Receipts                 []munkiPkginfoReceipt     `plist:"receipts,omitempty"`
	ItemsToCopy              []munkiPkginfoItemToCopy  `plist:"items_to_copy,omitempty"`
	Notes                    string                    `plist:"notes,omitempty"`
	InstallcheckScript       string                    `plist:"installcheck_script,omitempty"`
	UninstallcheckScript     string                    `plist:"uninstallcheck_script,omitempty"`
	PreinstallScript         string                    `plist:"preinstall_script,omitempty"`
	PostinstallScript        string                    `plist:"postinstall_script,omitempty"`
	PreuninstallScript       string                    `plist:"preuninstall_script,omitempty"`
	PostuninstallScript      string                    `plist:"postuninstall_script,omitempty"`
	UninstallScript          string                    `plist:"uninstall_script,omitempty"`
	VersionScript            string                    `plist:"version_script,omitempty"`
	PreinstallAlert          *munkiPkginfoAlert        `plist:"preinstall_alert,omitempty"`
	PreuninstallAlert        *munkiPkginfoAlert        `plist:"preuninstall_alert,omitempty"`
	IconName                 string                    `plist:"icon_name,omitempty"`
	IconHash                 string                    `plist:"icon_hash,omitempty"`
}

type munkiPkginfoInstallItem struct {
	Type                  PackageInstallItemType `plist:"type,omitempty"`
	Path                  string                 `plist:"path"`
	BundleIdentifier      string                 `plist:"CFBundleIdentifier,omitempty"`
	BundleName            string                 `plist:"CFBundleName,omitempty"`
	BundleShortVersion    string                 `plist:"CFBundleShortVersionString,omitempty"`
	BundleVersion         string                 `plist:"CFBundleVersion,omitempty"`
	VersionComparisonKey  string                 `plist:"version_comparison_key,omitempty"`
	MD5Checksum           string                 `plist:"md5checksum,omitempty"`
	MinimumOSVersion      string                 `plist:"minimum_os_version,omitempty"`
	InstallerItemLocation string                 `plist:"installer_item_location,omitempty"`
}

type munkiPkginfoReceipt struct {
	PackageID string `plist:"packageid"`
	Version   string `plist:"version,omitempty"`
	Optional  bool   `plist:"optional,omitempty"`
}

type munkiPkginfoItemToCopy struct {
	SourceItem      string `plist:"source_item"`
	DestinationPath string `plist:"destination_path"`
	DestinationItem string `plist:"destination_item,omitempty"`
	User            string `plist:"user,omitempty"`
	Group           string `plist:"group,omitempty"`
	Mode            string `plist:"mode,omitempty"`
}

type munkiPkginfoChoice struct {
	ChoiceIdentifier string `plist:"choiceIdentifier,omitempty"`
	ChoiceAttribute  string `plist:"choiceAttribute,omitempty"`
	AttributeSetting int32  `plist:"attributeSetting"`
}

type munkiPkginfoAlert struct {
	Title       string `plist:"alert_title,omitempty"`
	Detail      string `plist:"alert_detail,omitempty"`
	OKLabel     string `plist:"ok_label,omitempty"`
	CancelLabel string `plist:"cancel_label,omitempty"`
}

func munkiPkginfoFromPackage(pkg Package, objects PkginfoObjects) munkiPkginfo {
	item := munkiPkginfo{
		Name:                     MunkiName(pkg),
		Version:                  pkg.Version,
		DisplayName:              pkg.SoftwareName,
		Description:              pkg.SoftwareDescription,
		Category:                 pkg.SoftwareCategory,
		Developer:                pkg.SoftwareDeveloper,
		MinimumMunkiVersion:      pkg.MinimumMunkiVersion,
		MinimumOSVersion:         pkg.MinimumOSVersion,
		MaximumOSVersion:         pkg.MaximumOSVersion,
		SupportedArchitectures:   nonEmptyStrings(pkg.SupportedArchitectures),
		BlockingApplications:     munkiBlockingApplications(pkg.BlockingApplications),
		InstallableCondition:     pkg.InstallableCondition,
		BlockingAppsManualQuit:   pkg.BlockingAppsManualQuit,
		BlockingAppsQuitScript:   pkg.BlockingAppsQuitScript,
		Requires:                 munkiReferenceNames(pkg.Requires),
		UpdateFor:                munkiReferenceNames(pkg.UpdateFor),
		UnattendedInstall:        pkg.UnattendedInstall,
		UnattendedUninstall:      pkg.UnattendedUninstall,
		OnDemand:                 pkg.OnDemand,
		Precache:                 pkg.Precache,
		Autoremove:               pkg.Autoremove,
		AppleItem:                pkg.AppleItem,
		SuppressBundleRelocation: pkg.SuppressBundleRelocation,
		ForceInstallAfterDate:    pkg.ForceInstallAfterDate,
		InstalledSize:            pkg.InstalledSize,
		PackagePath:              pkg.PackagePath,
		InstallerChoicesXML:      munkiInstallerChoices(pkg.InstallerChoicesXML),
		InstallerEnvironment:     munkiInstallerEnvironment(pkg.InstallerEnvironment),
		Installs:                 munkiInstallItems(pkg.Installs),
		Receipts:                 munkiReceipts(pkg.Receipts),
		ItemsToCopy:              munkiItemsToCopy(pkg.ItemsToCopy),
		Notes:                    pkg.Notes,
		InstallcheckScript:       pkg.InstallcheckScript,
		UninstallcheckScript:     pkg.UninstallcheckScript,
		PreinstallScript:         pkg.PreinstallScript,
		PostinstallScript:        pkg.PostinstallScript,
		PreuninstallScript:       pkg.PreuninstallScript,
		PostuninstallScript:      pkg.PostuninstallScript,
		UninstallScript:          pkg.UninstallScript,
		VersionScript:            pkg.VersionScript,
		PreinstallAlert:          munkiAlert(pkg.PreinstallAlert),
		PreuninstallAlert:        munkiAlert(pkg.PreuninstallAlert),
	}
	if pkg.InstallerType != InstallerTypeNoPkg && objects.Installer != nil {
		item.InstallerItemLocation = InstallerItemLocation(pkg, *objects.Installer)
		item.InstallerItemHash = objectHash(*objects.Installer)
		item.InstallerItemSize = objectSizeKB(*objects.Installer)
	}
	if objects.Icon != nil {
		item.IconName = IconName(*objects.Icon)
		item.IconHash = objectHash(*objects.Icon)
	}
	if pkg.InstallerType != "" && pkg.InstallerType != InstallerTypePkg {
		item.InstallerType = pkg.InstallerType
	}
	if pkg.UninstallMethod != "" && pkg.UninstallMethod != UninstallMethodNone {
		item.UninstallMethod = pkg.UninstallMethod
		item.Uninstallable = true
	}
	if pkg.RestartAction != "" && pkg.RestartAction != RestartActionNone {
		item.RestartAction = pkg.RestartAction
	}
	return item
}

// InstallerItemLocation returns the Munki repository path for a package installer.
func InstallerItemLocation(pkg Package, obj storage.Object) string {
	return packageObjectLocation(pkg.ID, "installer", obj)
}

// IconName returns the Munki icon filename for a storage object.
func IconName(obj storage.Object) string {
	if obj.ID <= 0 || obj.Filename == "" {
		return ""
	}
	return fmt.Sprintf("%d-%s", obj.ID, obj.Filename)
}

func packageObjectLocation(packageID int64, role string, obj storage.Object) string {
	if packageID <= 0 || obj.ID <= 0 || obj.Filename == "" {
		return ""
	}
	return fmt.Sprintf("packages/%d/%s/%s", packageID, role, obj.Filename)
}

func objectHash(obj storage.Object) string {
	if obj.SHA256 == nil {
		return ""
	}
	return *obj.SHA256
}

func objectSizeKB(obj storage.Object) int64 {
	if obj.SizeBytes == nil || *obj.SizeBytes <= 0 {
		return 0
	}
	return (*obj.SizeBytes + 1023) / 1024
}

func (alert munkiPkginfoAlert) Empty() bool {
	return alert.Title == "" &&
		alert.Detail == "" &&
		alert.OKLabel == "" &&
		alert.CancelLabel == ""
}

func munkiReferenceNames(references []PackageReference) []string {
	if len(references) == 0 {
		return nil
	}
	out := make([]string, 0, len(references))
	for _, ref := range references {
		if name := munkiReferenceName(ref); name != "" {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func munkiReferenceName(ref PackageReference) string {
	return MunkiVersionedSoftwareName(ref.SoftwareID, ref.PackageVersion)
}

func munkiBlockingApplications(values []string) *[]string {
	if values == nil {
		return nil
	}
	return &values
}

func munkiInstallerEnvironment(values []PackageInstallerEnvironmentVariable) map[string]string {
	if len(values) == 0 {
		return nil
	}
	environment := make(map[string]string, len(values))
	for _, value := range values {
		environment[value.Name] = value.Value
	}
	return environment
}

func munkiInstallItems(values []PackageInstallItem) []munkiPkginfoInstallItem {
	if len(values) == 0 {
		return nil
	}
	out := make([]munkiPkginfoInstallItem, 0, len(values))
	for _, value := range values {
		out = append(out, munkiPkginfoInstallItem(value))
	}
	return out
}

func munkiReceipts(values []PackageReceipt) []munkiPkginfoReceipt {
	if len(values) == 0 {
		return nil
	}
	out := make([]munkiPkginfoReceipt, 0, len(values))
	for _, value := range values {
		out = append(out, munkiPkginfoReceipt(value))
	}
	return out
}

func munkiItemsToCopy(values []PackageItemToCopy) []munkiPkginfoItemToCopy {
	if len(values) == 0 {
		return nil
	}
	out := make([]munkiPkginfoItemToCopy, 0, len(values))
	for _, value := range values {
		out = append(out, munkiPkginfoItemToCopy(value))
	}
	return out
}

func munkiInstallerChoices(values []PackageInstallerChoice) []munkiPkginfoChoice {
	if len(values) == 0 {
		return nil
	}
	out := make([]munkiPkginfoChoice, 0, len(values))
	for _, value := range values {
		out = append(out, munkiPkginfoChoice(value))
	}
	return out
}

func nonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return values
}

func munkiAlert(alert PackageAlert) *munkiPkginfoAlert {
	if !alert.Enabled {
		return nil
	}
	out := munkiPkginfoAlert{
		Title:       alert.Title,
		Detail:      alert.Detail,
		OKLabel:     alert.OKLabel,
		CancelLabel: alert.CancelLabel,
	}
	if out.Empty() {
		return nil
	}
	return &out
}
