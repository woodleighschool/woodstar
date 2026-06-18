package packages

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const (
	munkiTag           = "Munki"
	munkiPackagePath   = "/api/munki/packages"
	munkiPackageIDPath = "/api/munki/packages/{id}"
	munkiPackageLabel  = "Munki package"
)

type munkiPackageListInput struct {
	apitypes.ListQueryInput

	Types      []InstallerType `query:"type,omitempty"`
	SoftwareID int64           `query:"software_id,omitempty"`
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

type munkiPackageBulkDeleteInput struct {
	Body apitypes.BulkIDsBody
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
		ListParams:     input.ListQueryInput.Params(),
		InstallerTypes: installerTypeFilterValues(input.Types),
		SoftwareID:     input.SoftwareID,
	}
}

func installerTypeFilterValues(types []InstallerType) []string {
	values := make([]string, len(types))
	for i, installerType := range types {
		values[i] = string(installerType)
	}
	return values
}

// RegisterAdminRoutes registers Munki package admin endpoints.
func RegisterAdminRoutes(
	api huma.API,
	store *Store,
	objects *storage.ObjectStore,
	storageStore storage.Presigner,
	notifier DesiredNotifier,
) {
	registerListMunkiPackages(api, store)
	registerCreateMunkiPackage(api, store)
	registerGetMunkiPackage(api, store)
	registerPutMunkiPackage(api, store)
	registerDeleteMunkiPackage(api, store, notifier)
	registerBulkDeleteMunkiPackages(api, store, notifier)
	registerObjectRoutes(api, store, objects, storageStore, notifier)
}

// DesiredNotifier is told when a mutation may have changed the set of installers
// distribution points should mirror. The MDP hub satisfies it; a nil notifier
// (schema generation) is a no-op.
type DesiredNotifier interface {
	DesiredChanged()
}

func notifyDesired(notifier DesiredNotifier) {
	if notifier != nil {
		notifier.DesiredChanged()
	}
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
			Body: apitypes.Page[MunkiPackage]{Items: MunkiPackagesFromPackages(rows), Count: int32(count)},
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
		return &munkiPackageOutput{Body: MunkiPackageFromPackage(*pkg)}, nil
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
		return &munkiPackageOutput{Body: MunkiPackageFromPackage(*pkg)}, nil
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
		return &munkiPackageOutput{Body: MunkiPackageFromPackage(*pkg)}, nil
	})
}

func registerDeleteMunkiPackage(api huma.API, store *Store, notifier DesiredNotifier) {
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
		notifyDesired(notifier)
		return &struct{}{}, nil
	})
}

func registerBulkDeleteMunkiPackages(api huma.API, store *Store, notifier DesiredNotifier) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-munki-packages",
		Method:      http.MethodPost,
		Path:        munkiPackagePath + "/bulk-delete",
		Tags:        []string{munkiTag},
		Summary:     "Delete Munki packages",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusConflict},
	}, func(ctx context.Context, input *munkiPackageBulkDeleteInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
		}
		notifyDesired(notifier)
		return &struct{}{}, nil
	})
}

// MunkiPackageFromPackage maps a package read model to the admin API shape.
func MunkiPackageFromPackage(pkg Package) MunkiPackage {
	return MunkiPackage{
		Package: pkg,
		IconURL: objectContentURL(pkg.IconObjectID),
	}
}

// MunkiPackagesFromPackages maps package read models to the admin API shape.
func MunkiPackagesFromPackages(rows []Package) []MunkiPackage {
	items := make([]MunkiPackage, len(rows))
	for i, row := range rows {
		items[i] = MunkiPackageFromPackage(row)
	}
	return items
}

func objectContentURL(objectID *int64) string {
	if objectID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/icons/%d/content", *objectID)
}
