package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
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
	ListQueryInput
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
	Body Page[munkisoftware.Software]
}

type munkiSoftwareDetailOutput struct {
	Body munkiSoftwareDetail
}

type munkiSoftwareDetail struct {
	munkisoftware.Software

	Packages []packages.Package    `json:"packages"`
	Targets  munkisoftware.Targets `json:"targets"`
}

func (input munkiSoftwareListInput) params() dbutil.ListParams {
	return input.ListQueryInput.params()
}

func registerMunkiSoftware(
	api huma.API,
	store *munkisoftware.Store,
	deletions *munki.SoftwareDeletionService,
	packageService *munki.PackageService,
	objects *storage.ObjectStore,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	registerListMunkiSoftware(api, store, logger)
	registerCreateMunkiSoftware(api, store, packageService, logger)
	registerGetMunkiSoftware(api, store, packageService, logger)
	registerPutMunkiSoftware(api, store, packageService, logger)
	registerDeleteMunkiSoftware(api, deletions, logger)
	registerBulkDeleteMunkiSoftware(api, deletions, logger)
	registerIconRoutes(api, store, objects, ingestor, logger)
}

func registerListMunkiSoftware(api huma.API, store *munkisoftware.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-software",
		Method:      http.MethodGet,
		Path:        munkiSoftwarePath,
		Tags:        []string{munkiSoftwareTag},
		Summary:     "List software titles",
	}, func(ctx context.Context, input *munkiSoftwareListInput) (*munkiSoftwareListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-munki-software", munkiSoftwareLabel, err)
		}
		return &munkiSoftwareListOutput{
			Body: Page[munkisoftware.Software]{Items: rows, Count: count},
		}, nil
	})
}

func registerCreateMunkiSoftware(
	api huma.API,
	store *munkisoftware.Store,
	packageService *munki.PackageService,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-software",
		Method:        http.MethodPost,
		Path:          munkiSoftwarePath,
		Tags:          []string{munkiSoftwareTag},
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
			return nil, resourceError(ctx, logger, "create-munki-software", munkiSoftwareLabel, err)
		}
		return loadMunkiSoftwareDetail(ctx, title.ID, store, packageService, logger, "create-munki-software")
	})
}

func registerGetMunkiSoftware(
	api huma.API,
	store *munkisoftware.Store,
	packageService *munki.PackageService,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-software",
		Method:      http.MethodGet,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{munkiSoftwareTag},
		Summary:     "Get a software title",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareGetInput) (*munkiSoftwareDetailOutput, error) {
		return loadMunkiSoftwareDetail(ctx, input.ID, store, packageService, logger, "get-munki-software")
	})
}

func registerPutMunkiSoftware(
	api huma.API,
	store *munkisoftware.Store,
	packageService *munki.PackageService,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-software",
		Method:      http.MethodPut,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{munkiSoftwareTag},
		Summary:     "Update a software title",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiSoftwarePutInput) (*munkiSoftwareDetailOutput, error) {
		title, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceError(
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
	api huma.API,
	deletions *munki.SoftwareDeletionService,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-munki-software",
		Method:      http.MethodDelete,
		Path:        munkiSoftwareIDPath,
		Tags:        []string{munkiSoftwareTag},
		Summary:     "Delete a software title",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareDeleteInput) (*struct{}, error) {
		if err := deletions.Delete(ctx, input.ID); err != nil {
			return nil, resourceError(
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
	api huma.API,
	deletions *munki.SoftwareDeletionService,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "bulk-delete-munki-software",
		Method:        http.MethodDelete,
		Path:          munkiSoftwarePath,
		Tags:          []string{munkiSoftwareTag},
		Summary:       "Delete software titles",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *deleteManyInput) (*struct{}, error) {
		if _, err := deletions.DeleteMany(ctx, input.IDs); err != nil {
			return nil, resourceError(ctx, logger, "bulk-delete-munki-software", munkiSoftwareLabel, err)
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
		return nil, resourceError(ctx, logger, operation, munkiSoftwareLabel, err, "software_id", id)
	}
	packageRows, _, err := packageService.List(ctx, packages.PackageListParams{
		ListParams: dbutil.ListParams{PageSize: 1000},
		SoftwareID: id,
	})
	if err != nil {
		return nil, resourceError(ctx, logger, operation, munkiPackageLabel, err, "software_id", id)
	}
	targets, err := store.TargetsForSoftware(ctx, id)
	if err != nil {
		return nil, resourceError(ctx, logger, operation, munkiSoftwareLabel, err, "software_id", id)
	}
	return &munkiSoftwareDetailOutput{
		Body: munkiSoftwareDetail{
			Software: *title,
			Packages: packageRows,
			Targets:  targets,
		},
	}, nil
}
