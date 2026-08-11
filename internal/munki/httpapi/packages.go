package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
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
	api.ListQueryInput

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

type munkiPackageListOutput struct {
	Body api.Page[packages.Package]
}

type munkiPackageOutput struct {
	Body packages.Package
}

func (input munkiPackageListInput) params() packages.PackageListParams {
	return packages.PackageListParams{
		ListParams:     input.Params(),
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
	humaAPI huma.API,
	longRunningAPI huma.API,
	store *munki.PackageService,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	registerListMunkiPackages(humaAPI, store, logger)
	registerCreateMunkiPackage(humaAPI, store, logger)
	registerGetMunkiPackage(humaAPI, store, logger)
	registerPutMunkiPackage(humaAPI, store, logger)
	registerBulkDeleteMunkiPackages(humaAPI, store, logger)
	registerPackageInstallerRoutes(humaAPI, longRunningAPI, ingestor, logger)
}

func registerListMunkiPackages(humaAPI huma.API, store *munki.PackageService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-munki-packages",
		Method:      http.MethodGet,
		Path:        munkiPackagePath,
		Tags:        []string{api.TagMunkiPackages},
		Summary:     "List packages",
	}, func(ctx context.Context, input *munkiPackageListInput) (*munkiPackageListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-munki-packages", munkiPackageLabel, err)
		}
		return &munkiPackageListOutput{
			Body: api.Page[packages.Package]{
				Items: rows,
				Count: count,
			},
		}, nil
	})
}

func registerCreateMunkiPackage(humaAPI huma.API, store *munki.PackageService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-munki-package",
		Method:        http.MethodPost,
		Path:          munkiPackagePath,
		Tags:          []string{api.TagMunkiPackages},
		Summary:       "Create a package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackageCreateInput) (*munkiPackageOutput, error) {
		pkg, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-munki-package", munkiPackageLabel, err)
		}
		return &munkiPackageOutput{Body: *pkg}, nil
	})
}

func registerGetMunkiPackage(humaAPI huma.API, store *munki.PackageService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-munki-package",
		Method:      http.MethodGet,
		Path:        munkiPackageIDPath,
		Tags:        []string{api.TagMunkiPackages},
		Summary:     "Get a package",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageGetInput) (*munkiPackageOutput, error) {
		pkg, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(
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

func registerPutMunkiPackage(humaAPI huma.API, store *munki.PackageService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-munki-package",
		Method:      http.MethodPut,
		Path:        munkiPackageIDPath,
		Tags:        []string{api.TagMunkiPackages},
		Summary:     "Update a package",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackagePutInput) (*munkiPackageOutput, error) {
		pkg, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(
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

func registerBulkDeleteMunkiPackages(
	humaAPI huma.API,
	store *munki.PackageService,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "bulk-delete-munki-packages",
		Method:        http.MethodDelete,
		Path:          munkiPackagePath,
		Tags:          []string{api.TagMunkiPackages},
		Summary:       "Delete packages",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest, http.StatusConflict},
	}, func(ctx context.Context, input *api.DeleteManyInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.IDs); err != nil {
			return nil, api.ResourceError(ctx, logger, "bulk-delete-munki-packages", munkiPackageLabel, err)
		}
		return &struct{}{}, nil
	})
}
