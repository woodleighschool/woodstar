package packages

import (
	"context"
	"fmt"
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

type packageObjectRoute struct {
	name    string
	article string
	attach  func(ctx context.Context, packageID int64, objectID int64) error
	clear   func(ctx context.Context, packageID int64) error
}

func registerObjectRoutes(api huma.API, packages *Store, objects *storage.ObjectStore, store storage.Store) {
	for _, route := range []packageObjectRoute{
		{
			name:    "installer",
			article: "an",
			attach:  packages.SetInstallerObject,
			clear:   packages.ClearInstallerObject,
		},
		{
			name:    "uninstaller",
			article: "an",
			attach:  packages.SetUninstallerObject,
		},
	} {
		registerObjectRoute(api, packages, objects, store, route)
	}
}

func registerObjectRoute(
	api huma.API,
	packages *Store,
	objects *storage.ObjectStore,
	store storage.Store,
	route packageObjectRoute,
) {
	objectPath := fmt.Sprintf("%s/{id}/%s", munkiPackagePath, route.name)
	huma.Register(api, huma.Operation{
		OperationID:   fmt.Sprintf("create-munki-package-%s-upload", route.name),
		Method:        http.MethodPost,
		Path:          objectPath,
		Tags:          []string{munkiTag},
		Summary:       fmt.Sprintf("Create %s %s upload for a Munki package", route.article, route.name),
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
		OperationID:   fmt.Sprintf("confirm-munki-package-%s-upload", route.name),
		Method:        http.MethodPost,
		Path:          objectPath + "/{object_id}/confirm",
		Tags:          []string{munkiTag},
		Summary:       fmt.Sprintf("Confirm and attach %s %s upload to a Munki package", route.article, route.name),
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
				return route.attach(ctx, input.PackageID, objectID)
			},
		)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiupload.Label, err)
		}
		return out, nil
	})

	if route.clear == nil {
		return
	}
	huma.Register(api, huma.Operation{
		OperationID:   fmt.Sprintf("delete-munki-package-%s", route.name),
		Method:        http.MethodDelete,
		Path:          objectPath,
		Tags:          []string{munkiTag},
		Summary:       fmt.Sprintf("Delete the %s file from a Munki package", route.name),
		DefaultStatus: http.StatusNoContent,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiPackageObjectInput) (*struct{}, error) {
		if err := route.clear(ctx, input.PackageID); err != nil {
			return nil, apitypes.ResourceMutationError(munkiupload.Label, err)
		}
		return nil, nil
	})
}
