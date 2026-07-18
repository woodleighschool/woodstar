package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const (
	munkiPackagePath   = "/api/munki/packages"
	munkiPackageIDPath = "/api/munki/packages/{id}"
	munkiPackageLabel  = "Munki package"
)

type munkiPackageListInput struct {
	ListQueryInput

	Types      []packages.InstallerType `query:"type,omitempty"`
	SoftwareID int64                    `query:"software_id,omitempty"`
}

type munkiPackageGetInput struct {
	ID int64 `path:"id"`
}

type munkiPackageCreateInput struct {
	Body packages.PackageCreateMutation
}

type munkiPackagePutInput struct {
	ID   int64 `path:"id"`
	Body packages.PackageMutation
}

type munkiPackageDeleteInput struct {
	ID int64 `path:"id"`
}

type munkiPackageListOutput struct {
	Body Page[packages.Package]
}

type munkiPackageOutput struct {
	Body packages.Package
}

func (input munkiPackageListInput) params() packages.PackageListParams {
	return packages.PackageListParams{
		ListParams:     input.ListQueryInput.params(),
		InstallerTypes: installerTypeFilterValues(input.Types),
		SoftwareID:     input.SoftwareID,
	}
}

func installerTypeFilterValues(types []packages.InstallerType) []string {
	values := make([]string, len(types))
	for i, installerType := range types {
		values[i] = string(installerType)
	}
	return values
}

func registerMunkiPackages(
	api huma.API,
	longRunningAPI huma.API,
	store *munki.PackageService,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	registerListMunkiPackages(api, store, logger)
	registerCreateMunkiPackage(api, store, logger)
	registerGetMunkiPackage(api, store, logger)
	registerPutMunkiPackage(api, store, logger)
	registerDeleteMunkiPackage(api, store, logger)
	registerBulkDeleteMunkiPackages(api, store, logger)
	registerPackageInstallerRoutes(api, longRunningAPI, ingestor, logger)
}

func registerListMunkiPackages(api huma.API, store *munki.PackageService, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-packages",
		Method:      http.MethodGet,
		Path:        munkiPackagePath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki packages",
	}, func(ctx context.Context, input *munkiPackageListInput) (*munkiPackageListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-munki-packages", munkiPackageLabel, err)
		}
		return &munkiPackageListOutput{
			Body: Page[packages.Package]{
				Items: rows,
				Count: count,
			},
		}, nil
	})
}

func registerCreateMunkiPackage(api huma.API, store *munki.PackageService, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-package",
		Method:        http.MethodPost,
		Path:          munkiPackagePath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackageCreateInput) (*munkiPackageOutput, error) {
		pkg, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, resourceError(ctx, logger, "create-munki-package", munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: *pkg}, nil
	})
}

func registerGetMunkiPackage(api huma.API, store *munki.PackageService, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-package",
		Method:      http.MethodGet,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki package",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageGetInput) (*munkiPackageOutput, error) {
		pkg, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"get-munki-package",
				munkiPackageLabel,
				err,
				"package_id",
				input.ID,
			)
		}
		return &munkiPackageOutput{Body: *pkg}, nil
	})
}

func registerPutMunkiPackage(api huma.API, store *munki.PackageService, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-package",
		Method:      http.MethodPut,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Update a Munki package",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackagePutInput) (*munkiPackageOutput, error) {
		pkg, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"update-munki-package",
				munkiPackageLabel,
				err,
				"package_id",
				input.ID,
			)
		}
		return &munkiPackageOutput{Body: *pkg}, nil
	})
}

func registerDeleteMunkiPackage(
	api huma.API,
	store *munki.PackageService,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-munki-package",
		Method:      http.MethodDelete,
		Path:        munkiPackageIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Delete a Munki package",
		Errors:      []int{http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *munkiPackageDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ID); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"delete-munki-package",
				munkiPackageLabel,
				err,
				"package_id",
				input.ID,
			)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteMunkiPackages(
	api huma.API,
	store *munki.PackageService,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "bulk-delete-munki-packages",
		Method:        http.MethodDelete,
		Path:          munkiPackagePath,
		Tags:          []string{munkiTag},
		Summary:       "Delete Munki packages",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest, http.StatusConflict},
	}, func(ctx context.Context, input *deleteManyInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.IDs); err != nil {
			return nil, resourceError(ctx, logger, "bulk-delete-munki-packages", munkiPackageLabel, err)
		}
		return &struct{}{}, nil
	})
}
