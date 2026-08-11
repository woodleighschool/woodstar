package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const (
	munkiSoftwarePath   = "/api/munki/software"
	munkiSoftwareIDPath = munkiSoftwarePath + "/{id}"
	munkiSoftwareLabel  = "Munki software"
)

type munkiSoftwareListInput struct {
	api.ListQueryInput
}

type munkiSoftwareGetInput struct {
	ID int64 `path:"id"`
}

type munkiSoftwareCreateInput struct {
	Body munkisoftware.CreateMutation
}

type munkiSoftwarePutInput struct {
	ID   int64 `path:"id"`
	Body munkisoftware.UpdateMutation
}

type munkiSoftwareDeleteInput struct {
	ID int64 `path:"id"`
}

type munkiSoftwareListOutput struct {
	Body api.Page[munkisoftware.Software]
}

type munkiSoftwareDetailOutput struct {
	Body munkiSoftwareDetail
}

type munkiSoftwareDetail struct {
	munkisoftware.Software

	Packages []packages.Package    `json:"packages"`
	Targets  munkisoftware.Targets `json:"targets"`
}

func (input munkiSoftwareListInput) params() listing.Params {
	return input.Params()
}

func registerMunkiSoftware(
	humaAPI huma.API,
	store *munkisoftware.Store,
	deletions *munki.SoftwareDeletionService,
	packageService *munki.PackageService,
	objects *storage.ObjectStore,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	registerListMunkiSoftware(humaAPI, store, logger)
	registerCreateMunkiSoftware(humaAPI, store, packageService, logger)
	registerGetMunkiSoftware(humaAPI, store, packageService, logger)
	registerPutMunkiSoftware(humaAPI, store, packageService, logger)
	registerDeleteMunkiSoftware(humaAPI, deletions, logger)
	registerBulkDeleteMunkiSoftware(humaAPI, deletions, logger)
	registerIconRoutes(humaAPI, store, objects, ingestor, logger)
}

func registerListMunkiSoftware(humaAPI huma.API, store *munkisoftware.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-munki-software",
		Method:      http.MethodGet,
		Path:        munkiSoftwarePath,
		Tags:        []string{api.TagMunkiSoftware},
		Summary:     "List software titles",
	}, func(ctx context.Context, input *munkiSoftwareListInput) (*munkiSoftwareListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-munki-software", munkiSoftwareLabel, err)
		}
		return &munkiSoftwareListOutput{
			Body: api.Page[munkisoftware.Software]{Items: rows, Count: count},
		}, nil
	})
}

func registerCreateMunkiSoftware(
	humaAPI huma.API,
	store *munkisoftware.Store,
	packageService *munki.PackageService,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-munki-software",
		Method:        http.MethodPost,
		Path:          munkiSoftwarePath,
		Tags:          []string{api.TagMunkiSoftware},
		Summary:       "Create a software title",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiSoftwareCreateInput) (*munkiSoftwareDetailOutput, error) {
		title, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-munki-software", munkiSoftwareLabel, err)
		}
		return loadMunkiSoftwareDetail(ctx, title.ID, store, packageService, logger, "create-munki-software")
	})
}

func registerGetMunkiSoftware(
	humaAPI huma.API,
	store *munkisoftware.Store,
	packageService *munki.PackageService,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-munki-software",
		Method:      http.MethodGet,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{api.TagMunkiSoftware},
		Summary:     "Get a software title",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareGetInput) (*munkiSoftwareDetailOutput, error) {
		return loadMunkiSoftwareDetail(ctx, input.ID, store, packageService, logger, "get-munki-software")
	})
}

func registerPutMunkiSoftware(
	humaAPI huma.API,
	store *munkisoftware.Store,
	packageService *munki.PackageService,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-munki-software",
		Method:      http.MethodPut,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{api.TagMunkiSoftware},
		Summary:     "Update a software title",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiSoftwarePutInput) (*munkiSoftwareDetailOutput, error) {
		title, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"update-munki-software",
				munkiSoftwareLabel,
				err,
				"software_id",
				input.ID,
			)
		}
		return loadMunkiSoftwareDetail(ctx, title.ID, store, packageService, logger, "update-munki-software")
	})
}

func registerDeleteMunkiSoftware(
	humaAPI huma.API,
	deletions *munki.SoftwareDeletionService,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-munki-software",
		Method:      http.MethodDelete,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{api.TagMunkiSoftware},
		Summary:     "Delete a software title",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareDeleteInput) (*struct{}, error) {
		if err := deletions.Delete(ctx, input.ID); err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"delete-munki-software",
				munkiSoftwareLabel,
				err,
				"software_id",
				input.ID,
			)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteMunkiSoftware(
	humaAPI huma.API,
	deletions *munki.SoftwareDeletionService,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "bulk-delete-munki-software",
		Method:        http.MethodDelete,
		Path:          munkiSoftwarePath,
		Tags:          []string{api.TagMunkiSoftware},
		Summary:       "Delete software titles",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *api.DeleteManyInput) (*struct{}, error) {
		if _, err := deletions.DeleteMany(ctx, input.IDs); err != nil {
			return nil, api.ResourceError(ctx, logger, "bulk-delete-munki-software", munkiSoftwareLabel, err)
		}
		return &struct{}{}, nil
	})
}

func loadMunkiSoftwareDetail(
	ctx context.Context,
	id int64,
	store *munkisoftware.Store,
	packageService *munki.PackageService,
	logger *slog.Logger,
	operation string,
) (*munkiSoftwareDetailOutput, error) {
	title, err := store.GetByID(ctx, id)
	if err != nil {
		return nil, api.ResourceError(ctx, logger, operation, munkiSoftwareLabel, err, "software_id", id)
	}
	packageRows, _, err := packageService.List(ctx, packages.PackageListParams{
		ListParams: listing.Params{PageSize: 1000},
		SoftwareID: id,
	})
	if err != nil {
		return nil, api.ResourceError(ctx, logger, operation, munkiPackageLabel, err, "software_id", id)
	}
	targets, err := store.TargetsForSoftware(ctx, id)
	if err != nil {
		return nil, api.ResourceError(ctx, logger, operation, munkiSoftwareLabel, err, "software_id", id)
	}
	return &munkiSoftwareDetailOutput{
		Body: munkiSoftwareDetail{
			Software: *title,
			Packages: packageRows,
			Targets:  targets,
		},
	}, nil
}
