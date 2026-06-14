package packages

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/storage"
)

var timeType = reflect.TypeFor[time.Time]()

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
func Pkginfo(pkg Package, objects PkginfoObjects) map[string]any {
	return munkiPkginfoFromPackage(pkg, objects).Map()
}

// PkginfoObjects are storage objects needed only while rendering Munki pkginfo.
type PkginfoObjects struct {
	Installer   *storage.Object
	Uninstaller *storage.Object
	Icon        *storage.Object
}

type munkiPkginfo struct {
	Name                     string                    `json:"name"`
	Version                  string                    `json:"version"`
	DisplayName              string                    `json:"display_name,omitempty"`
	Description              string                    `json:"description,omitempty"`
	Category                 string                    `json:"category,omitempty"`
	Developer                string                    `json:"developer,omitempty"`
	InstallerType            InstallerType             `json:"installer_type,omitempty"`
	UninstallMethod          UninstallMethod           `json:"uninstall_method,omitempty"`
	RestartAction            RestartAction             `json:"RestartAction,omitempty"`
	MinimumMunkiVersion      string                    `json:"minimum_munki_version,omitempty"`
	MinimumOSVersion         string                    `json:"minimum_os_version,omitempty"`
	MaximumOSVersion         string                    `json:"maximum_os_version,omitempty"`
	SupportedArchitectures   []string                  `json:"supported_architectures,omitempty"`
	BlockingApplications     []string                  `json:"blocking_applications,omitempty"`
	InstallableCondition     string                    `json:"installable_condition,omitempty"`
	BlockingAppsManualQuit   bool                      `json:"blocking_applications_manual_quit_only,omitempty"`
	BlockingAppsQuitScript   string                    `json:"blocking_applications_quit_script,omitempty"`
	Requires                 []string                  `json:"requires,omitempty"`
	UpdateFor                []string                  `json:"update_for,omitempty"`
	UnattendedInstall        bool                      `json:"unattended_install,omitempty"`
	UnattendedUninstall      bool                      `json:"unattended_uninstall,omitempty"`
	Uninstallable            bool                      `json:"uninstallable,omitempty"`
	OnDemand                 bool                      `json:"OnDemand,omitempty"`
	Precache                 bool                      `json:"precache,omitempty"`
	Autoremove               bool                      `json:"autoremove,omitempty"`
	AppleItem                bool                      `json:"apple_item,omitempty"`
	SuppressBundleRelocation bool                      `json:"suppress_bundle_relocation,omitempty"`
	ForceInstallAfterDate    *time.Time                `json:"force_install_after_date,omitempty"`
	InstalledSize            int64                     `json:"installed_size,omitempty"`
	InstallerItemLocation    string                    `json:"installer_item_location,omitempty"`
	InstallerItemHash        string                    `json:"installer_item_hash,omitempty"`
	InstallerItemSize        int64                     `json:"installer_item_size,omitempty"`
	PackagePath              string                    `json:"package_path,omitempty"`
	InstallerChoicesXML      []munkiPkginfoChoice      `json:"installer_choices_xml,omitempty"`
	InstallerEnvironment     map[string]string         `json:"installer_environment,omitempty"`
	Installs                 []munkiPkginfoInstallItem `json:"installs,omitempty"`
	Receipts                 []munkiPkginfoReceipt     `json:"receipts,omitempty"`
	ItemsToCopy              []munkiPkginfoItemToCopy  `json:"items_to_copy,omitempty"`
	Notes                    string                    `json:"notes,omitempty"`
	InstallcheckScript       string                    `json:"installcheck_script,omitempty"`
	UninstallcheckScript     string                    `json:"uninstallcheck_script,omitempty"`
	PreinstallScript         string                    `json:"preinstall_script,omitempty"`
	PostinstallScript        string                    `json:"postinstall_script,omitempty"`
	PreuninstallScript       string                    `json:"preuninstall_script,omitempty"`
	PostuninstallScript      string                    `json:"postuninstall_script,omitempty"`
	UninstallerItemLocation  string                    `json:"uninstaller_item_location,omitempty"`
	UninstallScript          string                    `json:"uninstall_script,omitempty"`
	VersionScript            string                    `json:"version_script,omitempty"`
	PreinstallAlert          *munkiPkginfoAlert        `json:"preinstall_alert,omitempty"`
	PreuninstallAlert        *munkiPkginfoAlert        `json:"preuninstall_alert,omitempty"`
	IconName                 string                    `json:"icon_name,omitempty"`
	IconHash                 string                    `json:"icon_hash,omitempty"`
}

type munkiPkginfoInstallItem struct {
	Type                  PackageInstallItemType `json:"type,omitempty"`
	Path                  string                 `json:"path"`
	BundleIdentifier      string                 `json:"CFBundleIdentifier,omitempty"`
	BundleName            string                 `json:"CFBundleName,omitempty"`
	BundleShortVersion    string                 `json:"CFBundleShortVersionString,omitempty"`
	BundleVersion         string                 `json:"CFBundleVersion,omitempty"`
	VersionComparisonKey  string                 `json:"version_comparison_key,omitempty"`
	MD5Checksum           string                 `json:"md5checksum,omitempty"`
	MinimumOSVersion      string                 `json:"minimum_os_version,omitempty"`
	InstallerItemLocation string                 `json:"installer_item_location,omitempty"`
}

type munkiPkginfoReceipt struct {
	PackageID string `json:"packageid"`
	Version   string `json:"version,omitempty"`
	Optional  bool   `json:"optional,omitempty"`
}

type munkiPkginfoItemToCopy struct {
	SourceItem      string `json:"source_item"`
	DestinationPath string `json:"destination_path"`
	DestinationItem string `json:"destination_item,omitempty"`
	User            string `json:"user,omitempty"`
	Group           string `json:"group,omitempty"`
	Mode            string `json:"mode,omitempty"`
}

type munkiPkginfoChoice struct {
	ChoiceIdentifier string `json:"choiceIdentifier,omitempty"`
	ChoiceAttribute  string `json:"choiceAttribute,omitempty"`
	AttributeSetting int32  `json:"attributeSetting"`
}

type munkiPkginfoAlert struct {
	Title       string `json:"alert_title,omitempty"`
	Detail      string `json:"alert_detail,omitempty"`
	OKLabel     string `json:"ok_label,omitempty"`
	CancelLabel string `json:"cancel_label,omitempty"`
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
		BlockingApplications:     pkg.BlockingApplications,
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
	if pkg.UninstallMethod == UninstallMethodUninstallPackage && objects.Uninstaller != nil {
		item.UninstallerItemLocation = UninstallerItemLocation(pkg, *objects.Uninstaller)
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

// UninstallerItemLocation returns the Munki repository path for a package uninstaller.
func UninstallerItemLocation(pkg Package, obj storage.Object) string {
	return packageObjectLocation(pkg.ID, "uninstaller", obj)
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

func (item munkiPkginfo) Map() map[string]any {
	return munkiStructMap(reflect.ValueOf(item))
}

func (alert munkiPkginfoAlert) Empty() bool {
	return alert.Title == "" &&
		alert.Detail == "" &&
		alert.OKLabel == "" &&
		alert.CancelLabel == ""
}

func munkiStructMap(value reflect.Value) map[string]any {
	value = reflect.Indirect(value)
	valueType := value.Type()
	out := make(map[string]any, value.NumField())
	for i := range value.NumField() {
		field := valueType.Field(i)
		jsonName, omitEmpty := jsonField(field)
		if jsonName == "" {
			continue
		}
		fieldValue := value.Field(i)
		if omitEmpty && fieldValue.IsZero() {
			continue
		}
		out[jsonName] = munkiValue(fieldValue)
	}
	return out
}

func jsonField(field reflect.StructField) (string, bool) {
	name, options, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "" || name == "-" {
		return "", false
	}
	if slices.Contains(strings.Split(options, ","), "omitempty") {
		return name, true
	}
	return name, false
}

func munkiValue(value reflect.Value) any {
	value = reflect.Indirect(value)
	if !value.IsValid() {
		return nil
	}
	if value.Type() == timeType {
		return value.Interface()
	}
	switch value.Kind() {
	case reflect.Struct:
		return munkiStructMap(value)
	case reflect.Slice, reflect.Array:
		return munkiSlice(value)
	default:
		return value.Interface()
	}
}

func munkiSlice(value reflect.Value) any {
	if value.Len() == 0 || value.Type().Elem().Kind() != reflect.Struct {
		return value.Interface()
	}
	out := make([]map[string]any, 0, value.Len())
	for i := range value.Len() {
		out = append(out, munkiStructMap(value.Index(i)))
	}
	return out
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
