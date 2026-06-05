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
	ID                        int64               `json:"id"`
	SoftwareID                int64               `json:"software_id"`
	SoftwareName              string              `json:"software_name"`
	SoftwareDisplayName       string              `json:"software_display_name"`
	Name                      string              `json:"name"`
	Version                   string              `json:"version"`
	DisplayName               string              `json:"display_name"`
	Description               string              `json:"description"`
	Category                  string              `json:"category"`
	Developer                 string              `json:"developer"`
	InstallerType             munki.InstallerType `json:"installer_type"`
	UnattendedInstall         bool                `json:"unattended_install"`
	UnattendedUninstall       bool                `json:"unattended_uninstall"`
	Uninstallable             bool                `json:"uninstallable"`
	UninstallMethod           string              `json:"uninstall_method"`
	RestartAction             munki.RestartAction `json:"restart_action,omitempty"`
	MinimumMunkiVersion       string              `json:"minimum_munki_version"`
	MinimumOSVersion          string              `json:"minimum_os_version"`
	MaximumOSVersion          string              `json:"maximum_os_version"`
	SupportedArchitectures    []string            `json:"supported_architectures"`
	BlockingApplications      []string            `json:"blocking_applications"`
	Requires                  []string            `json:"requires"`
	UpdateFor                 []string            `json:"update_for"`
	OnDemand                  bool                `json:"on_demand"`
	Precache                  bool                `json:"precache"`
	ExtraPkginfo              json.RawMessage     `json:"extra_pkginfo,omitempty"`
	Pkginfo                   json.RawMessage     `json:"pkginfo,omitempty"`
	InstallerArtifactID       *int64              `json:"installer_artifact_id,omitempty"`
	InstallerArtifactLocation string              `json:"installer_artifact_location,omitempty"`
	IconArtifactID            *int64              `json:"icon_artifact_id,omitempty"`
	IconArtifactLocation      string              `json:"icon_artifact_location,omitempty"`
	IconURL                   string              `json:"icon_url,omitempty"`
	Eligible                  bool                `json:"eligible"`
	CreatedAt                 time.Time           `json:"created_at"`
	UpdatedAt                 time.Time           `json:"updated_at"`
}

type munkiPackageMutation struct {
	SoftwareID             int64               `json:"software_id"`
	Name                   string              `json:"name"`
	Version                string              `json:"version"`
	DisplayName            string              `json:"display_name,omitempty"`
	Description            string              `json:"description,omitempty"`
	Category               string              `json:"category,omitempty"`
	Developer              string              `json:"developer,omitempty"`
	InstallerType          munki.InstallerType `json:"installer_type,omitempty"`
	UnattendedInstall      bool                `json:"unattended_install,omitempty"`
	UnattendedUninstall    bool                `json:"unattended_uninstall,omitempty"`
	Uninstallable          bool                `json:"uninstallable,omitempty"`
	UninstallMethod        string              `json:"uninstall_method,omitempty"`
	RestartAction          munki.RestartAction `json:"restart_action,omitempty"`
	MinimumMunkiVersion    string              `json:"minimum_munki_version,omitempty"`
	MinimumOSVersion       string              `json:"minimum_os_version,omitempty"`
	MaximumOSVersion       string              `json:"maximum_os_version,omitempty"`
	SupportedArchitectures []string            `json:"supported_architectures,omitempty"`
	BlockingApplications   []string            `json:"blocking_applications,omitempty"`
	Requires               []string            `json:"requires,omitempty"`
	UpdateFor              []string            `json:"update_for,omitempty"`
	OnDemand               bool                `json:"on_demand,omitempty"`
	Precache               bool                `json:"precache,omitempty"`
	ExtraPkginfo           json.RawMessage     `json:"extra_pkginfo,omitempty"`
	InstallerArtifactID    *int64              `json:"installer_artifact_id,omitempty"`
	IconArtifactID         *int64              `json:"icon_artifact_id,omitempty"`
	Eligible               bool                `json:"eligible"`
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
		ID:                        pkg.ID,
		SoftwareID:                pkg.SoftwareID,
		SoftwareName:              pkg.SoftwareName,
		SoftwareDisplayName:       pkg.SoftwareDisplayName,
		Name:                      pkg.Name,
		Version:                   pkg.Version,
		DisplayName:               pkg.DisplayName,
		Description:               pkg.Description,
		Category:                  pkg.Category,
		Developer:                 pkg.Developer,
		InstallerType:             pkg.InstallerType,
		UnattendedInstall:         pkg.UnattendedInstall,
		UnattendedUninstall:       pkg.UnattendedUninstall,
		Uninstallable:             pkg.Uninstallable,
		UninstallMethod:           pkg.UninstallMethod,
		RestartAction:             pkg.RestartAction,
		MinimumMunkiVersion:       pkg.MinimumMunkiVersion,
		MinimumOSVersion:          pkg.MinimumOSVersion,
		MaximumOSVersion:          pkg.MaximumOSVersion,
		SupportedArchitectures:    pkg.SupportedArchitectures,
		BlockingApplications:      pkg.BlockingApplications,
		Requires:                  pkg.Requires,
		UpdateFor:                 pkg.UpdateFor,
		OnDemand:                  pkg.OnDemand,
		Precache:                  pkg.Precache,
		ExtraPkginfo:              pkg.ExtraPkginfo,
		Pkginfo:                   pkg.Pkginfo,
		InstallerArtifactID:       pkg.InstallerArtifactID,
		InstallerArtifactLocation: pkg.InstallerArtifactLocation,
		IconArtifactID:            pkg.IconArtifactID,
		IconArtifactLocation:      pkg.IconArtifactLocation,
		IconURL:                   munkiPackageIconURL(pkg),
		Eligible:                  pkg.Eligible,
		CreatedAt:                 pkg.CreatedAt,
		UpdatedAt:                 pkg.UpdatedAt,
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
		SoftwareID:             body.SoftwareID,
		Name:                   body.Name,
		Version:                body.Version,
		DisplayName:            body.DisplayName,
		Description:            body.Description,
		Category:               body.Category,
		Developer:              body.Developer,
		InstallerType:          body.InstallerType,
		UnattendedInstall:      body.UnattendedInstall,
		UnattendedUninstall:    body.UnattendedUninstall,
		Uninstallable:          body.Uninstallable,
		UninstallMethod:        body.UninstallMethod,
		RestartAction:          body.RestartAction,
		MinimumMunkiVersion:    body.MinimumMunkiVersion,
		MinimumOSVersion:       body.MinimumOSVersion,
		MaximumOSVersion:       body.MaximumOSVersion,
		SupportedArchitectures: body.SupportedArchitectures,
		BlockingApplications:   body.BlockingApplications,
		Requires:               body.Requires,
		UpdateFor:              body.UpdateFor,
		OnDemand:               body.OnDemand,
		Precache:               body.Precache,
		ExtraPkginfo:           body.ExtraPkginfo,
		InstallerArtifactID:    body.InstallerArtifactID,
		IconArtifactID:         body.IconArtifactID,
		Eligible:               body.Eligible,
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
