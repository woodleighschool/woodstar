package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki"
)

const (
	munkiPackagePath       = "/api/munki/packages"
	munkiPackageImportPath = "/api/munki/packages/import"
	munkiPackageIDPath     = "/api/munki/packages/{id}"
	munkiPackageLabel      = "Munki package"
)

type munkiPackageListInput struct {
	ListQueryInput
	SoftwareID int64 `query:"software_id,omitempty"`
}

type munkiPackageGetInput struct {
	ID int64 `path:"id"`
}

type munkiPackageCreateInput struct {
	Body munkiPackageMutation
}

type munkiPackagePatchInput struct {
	ID   int64 `path:"id"`
	Body munkiPackageMutation
}

type munkiPackageImportInput struct {
	Body munkiPackageImportMutation
}

type munkiPackageListOutput struct {
	Body Page[munkiPackage]
}

type munkiPackageOutput struct {
	Body munkiPackage
}

type munkiPackage struct {
	ID                          int64                                       `json:"id"`
	SoftwareID                  int64                                       `json:"software_id"`
	SoftwareName                string                                      `json:"software_name"`
	SoftwareDisplayName         string                                      `json:"software_display_name"`
	Name                        string                                      `json:"name"`
	Version                     string                                      `json:"version"`
	DisplayName                 string                                      `json:"display_name"`
	Description                 string                                      `json:"description"`
	Category                    string                                      `json:"category"`
	Developer                   string                                      `json:"developer"`
	InstallerType               munki.InstallerType                         `json:"installer_type"`
	UnattendedInstall           bool                                        `json:"unattended_install"`
	UnattendedUninstall         bool                                        `json:"unattended_uninstall"`
	Uninstallable               bool                                        `json:"uninstallable"`
	UninstallMethod             munki.UninstallMethod                       `json:"uninstall_method"`
	CustomUninstallMethod       string                                      `json:"custom_uninstall_method"`
	RestartAction               munki.RestartAction                         `json:"restart_action,omitempty"`
	MinimumMunkiVersion         string                                      `json:"minimum_munki_version"`
	MinimumOSVersion            string                                      `json:"minimum_os_version"`
	MaximumOSVersion            string                                      `json:"maximum_os_version"`
	SupportedArchitectures      []string                                    `json:"supported_architectures"`
	BlockingApplications        []string                                    `json:"blocking_applications"`
	Requires                    []munki.PackageReference                    `json:"requires"`
	UpdateFor                   []munki.PackageReference                    `json:"update_for"`
	OnDemand                    bool                                        `json:"on_demand"`
	Precache                    bool                                        `json:"precache"`
	Autoremove                  bool                                        `json:"autoremove"`
	AppleItem                   bool                                        `json:"apple_item"`
	SuppressBundleRelocation    bool                                        `json:"suppress_bundle_relocation"`
	ForceInstallAfterDate       *time.Time                                  `json:"force_install_after_date,omitempty"`
	InstalledSize               int64                                       `json:"installed_size"`
	PayloadIdentifier           string                                      `json:"payload_identifier"`
	PackagePath                 string                                      `json:"package_path"`
	InstallerChoicesXML         string                                      `json:"installer_choices_xml"`
	InstallerEnvironment        []munki.PackageInstallerEnvironmentVariable `json:"installer_environment"`
	Installs                    []munki.PackageInstallItem                  `json:"installs"`
	Receipts                    []munki.PackageReceipt                      `json:"receipts"`
	ItemsToCopy                 []munki.PackageItemToCopy                   `json:"items_to_copy"`
	Notes                       string                                      `json:"notes"`
	InstallcheckScript          string                                      `json:"installcheck_script"`
	UninstallcheckScript        string                                      `json:"uninstallcheck_script"`
	PreinstallScript            string                                      `json:"preinstall_script"`
	PostinstallScript           string                                      `json:"postinstall_script"`
	PreuninstallScript          string                                      `json:"preuninstall_script"`
	PostuninstallScript         string                                      `json:"postuninstall_script"`
	UninstallScript             string                                      `json:"uninstall_script"`
	VersionScript               string                                      `json:"version_script"`
	PreinstallAlert             munki.PackageAlert                          `json:"preinstall_alert"`
	PreuninstallAlert           munki.PackageAlert                          `json:"preuninstall_alert"`
	InstallerArtifactID         *int64                                      `json:"installer_artifact_id,omitempty"`
	InstallerArtifactLocation   string                                      `json:"installer_artifact_location,omitempty"`
	UninstallerArtifactID       *int64                                      `json:"uninstaller_artifact_id,omitempty"`
	UninstallerArtifactLocation string                                      `json:"uninstaller_artifact_location,omitempty"`
	IconArtifactID              *int64                                      `json:"icon_artifact_id,omitempty"`
	IconArtifactLocation        string                                      `json:"icon_artifact_location,omitempty"`
	IconURL                     string                                      `json:"icon_url,omitempty"`
	Eligible                    bool                                        `json:"eligible"`
	CreatedAt                   time.Time                                   `json:"created_at"`
	UpdatedAt                   time.Time                                   `json:"updated_at"`
}

type munkiPackageMutation struct {
	SoftwareID               int64                                       `json:"software_id"`
	Name                     string                                      `json:"name"`
	Version                  string                                      `json:"version"`
	DisplayName              string                                      `json:"display_name,omitempty"`
	Description              string                                      `json:"description,omitempty"`
	Category                 string                                      `json:"category,omitempty"`
	Developer                string                                      `json:"developer,omitempty"`
	InstallerType            munki.InstallerType                         `json:"installer_type,omitempty"`
	UnattendedInstall        bool                                        `json:"unattended_install,omitempty"`
	UnattendedUninstall      bool                                        `json:"unattended_uninstall,omitempty"`
	Uninstallable            bool                                        `json:"uninstallable,omitempty"`
	UninstallMethod          munki.UninstallMethod                       `json:"uninstall_method,omitempty"`
	CustomUninstallMethod    string                                      `json:"custom_uninstall_method,omitempty"`
	RestartAction            munki.RestartAction                         `json:"restart_action,omitempty"`
	MinimumMunkiVersion      string                                      `json:"minimum_munki_version,omitempty"`
	MinimumOSVersion         string                                      `json:"minimum_os_version,omitempty"`
	MaximumOSVersion         string                                      `json:"maximum_os_version,omitempty"`
	SupportedArchitectures   []string                                    `json:"supported_architectures,omitempty"`
	BlockingApplications     []string                                    `json:"blocking_applications,omitempty"`
	Requires                 []munki.PackageReference                    `json:"requires,omitempty"`
	UpdateFor                []munki.PackageReference                    `json:"update_for,omitempty"`
	OnDemand                 bool                                        `json:"on_demand,omitempty"`
	Precache                 bool                                        `json:"precache,omitempty"`
	Autoremove               bool                                        `json:"autoremove,omitempty"`
	AppleItem                bool                                        `json:"apple_item,omitempty"`
	SuppressBundleRelocation bool                                        `json:"suppress_bundle_relocation,omitempty"`
	ForceInstallAfterDate    *time.Time                                  `json:"force_install_after_date,omitempty"`
	InstalledSize            int64                                       `json:"installed_size,omitempty"`
	PayloadIdentifier        string                                      `json:"payload_identifier,omitempty"`
	PackagePath              string                                      `json:"package_path,omitempty"`
	InstallerChoicesXML      string                                      `json:"installer_choices_xml,omitempty"`
	InstallerEnvironment     []munki.PackageInstallerEnvironmentVariable `json:"installer_environment,omitempty"`
	Installs                 []munki.PackageInstallItem                  `json:"installs,omitempty"`
	Receipts                 []munki.PackageReceipt                      `json:"receipts,omitempty"`
	ItemsToCopy              []munki.PackageItemToCopy                   `json:"items_to_copy,omitempty"`
	Notes                    string                                      `json:"notes,omitempty"`
	InstallcheckScript       string                                      `json:"installcheck_script,omitempty"`
	UninstallcheckScript     string                                      `json:"uninstallcheck_script,omitempty"`
	PreinstallScript         string                                      `json:"preinstall_script,omitempty"`
	PostinstallScript        string                                      `json:"postinstall_script,omitempty"`
	PreuninstallScript       string                                      `json:"preuninstall_script,omitempty"`
	PostuninstallScript      string                                      `json:"postuninstall_script,omitempty"`
	UninstallScript          string                                      `json:"uninstall_script,omitempty"`
	VersionScript            string                                      `json:"version_script,omitempty"`
	PreinstallAlert          munki.PackageAlert                          `json:"preinstall_alert,omitzero"`
	PreuninstallAlert        munki.PackageAlert                          `json:"preuninstall_alert,omitzero"`
	InstallerArtifactID      *int64                                      `json:"installer_artifact_id,omitempty"`
	UninstallerArtifactID    *int64                                      `json:"uninstaller_artifact_id,omitempty"`
	IconArtifactID           *int64                                      `json:"icon_artifact_id,omitempty"`
	Eligible                 bool                                        `json:"eligible"`
}

type munkiPackageImportMutation struct {
	SoftwareID          int64           `json:"software_id,omitempty"`
	Pkginfo             json.RawMessage `json:"pkginfo"`
	InstallerArtifactID *int64          `json:"installer_artifact_id,omitempty"`
	IconArtifactID      *int64          `json:"icon_artifact_id,omitempty"`
	Eligible            *bool           `json:"eligible,omitempty"`
}

func (input munkiPackageListInput) params() munki.PackageListParams {
	return munki.PackageListParams{
		ListParams: input.ListQueryInput.params(),
		SoftwareID: input.SoftwareID,
	}
}

func registerMunkiPackages(api huma.API, store *munki.Store) {
	registerListMunkiPackages(api, store)
	registerCreateMunkiPackage(api, store)
	registerImportMunkiPackage(api, store)
	registerGetMunkiPackage(api, store)
	registerPatchMunkiPackage(api, store)
}

func registerListMunkiPackages(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-packages",
		Method:      http.MethodGet,
		Path:        munkiPackagePath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki packages",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiPackageListInput) (*munkiPackageListOutput, error) {
		rows, count, err := store.ListPackages(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageListOutput{
			Body: Page[munkiPackage]{Items: munkiPackagesFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiPackage(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-package",
		Method:        http.MethodPost,
		Path:          munkiPackagePath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackageCreateInput) (*munkiPackageOutput, error) {
		pkg, err := store.CreatePackage(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerImportMunkiPackage(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "import-munki-package",
		Method:        http.MethodPost,
		Path:          munkiPackageImportPath,
		Tags:          []string{munkiTag},
		Summary:       "Import a Munki pkginfo package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackageImportInput) (*munkiPackageOutput, error) {
		pkg, err := store.ImportPackage(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerGetMunkiPackage(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-package",
		Method:      http.MethodGet,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki package",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageGetInput) (*munkiPackageOutput, error) {
		pkg, err := store.GetPackage(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerPatchMunkiPackage(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-package",
		Method:      http.MethodPatch,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Update a Munki package",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackagePatchInput) (*munkiPackageOutput, error) {
		pkg, err := store.UpdatePackage(ctx, input.ID, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func munkiPackageFromDomain(pkg munki.Package) munkiPackage {
	return munkiPackage{
		ID:                          pkg.ID,
		SoftwareID:                  pkg.SoftwareID,
		SoftwareName:                pkg.SoftwareName,
		SoftwareDisplayName:         pkg.SoftwareDisplayName,
		Name:                        pkg.Name,
		Version:                     pkg.Version,
		DisplayName:                 pkg.DisplayName,
		Description:                 pkg.Description,
		Category:                    pkg.Category,
		Developer:                   pkg.Developer,
		InstallerType:               pkg.InstallerType,
		UnattendedInstall:           pkg.UnattendedInstall,
		UnattendedUninstall:         pkg.UnattendedUninstall,
		Uninstallable:               pkg.Uninstallable,
		UninstallMethod:             pkg.UninstallMethod,
		CustomUninstallMethod:       pkg.CustomUninstallMethod,
		RestartAction:               pkg.RestartAction,
		MinimumMunkiVersion:         pkg.MinimumMunkiVersion,
		MinimumOSVersion:            pkg.MinimumOSVersion,
		MaximumOSVersion:            pkg.MaximumOSVersion,
		SupportedArchitectures:      pkg.SupportedArchitectures,
		BlockingApplications:        pkg.BlockingApplications,
		Requires:                    pkg.Requires,
		UpdateFor:                   pkg.UpdateFor,
		OnDemand:                    pkg.OnDemand,
		Precache:                    pkg.Precache,
		Autoremove:                  pkg.Autoremove,
		AppleItem:                   pkg.AppleItem,
		SuppressBundleRelocation:    pkg.SuppressBundleRelocation,
		ForceInstallAfterDate:       pkg.ForceInstallAfterDate,
		InstalledSize:               pkg.InstalledSize,
		PayloadIdentifier:           pkg.PayloadIdentifier,
		PackagePath:                 pkg.PackagePath,
		InstallerChoicesXML:         pkg.InstallerChoicesXML,
		InstallerEnvironment:        pkg.InstallerEnvironment,
		Installs:                    pkg.Installs,
		Receipts:                    pkg.Receipts,
		ItemsToCopy:                 pkg.ItemsToCopy,
		Notes:                       pkg.Notes,
		InstallcheckScript:          pkg.InstallcheckScript,
		UninstallcheckScript:        pkg.UninstallcheckScript,
		PreinstallScript:            pkg.PreinstallScript,
		PostinstallScript:           pkg.PostinstallScript,
		PreuninstallScript:          pkg.PreuninstallScript,
		PostuninstallScript:         pkg.PostuninstallScript,
		UninstallScript:             pkg.UninstallScript,
		VersionScript:               pkg.VersionScript,
		PreinstallAlert:             pkg.PreinstallAlert,
		PreuninstallAlert:           pkg.PreuninstallAlert,
		InstallerArtifactID:         pkg.InstallerArtifactID,
		InstallerArtifactLocation:   pkg.InstallerArtifactLocation,
		UninstallerArtifactID:       pkg.UninstallerArtifactID,
		UninstallerArtifactLocation: pkg.UninstallerArtifactLocation,
		IconArtifactID:              pkg.IconArtifactID,
		IconArtifactLocation:        pkg.IconArtifactLocation,
		IconURL:                     munkiPackageIconURL(pkg),
		Eligible:                    pkg.Eligible,
		CreatedAt:                   pkg.CreatedAt,
		UpdatedAt:                   pkg.UpdatedAt,
	}
}

func munkiPackagesFromDomain(rows []munki.Package) []munkiPackage {
	items := make([]munkiPackage, len(rows))
	for i, row := range rows {
		items[i] = munkiPackageFromDomain(row)
	}
	return items
}

func (body munkiPackageMutation) domain() munki.PackageMutation {
	return munki.PackageMutation{
		SoftwareID:               body.SoftwareID,
		Name:                     body.Name,
		Version:                  body.Version,
		DisplayName:              body.DisplayName,
		Description:              body.Description,
		Category:                 body.Category,
		Developer:                body.Developer,
		InstallerType:            body.InstallerType,
		UnattendedInstall:        body.UnattendedInstall,
		UnattendedUninstall:      body.UnattendedUninstall,
		Uninstallable:            body.Uninstallable,
		UninstallMethod:          body.UninstallMethod,
		CustomUninstallMethod:    body.CustomUninstallMethod,
		RestartAction:            body.RestartAction,
		MinimumMunkiVersion:      body.MinimumMunkiVersion,
		MinimumOSVersion:         body.MinimumOSVersion,
		MaximumOSVersion:         body.MaximumOSVersion,
		SupportedArchitectures:   body.SupportedArchitectures,
		BlockingApplications:     body.BlockingApplications,
		Requires:                 body.Requires,
		UpdateFor:                body.UpdateFor,
		OnDemand:                 body.OnDemand,
		Precache:                 body.Precache,
		Autoremove:               body.Autoremove,
		AppleItem:                body.AppleItem,
		SuppressBundleRelocation: body.SuppressBundleRelocation,
		ForceInstallAfterDate:    body.ForceInstallAfterDate,
		InstalledSize:            body.InstalledSize,
		PayloadIdentifier:        body.PayloadIdentifier,
		PackagePath:              body.PackagePath,
		InstallerChoicesXML:      body.InstallerChoicesXML,
		InstallerEnvironment:     body.InstallerEnvironment,
		Installs:                 body.Installs,
		Receipts:                 body.Receipts,
		ItemsToCopy:              body.ItemsToCopy,
		Notes:                    body.Notes,
		InstallcheckScript:       body.InstallcheckScript,
		UninstallcheckScript:     body.UninstallcheckScript,
		PreinstallScript:         body.PreinstallScript,
		PostinstallScript:        body.PostinstallScript,
		PreuninstallScript:       body.PreuninstallScript,
		PostuninstallScript:      body.PostuninstallScript,
		UninstallScript:          body.UninstallScript,
		VersionScript:            body.VersionScript,
		PreinstallAlert:          body.PreinstallAlert,
		PreuninstallAlert:        body.PreuninstallAlert,
		InstallerArtifactID:      body.InstallerArtifactID,
		UninstallerArtifactID:    body.UninstallerArtifactID,
		IconArtifactID:           body.IconArtifactID,
		Eligible:                 body.Eligible,
	}
}

func (body munkiPackageImportMutation) domain() munki.PackageImportMutation {
	return munki.PackageImportMutation{
		SoftwareID:          body.SoftwareID,
		Pkginfo:             body.Pkginfo,
		InstallerArtifactID: body.InstallerArtifactID,
		IconArtifactID:      body.IconArtifactID,
		Eligible:            body.Eligible,
	}
}

func munkiPackageIconURL(pkg munki.Package) string {
	artifactID := pkg.EffectiveIconArtifactID()
	if artifactID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/artifacts/%d/content", *artifactID)
}
