package munki

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const pkginfoReceiptPackageIDKey = "package" + "id"

// ImportPackage imports one Munki pkginfo item and upserts it by title, name,
// and version.
func (s *Store) ImportPackage(ctx context.Context, params PackageImportMutation) (*Package, error) {
	params = cleanPackageImportMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	mutation, err := packageMutationFromPkginfo(params.Pkginfo)
	if err != nil {
		return nil, err
	}
	title, err := s.importPackageSoftwareTitle(ctx, params.SoftwareID, mutation)
	if err != nil {
		return nil, err
	}
	mutation.SoftwareID = title.ID
	mutation.InstallerArtifactID = params.InstallerArtifactID
	mutation.IconArtifactID = params.IconArtifactID
	mutation.Eligible = true
	if params.Eligible != nil {
		mutation.Eligible = *params.Eligible
	}
	return s.UpsertPackage(ctx, mutation)
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
		Name:                     pkginfoString(item, "name"),
		Version:                  pkginfoString(item, "version"),
		DisplayName:              pkginfoString(item, "display_name"),
		Description:              pkginfoString(item, "description"),
		Category:                 pkginfoString(item, "category"),
		Developer:                pkginfoString(item, "developer"),
		InstallerType:            InstallerType(pkginfoString(item, "installer_type")),
		UninstallMethod:          pkginfoUninstallMethod(item),
		CustomUninstallMethod:    pkginfoCustomUninstallMethod(item),
		RestartAction:            RestartAction(pkginfoString(item, "RestartAction")),
		MinimumMunkiVersion:      pkginfoString(item, "minimum_munki_version"),
		MinimumOSVersion:         pkginfoString(item, "minimum_os_version"),
		MaximumOSVersion:         pkginfoString(item, "maximum_os_version"),
		IconName:                 pkginfoString(item, "icon_name"),
		IconHash:                 pkginfoString(item, "icon_hash"),
		UnattendedInstall:        pkginfoBool(item, "unattended_install"),
		UnattendedUninstall:      pkginfoBool(item, "unattended_uninstall"),
		Uninstallable:            pkginfoBool(item, "uninstallable"),
		OnDemand:                 pkginfoBool(item, "OnDemand"),
		Precache:                 pkginfoBool(item, "precache"),
		Autoremove:               pkginfoBool(item, "autoremove"),
		AppleItem:                pkginfoBool(item, "apple_item"),
		SuppressBundleRelocation: pkginfoBool(item, "suppress_bundle_relocation"),
		InstalledSize:            pkginfoInt64(item, "installed_size"),
		PayloadIdentifier:        pkginfoString(item, "payload_identifier"),
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
	var err error
	if mutation.SupportedArchitectures, err = pkginfoStringList(item, "supported_architectures"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.BlockingApplications, err = pkginfoStringList(item, "blocking_applications"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.Requires, err = pkginfoPackageReferences(item, "requires"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.UpdateFor, err = pkginfoPackageReferences(item, "update_for"); err != nil {
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

func (s *Store) importPackageSoftwareTitle(
	ctx context.Context,
	softwareID int64,
	mutation PackageMutation,
) (*SoftwareTitle, error) {
	if softwareID > 0 {
		return s.GetSoftwareTitle(ctx, softwareID)
	}
	title, err := s.GetSoftwareTitleByName(ctx, mutation.Name)
	if err == nil {
		return title, nil
	}
	if !errors.Is(err, dbutil.ErrNotFound) {
		return nil, err
	}
	return s.CreateSoftwareTitle(ctx, SoftwareTitleMutation{
		Name:        mutation.Name,
		DisplayName: mutation.DisplayName,
		Description: mutation.Description,
		Category:    mutation.Category,
		Developer:   mutation.Developer,
	})
}

func cleanPackageImportMutation(params PackageImportMutation) PackageImportMutation {
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
		UninstallMethodRemoveProfile,
		UninstallMethodRemoveApp,
		UninstallMethodUninstallScript,
		UninstallMethodUninstallPackage:
		return UninstallMethod(method)
	default:
		return UninstallMethodCustom
	}
}

func pkginfoCustomUninstallMethod(item map[string]any) string {
	method := pkginfoString(item, "uninstall_method")
	if pkginfoUninstallMethod(item) == UninstallMethodCustom {
		return method
	}
	return ""
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

func pkginfoPackageReferences(item map[string]any, key string) ([]PackageReference, error) {
	values, err := pkginfoStringList(item, key)
	if err != nil {
		return nil, err
	}
	out := make([]PackageReference, 0, len(values))
	for _, value := range values {
		out = append(out, PackageReference{Name: value})
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
