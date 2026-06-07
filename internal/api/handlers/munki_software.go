package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

const (
	munkiSoftwarePath   = "/api/munki/software"
	munkiSoftwareIDPath = "/api/munki/software/{id}"
	munkiSoftwareLabel  = "Munki software"
)

type munkiSoftwareGetInput struct {
	ID int64 `path:"id"`
}

type munkiSoftwareCreateInput struct {
	Body munkisoftware.SoftwareMutation
}

type munkiSoftwarePutInput struct {
	ID   int64 `path:"id"`
	Body munkisoftware.SoftwareMutation
}

type munkiSoftwareDeleteInput struct {
	ID int64 `path:"id"`
}

type munkiSoftwareBulkDeleteInput struct {
	Body bulkIDsBody
}

type munkiSoftwareListOutput struct {
	Body Page[munkiSoftware]
}

type munkiSoftwareDetailOutput struct {
	Body munkiSoftwareDetail
}

type munkiSoftwareDetail struct {
	munkisoftware.SoftwareTitle
	IconURL  string                        `json:"icon_url,omitempty"`
	Packages []munkiPackage                `json:"packages"`
	Targets  munkisoftware.SoftwareTargets `json:"targets"`
}

type munkiSoftware struct {
	munkisoftware.SoftwareTitle
	IconURL string `json:"icon_url,omitempty"`
}

func registerMunkiSoftware(
	api huma.API,
	store *munkisoftware.Store,
	packageStore *packages.Store,
) {
	registerListMunkiSoftware(api, store)
	registerCreateMunkiSoftware(api, store, packageStore)
	registerGetMunkiSoftware(api, store, packageStore)
	registerPutMunkiSoftware(api, store, packageStore)
	registerDeleteMunkiSoftware(api, store)
	registerBulkDeleteMunkiSoftware(api, store)
}

func registerListMunkiSoftware(api huma.API, store *munkisoftware.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-software",
		Method:      http.MethodGet,
		Path:        munkiSoftwarePath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki software",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiListInput) (*munkiSoftwareListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareLabel, err)
		}
		return &munkiSoftwareListOutput{
			Body: Page[munkiSoftware]{Items: munkiSoftwareFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiSoftware(
	api huma.API,
	store *munkisoftware.Store,
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
			return nil, resourceMutationError(munkiSoftwareLabel, err)
		}
		return loadMunkiSoftwareDetail(ctx, title.ID, store, packageStore)
	})
}

func registerGetMunkiSoftware(
	api huma.API,
	store *munkisoftware.Store,
	packageStore *packages.Store,
) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-software",
		Method:      http.MethodGet,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get Munki software",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareGetInput) (*munkiSoftwareDetailOutput, error) {
		return loadMunkiSoftwareDetail(ctx, input.ID, store, packageStore)
	})
}

func registerPutMunkiSoftware(
	api huma.API,
	store *munkisoftware.Store,
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
		title, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareLabel, err)
		}
		return loadMunkiSoftwareDetail(ctx, title.ID, store, packageStore)
	})
}

func registerDeleteMunkiSoftware(api huma.API, store *munkisoftware.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-munki-software",
		Method:      http.MethodDelete,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Delete Munki software",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if err := store.Delete(ctx, input.ID); err != nil {
			return nil, resourceMutationError(munkiSoftwareLabel, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteMunkiSoftware(api huma.API, store *munkisoftware.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-munki-software",
		Method:      http.MethodPost,
		Path:        munkiSoftwarePath + "/bulk-delete",
		Tags:        []string{munkiTag},
		Summary:     "Delete Munki software",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiSoftwareBulkDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if _, err := store.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, resourceMutationError(munkiSoftwareLabel, err)
		}
		return &struct{}{}, nil
	})
}

func munkiSoftwareDetailFromDomain(
	title munkisoftware.SoftwareTitle,
	packageRows []packages.Package,
	targets munkisoftware.SoftwareTargets,
) munkiSoftwareDetail {
	return munkiSoftwareDetail{
		SoftwareTitle: title,
		IconURL:       munkiSoftwareIconURL(title),
		Packages:      munkiPackagesFromDomain(packageRows),
		Targets:       targets,
	}
}

func loadMunkiSoftwareDetail(
	ctx context.Context,
	id int64,
	store *munkisoftware.Store,
	packageStore *packages.Store,
) (*munkiSoftwareDetailOutput, error) {
	title, err := store.GetByID(ctx, id)
	if err != nil {
		return nil, resourceMutationError(munkiSoftwareLabel, err)
	}
	packageRows, _, err := packageStore.List(ctx, packages.PackageListParams{
		ListParams: dbutil.ListParams{PageSize: 1000},
		SoftwareID: id,
	})
	if err != nil {
		return nil, resourceMutationError(munkiPackageLabel, err)
	}
	targets, err := store.TargetsForSoftwareTitle(ctx, id)
	if err != nil {
		return nil, resourceMutationError(munkiSoftwareLabel, err)
	}
	return &munkiSoftwareDetailOutput{
		Body: munkiSoftwareDetailFromDomain(*title, packageRows, targets),
	}, nil
}

func munkiSoftwareItemFromDomain(title munkisoftware.SoftwareTitle) munkiSoftware {
	return munkiSoftware{
		SoftwareTitle: title,
		IconURL:       munkiSoftwareIconURL(title),
	}
}

func munkiSoftwareFromDomain(rows []munkisoftware.SoftwareTitle) []munkiSoftware {
	items := make([]munkiSoftware, len(rows))
	for i, row := range rows {
		items[i] = munkiSoftwareItemFromDomain(row)
	}
	return items
}

func munkiSoftwareIconURL(title munkisoftware.SoftwareTitle) string {
	if title.IconArtifactID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/artifacts/%d/content", *title.IconArtifactID)
}
