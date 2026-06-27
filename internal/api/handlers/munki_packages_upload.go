package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

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
	packageStore *packages.Store,
	objects *storage.ObjectStore,
	store storage.Presigner,
	notifier desiredNotifier,
	logger *slog.Logger,
) {
	objectPath := munkiPackagePath + "/{id}/installer"
	registerCreatePackageInstallerRoute(api, packageStore, objects, store, objectPath, logger)
	registerConfirmPackageInstallerRoute(api, packageStore, objects, store, notifier, objectPath, logger)
	registerDeletePackageInstallerRoute(api, packageStore, notifier, objectPath, logger)
}

func registerCreatePackageInstallerRoute(
	api huma.API,
	packageStore *packages.Store,
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
	packageStore *packages.Store,
	objects *storage.ObjectStore,
	store storage.Presigner,
	notifier desiredNotifier,
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
		obj, err := munkiupload.Confirm(
			ctx,
			objects,
			packages.ObjectPrefix,
			input.ObjectID,
			func(objectID int64) error {
				return packageStore.SetInstallerObject(ctx, input.PackageID, objectID)
			},
		)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"confirm-munki-package-installer-upload",
				munkiupload.Label,
				err,
				"package_id", input.PackageID,
				"object_id", input.ObjectID,
			)
		}
		notifyDesired(notifier)
		view, err := munkiObjectViewWithContentURL(ctx, store, *obj)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"confirm-munki-package-installer-upload",
				munkiupload.Label,
				err,
				"package_id", input.PackageID,
				"object_id", input.ObjectID,
			)
		}
		return &munkiObjectOutput{Body: view}, nil
	})
}

func registerDeletePackageInstallerRoute(
	api huma.API,
	packageStore *packages.Store,
	notifier desiredNotifier,
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
		notifyDesired(notifier)
		return nil, nil
	})
}
