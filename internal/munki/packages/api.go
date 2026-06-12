package packages

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
)

const (
	munkiTag           = "Munki"
	munkiPackagePath   = "/api/munki/packages"
	munkiPackageIDPath = "/api/munki/packages/{id}"
	munkiPackageLabel  = "Munki package"
)

type munkiPackageListInput struct {
	apitypes.ListQueryInput

	SoftwareID int64 `query:"software_id,omitempty"`
}

type munkiPackageGetInput struct {
	PackageID int64 `path:"id"`
}

type munkiPackageCreateInput struct {
	Body PackageCreateMutation
}

type munkiPackagePutInput struct {
	PackageID int64 `path:"id"`
	Body      PackageMutation
}

type munkiPackageDeleteInput struct {
	PackageID int64 `path:"id"`
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
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *munkiPackageListInput) (*munkiPackageListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageListOutput{
			Body: apitypes.Page[MunkiPackage]{Items: MunkiPackagesFromRecords(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiPackage(api huma.API, store *Store) {
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
		pkg, err := store.Create(ctx, input.Body.SoftwareID, input.Body.PackageMutation)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: MunkiPackageFromRecord(*pkg)}, nil
	})
}

func registerGetMunkiPackage(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-package",
		Method:      http.MethodGet,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki package",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageGetInput) (*munkiPackageOutput, error) {
		pkg, err := store.GetByID(ctx, input.PackageID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: MunkiPackageFromRecord(*pkg)}, nil
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
		return &munkiPackageOutput{Body: MunkiPackageFromRecord(*pkg)}, nil
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

// MunkiPackageFromRecord maps a joined package read model to the admin API shape.
func MunkiPackageFromRecord(record PackageRecord) MunkiPackage {
	return MunkiPackage{
		Package: record.Package,
		IconURL: munkiPackageIconURL(record.SoftwareIcon),
	}
}

// MunkiPackagesFromRecords maps joined package read models to the admin API shape.
func MunkiPackagesFromRecords(rows []PackageRecord) []MunkiPackage {
	items := make([]MunkiPackage, len(rows))
	for i, row := range rows {
		items[i] = MunkiPackageFromRecord(row)
	}
	return items
}

func munkiPackageIconURL(softwareIcon IconRef) string {
	return munkiArtifactContentURL(softwareIcon.ArtifactID)
}

func munkiArtifactContentURL(artifactID *int64) string {
	if artifactID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/artifacts/%d/content", *artifactID)
}
