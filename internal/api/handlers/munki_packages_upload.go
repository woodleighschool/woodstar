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
	PackageID int64 `path:"id"`
	Body      MunkiUploadRequest
}

type munkiPackageObjectInput struct {
	PackageID int64 `path:"id"`
}

type munkiPackageConfirmInput struct {
	PackageID int64 `path:"id"`
	ObjectID  int64 `path:"object_id"`
}

func registerObjectRoutes(
	api huma.API,
	packageStore *munki.PackageService,
	objects *storage.ObjectStore,
	store storage.Presigner,
	logger *slog.Logger,
) {
	objectPath := munkiPackagePath + "/{id}/installer"
	registerCreatePackageInstallerRoute(api, packageStore, objects, store, objectPath, logger)
	registerConfirmPackageInstallerRoute(api, packageStore, objects, store, objectPath, logger)
	registerDeletePackageInstallerRoute(api, packageStore, objectPath, logger)
}

func registerCreatePackageInstallerRoute(
	api huma.API,
	packageStore *munki.PackageService,
	objects *storage.ObjectStore,
	store storage.Presigner,
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
			http.StatusUnauthorized,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiPackageUploadInput) (*munkiUploadOutput, error) {
		if _, err := packageStore.GetByID(ctx, input.PackageID); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"create-munki-package-installer-upload",
				munkiupload.Label,
				err,
				"package_id",
				input.PackageID,
			)
		}
		obj, target, err := munkiupload.Create(
			ctx,
			objects,
			store,
			packages.ObjectPrefix,
			input.Body.Filename,
			input.Body.ContentType,
		)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"create-munki-package-installer-upload",
				munkiupload.Label,
				err,
				"package_id",
				input.PackageID,
			)
		}
		return munkiUploadOutputFromTarget(obj, target), nil
	})
}

func registerConfirmPackageInstallerRoute(
	api huma.API,
	packageStore *munki.PackageService,
	objects *storage.ObjectStore,
	store storage.Presigner,
	objectPath string,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "confirm-munki-package-installer-upload",
		Method:        http.MethodPost,
		Path:          objectPath + "/{object_id}/confirm",
		Tags:          []string{munkiTag},
		Summary:       "Confirm and attach an installer upload to a Munki package",
		DefaultStatus: http.StatusOK,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiPackageConfirmInput) (*munkiObjectOutput, error) {
		return confirmMunkiObjectUpload(
			ctx,
			objects,
			store,
			logger,
			munkiUploadConfirm{
				Operation: "confirm-munki-package-installer-upload",
				Prefix:    packages.ObjectPrefix,
				ObjectID:  input.ObjectID,
				Attach: func(objectID int64) error {
					return packageStore.SetInstallerObject(ctx, input.PackageID, objectID)
				},
				Attrs: []any{"package_id", input.PackageID},
			},
		)
	})
}

func registerDeletePackageInstallerRoute(
	api huma.API,
	packageStore *munki.PackageService,
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
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackageObjectInput) (*struct{}, error) {
		if err := packageStore.ClearInstallerObject(ctx, input.PackageID); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"delete-munki-package-installer",
				munkiupload.Label,
				err,
				"package_id",
				input.PackageID,
			)
		}
		return nil, nil
	})
}
