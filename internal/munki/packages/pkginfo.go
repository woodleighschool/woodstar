package packages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const pkginfoReceiptPackageIDKey = "package" + "id"

// MunkiName returns the stable internal pkginfo name Woodstar gives Munki.
func MunkiName(packageID int64) string {
	if packageID <= 0 {
		return ""
	}
	return strconv.FormatInt(packageID, 10)
}

func Pkginfo(pkg Package) map[string]any {
	item := make(map[string]any)
	item["name"] = MunkiName(pkg.ID)
	item["version"] = pkg.Version

	addPkginfoString(item, "display_name", pkg.SoftwareName)
	addPkginfoString(item, "description", pkg.SoftwareDescription)
	addPkginfoString(item, "category", pkg.SoftwareCategory)
	addPkginfoString(item, "developer", pkg.SoftwareDeveloper)
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
	addPkginfoStrings(item, "requires", referenceNames(pkg.Requires))
	addPkginfoStrings(item, "update_for", referenceNames(pkg.UpdateFor))
	addPkginfoBool(item, "unattended_install", pkg.UnattendedInstall)
	addPkginfoBool(item, "unattended_uninstall", pkg.UnattendedUninstall)
	addPkginfoBool(item, "uninstallable", pkg.UninstallMethod != "" && pkg.UninstallMethod != UninstallMethodNone)
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

func referenceNames(references []PackageReference) []string {
	out := make([]string, 0, len(references))
	for _, ref := range references {
		name := referenceName(ref)
		if name != "" {
			out = append(out, name)
		}
	}
	return cleanStringList(out)
}

func referenceName(ref PackageReference) string {
	return MunkiName(ref.PackageID)
}

// Import imports one Munki pkginfo item into selected Woodstar-managed Munki software.
func (s *Store) Import(ctx context.Context, params PackageImportMutation) (*Package, error) {
	params = cleanImportMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	mutation, err := packageMutationFromPkginfo(params.Pkginfo)
	if err != nil {
		return nil, err
	}
	mutation.SoftwareID = params.SoftwareID
	mutation.InstallerArtifactID = params.InstallerArtifactID
	mutation.UninstallerArtifactID = params.UninstallerArtifactID
	mutation.IconArtifactID = params.IconArtifactID
	mutation.Eligible = true
	if params.Eligible != nil {
		mutation.Eligible = *params.Eligible
	}
	return s.Upsert(ctx, mutation)
}

func packageMutationFromPkginfo(raw json.RawMessage) (PackageMutation, error) {
	item, err := decodePkginfoObject(raw)
	if err != nil {
		return PackageMutation{}, err
	}
	return packageMutationFromPkginfoFields(item)
}

func packageMutationFromPkginfoFields(item map[string]any) (PackageMutation, error) {
	mutation := PackageMutation{
		Version:                  pkginfoString(item, "version"),
		InstallerType:            InstallerType(pkginfoString(item, "installer_type")),
		UninstallMethod:          pkginfoUninstallMethod(item),
		RestartAction:            RestartAction(pkginfoString(item, "RestartAction")),
		MinimumMunkiVersion:      pkginfoString(item, "minimum_munki_version"),
		MinimumOSVersion:         pkginfoString(item, "minimum_os_version"),
		MaximumOSVersion:         pkginfoString(item, "maximum_os_version"),
		IconName:                 pkginfoString(item, "icon_name"),
		IconHash:                 pkginfoString(item, "icon_hash"),
		UnattendedInstall:        pkginfoBool(item, "unattended_install"),
		UnattendedUninstall:      pkginfoBool(item, "unattended_uninstall"),
		OnDemand:                 pkginfoBool(item, "OnDemand"),
		Precache:                 pkginfoBool(item, "precache"),
		Autoremove:               pkginfoBool(item, "autoremove"),
		AppleItem:                pkginfoBool(item, "apple_item"),
		SuppressBundleRelocation: pkginfoBool(item, "suppress_bundle_relocation"),
		InstalledSize:            pkginfoInt64(item, "installed_size"),
		PackagePath:              pkginfoString(item, "package_path"),
		InstallerChoicesXML:      pkginfoString(item, "installer_choices_xml"),
		Notes:                    pkginfoString(item, "notes"),
		InstallcheckScript:       pkginfoString(item, "installcheck_script"),
		UninstallcheckScript:     pkginfoString(item, "uninstallcheck_script"),
		PreinstallScript:         pkginfoString(item, "preinstall_script"),
		PostinstallScript:        pkginfoString(item, "postinstall_script"),
		PreuninstallScript:       pkginfoString(item, "preuninstall_script"),
		PostuninstallScript:      pkginfoString(item, "postuninstall_script"),
		UninstallScript:          pkginfoString(item, "uninstall_script"),
		VersionScript:            pkginfoString(item, "version_script"),
	}
	if mutation.Version == "" {
		return PackageMutation{}, fmt.Errorf("%w: pkginfo version is required", dbutil.ErrInvalidInput)
	}
	var err error
	if mutation.SupportedArchitectures, err = pkginfoStringList(item, "supported_architectures"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.BlockingApplications, err = pkginfoStringList(item, "blocking_applications"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.Requires, err = pkginfoReferences(item, "requires"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.UpdateFor, err = pkginfoReferences(item, "update_for"); err != nil {
		return PackageMutation{}, err
	}
	forceInstallAfterDate, ok, err := pkginfoTime(item, "force_install_after_date")
	if err != nil {
		return PackageMutation{}, err
	}
	if ok {
		mutation.ForceInstallAfterDate = &forceInstallAfterDate
	}
	if mutation.InstallerEnvironment, err = pkginfoInstallerEnvironment(item, "installer_environment"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.Installs, err = pkginfoInstallItems(item, "installs"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.Receipts, err = pkginfoReceipts(item, "receipts"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.ItemsToCopy, err = pkginfoItemsToCopy(item, "items_to_copy"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.PreinstallAlert, err = pkginfoAlert(item, "preinstall_alert"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.PreuninstallAlert, err = pkginfoAlert(item, "preuninstall_alert"); err != nil {
		return PackageMutation{}, err
	}
	return mutation, nil
}

func decodePkginfoObject(raw json.RawMessage) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var item map[string]any
	if err := decoder.Decode(&item); err != nil {
		return nil, fmt.Errorf("%w: pkginfo must be a JSON object", dbutil.ErrInvalidInput)
	}
	if item == nil {
		return nil, fmt.Errorf("%w: pkginfo must be a JSON object", dbutil.ErrInvalidInput)
	}
	return item, nil
}

func cleanImportMutation(params PackageImportMutation) PackageImportMutation {
	params.Pkginfo = bytes.TrimSpace(params.Pkginfo)
	return params
}

func pkginfoString(item map[string]any, key string) string {
	value, ok := item[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func pkginfoBool(item map[string]any, key string) bool {
	value, ok := item[key]
	if !ok || value == nil {
		return false
	}
	boolean, ok := value.(bool)
	return ok && boolean
}

func pkginfoInt64(item map[string]any, key string) int64 {
	value, ok := item[key]
	if !ok || value == nil {
		return 0
	}
	switch value := value.(type) {
	case json.Number:
		number, err := value.Int64()
		if err == nil {
			return number
		}
	case float64:
		return int64(value)
	}
	return 0
}

func pkginfoUninstallMethod(item map[string]any) UninstallMethod {
	method := pkginfoString(item, "uninstall_method")
	switch UninstallMethod(method) {
	case "", UninstallMethodNone:
		return UninstallMethodNone
	case UninstallMethodRemovePackages,
		UninstallMethodRemoveCopiedItems,
		UninstallMethodUninstallScript,
		UninstallMethodUninstallPackage:
		return UninstallMethod(method)
	default:
		return UninstallMethod(method)
	}
}

func pkginfoStringList(item map[string]any, key string) ([]string, error) {
	value, ok := item[key]
	if !ok || value == nil {
		return nil, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: pkginfo %s must be an array of strings", dbutil.ErrInvalidInput, key)
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%w: pkginfo %s must be an array of strings", dbutil.ErrInvalidInput, key)
		}
		out = append(out, strings.TrimSpace(text))
	}
	return cleanStringList(out), nil
}

func pkginfoReferences(item map[string]any, key string) ([]PackageReference, error) {
	values, err := pkginfoStringList(item, key)
	if err != nil {
		return nil, err
	}
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

func pkginfoTime(item map[string]any, key string) (time.Time, bool, error) {
	value := pkginfoString(item, key)
	if value == "" {
		return time.Time{}, false, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false, fmt.Errorf(
			"%w: pkginfo %s must be an RFC3339 timestamp",
			dbutil.ErrInvalidInput,
			key,
		)
	}
	return parsed, true, nil
}

func pkginfoInstallerEnvironment(
	item map[string]any,
	key string,
) ([]PackageInstallerEnvironmentVariable, error) {
	value, ok := item[key]
	if !ok || value == nil {
		return nil, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: pkginfo %s must be an object", dbutil.ErrInvalidInput, key)
	}
	out := make([]PackageInstallerEnvironmentVariable, 0, len(object))
	for name, value := range object {
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%w: pkginfo %s values must be strings", dbutil.ErrInvalidInput, key)
		}
		out = append(out, PackageInstallerEnvironmentVariable{Name: name, Value: text})
	}
	return out, nil
}

func pkginfoInstallItems(item map[string]any, key string) ([]PackageInstallItem, error) {
	values, err := pkginfoObjectList(item, key)
	if err != nil {
		return nil, err
	}
	out := make([]PackageInstallItem, 0, len(values))
	for _, value := range values {
		itemType := PackageInstallItemType(pkginfoMapString(value, "type"))
		if itemType == "" {
			itemType = PackageInstallItemFile
		}
		out = append(out, PackageInstallItem{
			Type:                  itemType,
			Path:                  pkginfoMapString(value, "path"),
			BundleIdentifier:      pkginfoMapString(value, "CFBundleIdentifier"),
			BundleName:            pkginfoMapString(value, "CFBundleName"),
			BundleShortVersion:    pkginfoMapString(value, "CFBundleShortVersionString"),
			BundleVersion:         pkginfoMapString(value, "CFBundleVersion"),
			VersionComparisonKey:  pkginfoMapString(value, "version_comparison_key"),
			MD5Checksum:           pkginfoMapString(value, "md5checksum"),
			MinimumOSVersion:      pkginfoMapString(value, "minimum_os_version"),
			InstallerItemLocation: pkginfoMapString(value, "installer_item_location"),
		})
	}
	return out, nil
}

func pkginfoReceipts(item map[string]any, key string) ([]PackageReceipt, error) {
	values, err := pkginfoObjectList(item, key)
	if err != nil {
		return nil, err
	}
	out := make([]PackageReceipt, 0, len(values))
	for _, value := range values {
		out = append(out, PackageReceipt{
			PackageID: pkginfoMapString(value, pkginfoReceiptPackageIDKey),
			Version:   pkginfoMapString(value, "version"),
			Optional:  pkginfoMapBool(value, "optional"),
		})
	}
	return out, nil
}

func pkginfoItemsToCopy(item map[string]any, key string) ([]PackageItemToCopy, error) {
	values, err := pkginfoObjectList(item, key)
	if err != nil {
		return nil, err
	}
	out := make([]PackageItemToCopy, 0, len(values))
	for _, value := range values {
		out = append(out, PackageItemToCopy{
			SourceItem:      pkginfoMapString(value, "source_item"),
			DestinationPath: pkginfoMapString(value, "destination_path"),
			DestinationItem: pkginfoMapString(value, "destination_item"),
			User:            pkginfoMapString(value, "user"),
			Group:           pkginfoMapString(value, "group"),
			Mode:            pkginfoMapString(value, "mode"),
		})
	}
	return out, nil
}

func pkginfoAlert(item map[string]any, key string) (PackageAlert, error) {
	value, ok := item[key]
	if !ok || value == nil {
		return PackageAlert{}, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return PackageAlert{}, fmt.Errorf("%w: pkginfo %s must be an object", dbutil.ErrInvalidInput, key)
	}
	return PackageAlert{
		Enabled:     true,
		Title:       pkginfoMapString(object, "alert_title"),
		Detail:      pkginfoMapString(object, "alert_detail"),
		OKLabel:     pkginfoMapString(object, "ok_label"),
		CancelLabel: pkginfoMapString(object, "cancel_label"),
	}, nil
}

func pkginfoObjectList(item map[string]any, key string) ([]map[string]any, error) {
	value, ok := item[key]
	if !ok || value == nil {
		return nil, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: pkginfo %s must be an array of objects", dbutil.ErrInvalidInput, key)
	}
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: pkginfo %s must be an array of objects", dbutil.ErrInvalidInput, key)
		}
		out = append(out, object)
	}
	return out, nil
}

func pkginfoMapString(item map[string]any, key string) string {
	value, ok := item[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func pkginfoMapBool(item map[string]any, key string) bool {
	value, ok := item[key]
	if !ok || value == nil {
		return false
	}
	boolean, ok := value.(bool)
	return ok && boolean
}
