package packages

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
)

const (
	munkiTag               = "Munki"
	munkiPackagePath       = "/api/munki/packages"
	munkiPackageCreatePath = "/api/munki/software/{id}/packages"
	munkiPackageImportPath = munkiPackageCreatePath + "/import"
	munkiPackageIDPath     = "/api/munki/packages/{id}"
	munkiPackageLabel      = "Munki package"
)

type munkiPackageListInput struct {
	apitypes.ListQueryInput
	SoftwareID int64 `query:"software_id,omitempty"`
}

type munkiPackageGetInput struct {
	PackageID int64 `path:"id"`
}

type munkiPackageCreateInput struct {
	SoftwareID int64 `path:"id"`
	Body       PackageMutation
}

type munkiPackagePutInput struct {
	PackageID int64 `path:"id"`
	Body      PackageMutation
}

type munkiPackageDeleteInput struct {
	PackageID int64 `path:"id"`
}

type munkiPackageImportInput struct {
	SoftwareID int64 `path:"id"`
	Body       PackageImportMutation
}

type munkiPackageListOutput struct {
	Body apitypes.Page[MunkiPackage]
}

type munkiPackageOutput struct {
	Body MunkiPackage
}

// MunkiPackage is the shared admin API representation of one Munki package version.
type MunkiPackage struct {
	Package
	IconURL string `json:"icon_url,omitempty"`
}

func (input munkiPackageListInput) params() PackageListParams {
	return PackageListParams{
		ListParams: input.ListQueryInput.Params(),
		SoftwareID: input.SoftwareID,
	}
}

// RegisterAdminRoutes registers Munki package admin endpoints.
func RegisterAdminRoutes(api huma.API, store *Store) {
	registerListMunkiPackages(api, store)
	registerCreateMunkiPackage(api, store)
	registerImportMunkiPackage(api, store)
	registerGetMunkiPackage(api, store)
	registerPutMunkiPackage(api, store)
	registerDeleteMunkiPackage(api, store)
}

func registerListMunkiPackages(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-packages",
		Method:      http.MethodGet,
		Path:        munkiPackagePath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki packages",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiPackageListInput) (*munkiPackageListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageListOutput{
			Body: apitypes.Page[MunkiPackage]{Items: munkiPackagesFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiPackage(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-package",
		Method:        http.MethodPost,
		Path:          munkiPackageCreatePath,
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
		pkg, err := store.Create(ctx, input.SoftwareID, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerImportMunkiPackage(api huma.API, store *Store) {
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
		pkg, err := store.Import(ctx, input.SoftwareID, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerGetMunkiPackage(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-package",
		Method:      http.MethodGet,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki package",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageGetInput) (*munkiPackageOutput, error) {
		pkg, err := store.GetByID(ctx, input.PackageID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerPutMunkiPackage(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-package",
		Method:      http.MethodPut,
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
	}, func(ctx context.Context, input *munkiPackagePutInput) (*munkiPackageOutput, error) {
		pkg, err := store.Update(ctx, input.PackageID, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerDeleteMunkiPackage(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-munki-package",
		Method:      http.MethodDelete,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Delete a Munki package",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *munkiPackageDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.PackageID); err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &struct{}{}, nil
	})
}

func munkiPackageFromDomain(pkg Package) MunkiPackage {
	return MunkiPackage{
		Package: pkg,
		IconURL: munkiPackageIconURL(pkg),
	}
}

func munkiPackagesFromDomain(rows []Package) []MunkiPackage {
	items := make([]MunkiPackage, len(rows))
	for i, row := range rows {
		items[i] = munkiPackageFromDomain(row)
	}
	return items
}

func munkiPackageIconURL(pkg Package) string {
	artifactID := EffectiveIconArtifactID(pkg)
	if artifactID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/artifacts/%d/content", *artifactID)
}
