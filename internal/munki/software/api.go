package software

import (
	"context"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const (
	munkiTag            = "Munki"
	munkiPackageLabel   = "Munki package"
	munkiSoftwarePath   = "/api/munki/software"
	munkiSoftwareIDPath = "/api/munki/software/{id}"
	munkiSoftwareLabel  = "Munki software"
)

type munkiSoftwareListInput struct {
	apitypes.ListQueryInput
}

type munkiSoftwareGetInput struct {
	SoftwareID int64 `path:"id"`
}

type munkiSoftwareCreateInput struct {
	Body Mutation
}

type munkiSoftwarePutInput struct {
	SoftwareID int64 `path:"id"`
	Body       Mutation
}

type munkiSoftwareDeleteInput struct {
	SoftwareID int64 `path:"id"`
}

type munkiSoftwareBulkDeleteInput struct {
	Body apitypes.BulkIDsBody
}

type munkiSoftwareListOutput struct {
	Body apitypes.Page[munkiSoftware]
}

type munkiSoftwareDetailOutput struct {
	Body munkiSoftwareDetail
}

type munkiSoftwareDetail struct {
	Software

	IconURL  string                  `json:"icon_url,omitempty"`
	Packages []packages.MunkiPackage `json:"packages"`
	Targets  Targets                 `json:"targets"`
}

type munkiSoftware struct {
	Software

	IconURL string `json:"icon_url,omitempty"`
}

func (input munkiSoftwareListInput) params() dbutil.ListParams {
	return input.ListQueryInput.Params()
}

// RegisterAdminRoutes registers Munki software admin endpoints.
func RegisterAdminRoutes(
	api huma.API,
	store *Store,
	packageStore *packages.Store,
	objects *storage.ObjectStore,
	storageStore storage.Presigner,
	notifier desiredNotifier,
) {
	registerListMunkiSoftware(api, store)
	registerCreateMunkiSoftware(api, store, packageStore)
	registerGetMunkiSoftware(api, store, packageStore)
	registerPutMunkiSoftware(api, store, packageStore)
	registerDeleteMunkiSoftware(api, store, notifier)
	registerBulkDeleteMunkiSoftware(api, store, notifier)
	registerIconRoutes(api, store, objects, storageStore)
}

// desiredNotifier is told when deleting software changes the set of installers
// distribution points mirror. Deleting software cascades to its packages, so the
// MDP hub must re-push the desired set. The hub satisfies it; a nil notifier
// (schema generation) is a no-op.
type desiredNotifier interface {
	DesiredChanged()
}

func notifyDesired(notifier desiredNotifier) {
	if notifier != nil {
		notifier.DesiredChanged()
	}
}

func registerListMunkiSoftware(
	api huma.API,
	store *Store,
) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-software",
		Method:      http.MethodGet,
		Path:        munkiSoftwarePath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki software",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *munkiSoftwareListInput) (*munkiSoftwareListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiSoftwareLabel, err)
		}
		items := make([]munkiSoftware, len(rows))
		for i, row := range rows {
			items[i] = munkiSoftware{
				Software: row,
				IconURL:  munkiSoftwareIconURL(row),
			}
		}
		return &munkiSoftwareListOutput{
			Body: apitypes.Page[munkiSoftware]{Items: items, Count: int32(count)},
		}, nil
	})
}

func registerCreateMunkiSoftware(
	api huma.API,
	store *Store,
	packageStore *packages.Store,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-software",
		Method:        http.MethodPost,
		Path:          munkiSoftwarePath,
		Tags:          []string{munkiTag},
		Summary:       "Create Munki software",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiSoftwareCreateInput) (*munkiSoftwareDetailOutput, error) {
		title, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiSoftwareLabel, err)
		}
		return loadMunkiSoftwareDetail(ctx, title.ID, store, packageStore)
	})
}

func registerGetMunkiSoftware(
	api huma.API,
	store *Store,
	packageStore *packages.Store,
) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-software",
		Method:      http.MethodGet,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get Munki software",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareGetInput) (*munkiSoftwareDetailOutput, error) {
		return loadMunkiSoftwareDetail(ctx, input.SoftwareID, store, packageStore)
	})
}

func registerPutMunkiSoftware(
	api huma.API,
	store *Store,
	packageStore *packages.Store,
) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-software",
		Method:      http.MethodPut,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Update Munki software",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiSoftwarePutInput) (*munkiSoftwareDetailOutput, error) {
		title, err := store.Update(ctx, input.SoftwareID, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiSoftwareLabel, err)
		}
		return loadMunkiSoftwareDetail(ctx, title.ID, store, packageStore)
	})
}

func registerDeleteMunkiSoftware(api huma.API, store *Store, notifier desiredNotifier) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-munki-software",
		Method:      http.MethodDelete,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Delete Munki software",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.SoftwareID); err != nil {
			return nil, apitypes.ResourceMutationError(munkiSoftwareLabel, err)
		}
		notifyDesired(notifier)
		return &struct{}{}, nil
	})
}

func registerBulkDeleteMunkiSoftware(api huma.API, store *Store, notifier desiredNotifier) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-munki-software",
		Method:      http.MethodPost,
		Path:        munkiSoftwarePath + "/bulk-delete",
		Tags:        []string{munkiTag},
		Summary:     "Delete Munki software",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiSoftwareBulkDeleteInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, apitypes.ResourceMutationError(munkiSoftwareLabel, err)
		}
		notifyDesired(notifier)
		return &struct{}{}, nil
	})
}

func loadMunkiSoftwareDetail(
	ctx context.Context,
	id int64,
	store *Store,
	packageStore *packages.Store,
) (*munkiSoftwareDetailOutput, error) {
	title, err := store.GetByID(ctx, id)
	if err != nil {
		return nil, apitypes.ResourceMutationError(munkiSoftwareLabel, err)
	}
	packageRows, _, err := packageStore.List(ctx, packages.PackageListParams{
		ListParams: dbutil.ListParams{PageSize: 1000},
		SoftwareID: id,
	})
	if err != nil {
		return nil, apitypes.ResourceMutationError(munkiPackageLabel, err)
	}
	targets, err := store.TargetsForSoftware(ctx, id)
	if err != nil {
		return nil, apitypes.ResourceMutationError(munkiSoftwareLabel, err)
	}
	return &munkiSoftwareDetailOutput{
		Body: munkiSoftwareDetail{
			Software: *title,
			IconURL:  munkiSoftwareIconURL(*title),
			Packages: packages.MunkiPackagesFromPackages(packageRows),
			Targets:  targets,
		},
	}, nil
}

func munkiSoftwareIconURL(title Software) string {
	if title.IconObjectID == nil {
		return ""
	}
	return munkiSoftwarePath + "/" + strconv.FormatInt(title.ID, 10) + "/icon"
}
