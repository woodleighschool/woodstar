package packages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

var timeType = reflect.TypeFor[time.Time]()

// MunkiName returns the stable internal pkginfo name Woodstar gives Munki.
func MunkiName(packageID int64) string {
	if packageID <= 0 {
		return ""
	}
	return strconv.FormatInt(packageID, 10)
}

func Pkginfo(pkg Package, softwareIcon IconRef) map[string]any {
	return munkiPkginfoFromPackage(pkg, softwareIcon).Map()
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
	BlockingApplications     []string                  `json:"blocking_applications"`
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
	PackagePath              string                    `json:"package_path,omitempty"`
	InstallerChoicesXML      string                    `json:"installer_choices_xml,omitempty"`
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
	PackageID string `json:"packageid"` //nolint:misspell
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

type munkiPkginfoAlert struct {
	Title       string `json:"alert_title,omitempty"`
	Detail      string `json:"alert_detail,omitempty"`
	OKLabel     string `json:"ok_label,omitempty"`
	CancelLabel string `json:"cancel_label,omitempty"`
}

func munkiPkginfoFromPackage(pkg Package, softwareIcon IconRef) munkiPkginfo {
	item := munkiPkginfo{
		Name:                     MunkiName(pkg.ID),
		Version:                  pkg.Version,
		DisplayName:              pkg.SoftwareName,
		Description:              pkg.SoftwareDescription,
		Category:                 pkg.SoftwareCategory,
		Developer:                pkg.SoftwareDeveloper,
		MinimumMunkiVersion:      pkg.MinimumMunkiVersion,
		MinimumOSVersion:         pkg.MinimumOSVersion,
		MaximumOSVersion:         pkg.MaximumOSVersion,
		SupportedArchitectures:   pkg.SupportedArchitectures,
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
		InstallerChoicesXML:      pkg.InstallerChoicesXML,
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
		IconName:                 softwareIcon.Name,
		IconHash:                 softwareIcon.Hash,
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

func (item munkiPkginfo) Map() map[string]any {
	return munkiStructMap(reflect.ValueOf(item))
}

func (item munkiPkginfo) PackageMutation() (PackageMutation, error) {
	mutation := PackageMutation{
		Version:                  item.Version,
		InstallerType:            item.InstallerType,
		UninstallMethod:          item.UninstallMethod,
		RestartAction:            item.RestartAction,
		MinimumMunkiVersion:      item.MinimumMunkiVersion,
		MinimumOSVersion:         item.MinimumOSVersion,
		MaximumOSVersion:         item.MaximumOSVersion,
		SupportedArchitectures:   item.SupportedArchitectures,
		BlockingApplications:     item.BlockingApplications,
		InstallableCondition:     item.InstallableCondition,
		BlockingAppsManualQuit:   item.BlockingAppsManualQuit,
		BlockingAppsQuitScript:   item.BlockingAppsQuitScript,
		UnattendedInstall:        item.UnattendedInstall,
		UnattendedUninstall:      item.UnattendedUninstall,
		OnDemand:                 item.OnDemand,
		Precache:                 item.Precache,
		Autoremove:               item.Autoremove,
		AppleItem:                item.AppleItem,
		SuppressBundleRelocation: item.SuppressBundleRelocation,
		ForceInstallAfterDate:    item.ForceInstallAfterDate,
		InstalledSize:            item.InstalledSize,
		PackagePath:              item.PackagePath,
		InstallerChoicesXML:      item.InstallerChoicesXML,
		InstallerEnvironment:     item.PackageInstallerEnvironment(),
		Installs:                 item.PackageInstallItems(),
		Receipts:                 item.PackageReceipts(),
		ItemsToCopy:              item.PackageItemsToCopy(),
		Notes:                    item.Notes,
		InstallcheckScript:       item.InstallcheckScript,
		UninstallcheckScript:     item.UninstallcheckScript,
		PreinstallScript:         item.PreinstallScript,
		PostinstallScript:        item.PostinstallScript,
		PreuninstallScript:       item.PreuninstallScript,
		PostuninstallScript:      item.PostuninstallScript,
		UninstallScript:          item.UninstallScript,
		VersionScript:            item.VersionScript,
		PreinstallAlert:          item.PreinstallAlert.PackageAlert(),
		PreuninstallAlert:        item.PreuninstallAlert.PackageAlert(),
	}
	if mutation.Version == "" {
		return PackageMutation{}, fmt.Errorf("%w: pkginfo version is required", dbutil.ErrInvalidInput)
	}
	var err error
	if mutation.Requires, err = packageReferences(item.Requires, "requires"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.UpdateFor, err = packageReferences(item.UpdateFor, "update_for"); err != nil {
		return PackageMutation{}, err
	}
	return mutation, nil
}

func (item munkiPkginfo) PackageInstallerEnvironment() []PackageInstallerEnvironmentVariable {
	if len(item.InstallerEnvironment) == 0 {
		return nil
	}
	names := make([]string, 0, len(item.InstallerEnvironment))
	for name := range item.InstallerEnvironment {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]PackageInstallerEnvironmentVariable, 0, len(names))
	for _, name := range names {
		out = append(out, PackageInstallerEnvironmentVariable{Name: name, Value: item.InstallerEnvironment[name]})
	}
	return out
}

func (item munkiPkginfo) PackageInstallItems() []PackageInstallItem {
	out := make([]PackageInstallItem, 0, len(item.Installs))
	for _, value := range item.Installs {
		itemType := value.Type
		if itemType == "" {
			itemType = PackageInstallItemFile
		}
		out = append(out, PackageInstallItem{
			Type:                  itemType,
			Path:                  value.Path,
			BundleIdentifier:      value.BundleIdentifier,
			BundleName:            value.BundleName,
			BundleShortVersion:    value.BundleShortVersion,
			BundleVersion:         value.BundleVersion,
			VersionComparisonKey:  value.VersionComparisonKey,
			MD5Checksum:           value.MD5Checksum,
			MinimumOSVersion:      value.MinimumOSVersion,
			InstallerItemLocation: value.InstallerItemLocation,
		})
	}
	return out
}

func (item munkiPkginfo) PackageReceipts() []PackageReceipt {
	out := make([]PackageReceipt, 0, len(item.Receipts))
	for _, value := range item.Receipts {
		out = append(out, PackageReceipt(value))
	}
	return out
}

func (item munkiPkginfo) PackageItemsToCopy() []PackageItemToCopy {
	out := make([]PackageItemToCopy, 0, len(item.ItemsToCopy))
	for _, value := range item.ItemsToCopy {
		out = append(out, PackageItemToCopy(value))
	}
	return out
}

func (alert *munkiPkginfoAlert) PackageAlert() PackageAlert {
	if alert == nil || alert.Empty() {
		return PackageAlert{}
	}
	return PackageAlert{
		Enabled:     true,
		Title:       alert.Title,
		Detail:      alert.Detail,
		OKLabel:     alert.OKLabel,
		CancelLabel: alert.CancelLabel,
	}
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
	out := make([]string, 0, len(references))
	for _, ref := range references {
		if name := MunkiName(ref.PackageID); name != "" {
			out = append(out, name)
		}
	}
	return out
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
	out := make([]munkiPkginfoInstallItem, 0, len(values))
	for _, value := range values {
		out = append(out, munkiPkginfoInstallItem(value))
	}
	return out
}

func munkiReceipts(values []PackageReceipt) []munkiPkginfoReceipt {
	out := make([]munkiPkginfoReceipt, 0, len(values))
	for _, value := range values {
		out = append(out, munkiPkginfoReceipt(value))
	}
	return out
}

func munkiItemsToCopy(values []PackageItemToCopy) []munkiPkginfoItemToCopy {
	out := make([]munkiPkginfoItemToCopy, 0, len(values))
	for _, value := range values {
		out = append(out, munkiPkginfoItemToCopy(value))
	}
	return out
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

// Import imports one Munki pkginfo item into selected Woodstar-managed Munki software.
func (s *Store) Import(ctx context.Context, softwareID int64, params PackageImportMutation) (*PackageRecord, error) {
	if softwareID <= 0 {
		return nil, fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	mutation, err := packageMutationFromPkginfo(params.Pkginfo)
	if err != nil {
		return nil, err
	}
	mutation.InstallerArtifactID = params.InstallerArtifactID
	mutation.UninstallerArtifactID = params.UninstallerArtifactID
	mutation.Eligible = true
	if params.Eligible != nil {
		mutation.Eligible = *params.Eligible
	}
	return s.Upsert(ctx, softwareID, mutation)
}

func packageMutationFromPkginfo(raw json.RawMessage) (PackageMutation, error) {
	item, err := decodeMunkiPkginfo(raw)
	if err != nil {
		return PackageMutation{}, err
	}
	return item.PackageMutation()
}

func decodeMunkiPkginfo(raw json.RawMessage) (munkiPkginfo, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var item *munkiPkginfo
	if err := decoder.Decode(&item); err != nil {
		return munkiPkginfo{}, fmt.Errorf("%w: pkginfo has invalid shape: %w", dbutil.ErrInvalidInput, err)
	}
	if item == nil {
		return munkiPkginfo{}, fmt.Errorf("%w: pkginfo must be a JSON object", dbutil.ErrInvalidInput)
	}
	return *item, nil
}

func packageReferences(values []string, key string) ([]PackageReference, error) {
	out := make([]PackageReference, 0, len(values))
	for _, value := range values {
		packageID, err := strconv.ParseInt(value, 10, 64)
		if err != nil || packageID <= 0 {
			return nil, fmt.Errorf(
				"%w: pkginfo %s entries must be Woodstar package IDs",
				dbutil.ErrInvalidInput,
				key,
			)
		}
		out = append(out, PackageReference{PackageID: packageID})
	}
	return out, nil
}
