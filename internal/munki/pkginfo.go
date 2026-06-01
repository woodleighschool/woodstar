package munki

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

var importedPkginfoKeys = map[string]struct{}{
	"OnDemand":                {},
	"PackageCompleteURL":      {},
	"PackageURL":              {},
	"RestartAction":           {},
	"blocking_applications":   {},
	"catalogs":                {},
	"category":                {},
	"description":             {},
	"developer":               {},
	"display_name":            {},
	"icon_hash":               {},
	"icon_name":               {},
	"installer_item_location": {},
	"installer_type":          {},
	"maximum_os_version":      {},
	"minimum_munki_version":   {},
	"minimum_os_version":      {},
	"name":                    {},
	"precache":                {},
	"requires":                {},
	"supported_architectures": {},
	"unattended_install":      {},
	"unattended_uninstall":    {},
	"uninstall_method":        {},
	"uninstallable":           {},
	"update_for":              {},
	"version":                 {},
}

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
	mutation, err := packageMutationFromPkginfoFields(item)
	if err != nil {
		return PackageMutation{}, err
	}
	extra, err := importedExtraPkginfo(item)
	if err != nil {
		return PackageMutation{}, err
	}
	mutation.ExtraPkginfo = extra
	return mutation, nil
}

func packageMutationFromPkginfoFields(item map[string]any) (PackageMutation, error) {
	mutation := PackageMutation{
		Name:                pkginfoString(item, "name"),
		Version:             pkginfoString(item, "version"),
		DisplayName:         pkginfoString(item, "display_name"),
		Description:         pkginfoString(item, "description"),
		Category:            pkginfoString(item, "category"),
		Developer:           pkginfoString(item, "developer"),
		InstallerType:       InstallerType(pkginfoString(item, "installer_type")),
		UninstallMethod:     pkginfoString(item, "uninstall_method"),
		RestartAction:       RestartAction(pkginfoString(item, "RestartAction")),
		MinimumMunkiVersion: pkginfoString(item, "minimum_munki_version"),
		MinimumOSVersion:    pkginfoString(item, "minimum_os_version"),
		MaximumOSVersion:    pkginfoString(item, "maximum_os_version"),
		IconName:            pkginfoString(item, "icon_name"),
		IconHash:            pkginfoString(item, "icon_hash"),
		UnattendedInstall:   pkginfoBool(item, "unattended_install"),
		UnattendedUninstall: pkginfoBool(item, "unattended_uninstall"),
		Uninstallable:       pkginfoBool(item, "uninstallable"),
		OnDemand:            pkginfoBool(item, "OnDemand"),
		Precache:            pkginfoBool(item, "precache"),
	}
	var err error
	if mutation.SupportedArchitectures, err = pkginfoStringList(item, "supported_architectures"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.BlockingApplications, err = pkginfoStringList(item, "blocking_applications"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.Requires, err = pkginfoStringList(item, "requires"); err != nil {
		return PackageMutation{}, err
	}
	if mutation.UpdateFor, err = pkginfoStringList(item, "update_for"); err != nil {
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

func importedExtraPkginfo(item map[string]any) (json.RawMessage, error) {
	extra := make(map[string]any, len(item))
	for key, value := range item {
		if _, ok := importedPkginfoKeys[key]; ok {
			continue
		}
		extra[key] = value
	}
	if len(extra) == 0 {
		return json.RawMessage(`{}`), nil
	}
	raw, err := json.Marshal(extra)
	if err != nil {
		return nil, fmt.Errorf("marshal extra pkginfo: %w", err)
	}
	return raw, nil
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
