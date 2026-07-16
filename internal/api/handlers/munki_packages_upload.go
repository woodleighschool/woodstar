package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki"
	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type munkiPackageUploadInput struct {
	ID   int64 `path:"id"`
	Body MunkiUploadRequest
}

type munkiPackageObjectInput struct {
	ID int64 `path:"id"`
}

type munkiPackageInstallerPutInput struct {
	ID   int64 `path:"id"`
	Body MunkiObjectMutation
}

func registerObjectRoutes(
	api huma.API,
	packageService *munki.PackageService,
	objects *storage.ObjectStore,
	uploads *munkiupload.Service,
	logger *slog.Logger,
) {
	objectPath := munkiPackagePath + "/{id}/installer"
	registerCreatePackageInstallerRoute(api, packageService, uploads, objectPath, logger)
	registerSetPackageInstallerRoute(api, packageService, objects, uploads, objectPath, logger)
	registerDeletePackageInstallerRoute(api, packageService, objectPath, logger)
}

func registerCreatePackageInstallerRoute(
	api huma.API,
	packageService *munki.PackageService,
	uploads *munkiupload.Service,
	objectPath string,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-package-installer-upload",
		Method:        http.MethodPost,
		Path:          objectPath,
		Tags:          []string{munkiTag},
		Summary:       "Create an installer upload for a Munki package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiPackageUploadInput) (*munkiUploadOutput, error) {
		if _, err := packageService.GetByID(ctx, input.ID); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"create-munki-package-installer-upload",
				munkiUploadLabel,
				err,
				"package_id",
				input.ID,
			)
		}
		obj, target, err := uploads.Begin(ctx, packages.ObjectPrefix, input.Body.Filename)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"create-munki-package-installer-upload",
				munkiUploadLabel,
				err,
				"package_id",
				input.ID,
			)
		}
		return munkiUploadOutputFromTarget(obj, target), nil
	})
}

func registerSetPackageInstallerRoute(
	api huma.API,
	packageService *munki.PackageService,
	objects *storage.ObjectStore,
	uploads *munkiupload.Service,
	objectPath string,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "set-munki-package-installer",
		Method:        http.MethodPut,
		Path:          objectPath,
		Tags:          []string{munkiTag},
		Summary:       "Set the uploaded installer for a Munki package",
		DefaultStatus: http.StatusOK,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiPackageInstallerPutInput) (*munkiObjectOutput, error) {
		view, err := setMunkiObject(
			ctx,
			objects,
			uploads,
			packages.ObjectPrefix,
			input.Body.ObjectID,
			func(objectID int64) error {
				return packageService.SetInstallerObject(ctx, input.ID, objectID)
			},
		)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"set-munki-package-installer",
				munkiUploadLabel,
				err,
				"package_id", input.ID,
				"object_id", input.Body.ObjectID,
			)
		}
		return &munkiObjectOutput{Body: view}, nil
	})
}

func registerDeletePackageInstallerRoute(
	api huma.API,
	packageService *munki.PackageService,
	objectPath string,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "delete-munki-package-installer",
		Method:        http.MethodDelete,
		Path:          objectPath,
		Tags:          []string{munkiTag},
		Summary:       "Delete the installer file from a Munki package",
		DefaultStatus: http.StatusNoContent,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackageObjectInput) (*struct{}, error) {
		if err := packageService.ClearInstallerObject(ctx, input.ID); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"delete-munki-package-installer",
				munkiUploadLabel,
				err,
				"package_id",
				input.ID,
			)
		}
		return &struct{}{}, nil
	})
}
