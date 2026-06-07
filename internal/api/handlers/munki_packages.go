package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

const (
	munkiPackagePath       = "/api/munki/packages"
	munkiPackageImportPath = "/api/munki/packages/import"
	munkiPackageIDPath     = "/api/munki/packages/{id}"
	munkiPackageLabel      = "Munki package"
)

type munkiPackageListInput struct {
	apitypes.ListQueryInput
	SoftwareID int64 `query:"software_id,omitempty"`
}

type munkiPackageGetInput struct {
	ID int64 `path:"id"`
}

type munkiPackageCreateInput struct {
	Body packages.PackageMutation
}

type munkiPackagePatchInput struct {
	ID   int64 `path:"id"`
	Body packages.PackageMutation
}

type munkiPackageImportInput struct {
	Body packages.PackageImportMutation
}

type munkiPackageListOutput struct {
	Body apitypes.Page[munkiPackage]
}

type munkiPackageOutput struct {
	Body munkiPackage
}

type munkiPackage struct {
	packages.Package
	IconURL string `json:"icon_url,omitempty"`
}

func (input munkiPackageListInput) params() packages.PackageListParams {
	return packages.PackageListParams{
		ListParams: input.ListQueryInput.Params(),
		SoftwareID: input.SoftwareID,
	}
}

func registerMunkiPackages(api huma.API, store *packages.Store) {
	registerListMunkiPackages(api, store)
	registerCreateMunkiPackage(api, store)
	registerImportMunkiPackage(api, store)
	registerGetMunkiPackage(api, store)
	registerPatchMunkiPackage(api, store)
}

func registerListMunkiPackages(api huma.API, store *packages.Store) {
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
			Body: apitypes.Page[munkiPackage]{Items: munkiPackagesFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiPackage(api huma.API, store *packages.Store) {
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
		pkg, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerImportMunkiPackage(api huma.API, store *packages.Store) {
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
		pkg, err := store.Import(ctx, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerGetMunkiPackage(api huma.API, store *packages.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-package",
		Method:      http.MethodGet,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki package",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageGetInput) (*munkiPackageOutput, error) {
		pkg, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func registerPatchMunkiPackage(api huma.API, store *packages.Store) {
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
		pkg, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: munkiPackageFromDomain(*pkg)}, nil
	})
}

func munkiPackageFromDomain(pkg packages.Package) munkiPackage {
	return munkiPackage{
		Package: pkg,
		IconURL: munkiPackageIconURL(pkg),
	}
}

func munkiPackagesFromDomain(rows []packages.Package) []munkiPackage {
	items := make([]munkiPackage, len(rows))
	for i, row := range rows {
		items[i] = munkiPackageFromDomain(row)
	}
	return items
}

func munkiPackageIconURL(pkg packages.Package) string {
	artifactID := packages.EffectiveIconArtifactID(pkg)
	if artifactID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/artifacts/%d/content", *artifactID)
}
