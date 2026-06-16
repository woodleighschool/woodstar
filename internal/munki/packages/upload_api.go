package packages

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type munkiPackageUploadInput struct {
	PackageID int64 `path:"id"`
	Body      munkiupload.MunkiUploadRequest
}

type munkiPackageObjectInput struct {
	PackageID int64 `path:"id"`
}

type munkiPackageConfirmInput struct {
	PackageID int64 `path:"id"`
	ObjectID  int64 `path:"object_id"`
}

func registerObjectRoutes(api huma.API, packages *Store, objects *storage.ObjectStore, store storage.Presigner) {
	objectPath := munkiPackagePath + "/{id}/installer"
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
	}, func(ctx context.Context, input *munkiPackageUploadInput) (*munkiupload.UploadOutput, error) {
		if _, err := packages.GetByID(ctx, input.PackageID); err != nil {
			return nil, apitypes.ResourceMutationError(munkiupload.Label, err)
		}
		out, err := munkiupload.Create(ctx, objects, store, ObjectPrefix, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiupload.Label, err)
		}
		return out, nil
	})

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
	}, func(ctx context.Context, input *munkiPackageConfirmInput) (*munkiupload.ObjectOutput, error) {
		out, err := munkiupload.Confirm(
			ctx,
			objects,
			ObjectPrefix,
			input.ObjectID,
			func(objectID int64) error {
				return packages.SetInstallerObject(ctx, input.PackageID, objectID)
			},
		)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiupload.Label, err)
		}
		return out, nil
	})

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
		if err := packages.ClearInstallerObject(ctx, input.PackageID); err != nil {
			return nil, apitypes.ResourceMutationError(munkiupload.Label, err)
		}
		return nil, nil
	})
}
