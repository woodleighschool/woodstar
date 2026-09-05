package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

const munkiPackageInstallerPath = "/api/munki/package-installers"

type munkiPackageInstallerCreateInput struct {
	Body MunkiPackageInstallerUploadRequest
}

type munkiPackageInstallerInput struct {
	ID int64 `path:"id"`
}

type munkiPackageInstallerPartInput struct {
	ID         int64 `path:"id"`
	PartNumber int32 `path:"part_number" minimum:"1" maximum:"10000"`
}

type munkiPackageInstallerCompleteInput struct {
	ID   int64 `path:"id"`
	Body MunkiMultipartCompleteRequest
}

type munkiMultipartPartOutput struct {
	Body bloby.UploadTarget
}

func registerPackageInstallerRoutes(
	humaAPI huma.API,
	longRunningAPI huma.API,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	registerCreatePackageInstallerUploadRoute(humaAPI, objects, logger)
	registerCompletePackageInstallerUploadRoute(longRunningAPI, objects, logger)
	registerDeletePackageInstallerUploadRoute(humaAPI, objects, logger)
	registerSignPackageInstallerPartRoute(humaAPI, objects, logger)
	registerCompletePackageInstallerMultipartRoute(longRunningAPI, objects, logger)
}

func registerCreatePackageInstallerUploadRoute(
	humaAPI huma.API,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-munki-package-installer-upload",
		Method:        http.MethodPost,
		Path:          munkiPackageInstallerPath,
		Tags:          []string{api.TagMunkiPackageInstallers},
		Summary:       "Create a package installer upload",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, func(
		ctx context.Context,
		input *munkiPackageInstallerCreateInput,
	) (*munkiUploadOutput, error) {
		object, action, err := objects.Begin(
			ctx,
			packages.ObjectPrefix,
			input.Body.Filename,
			input.Body.SizeBytes,
		)
		if err != nil {
			return nil, api.ResourceError(
				ctx, logger, "create-munki-package-installer-upload", munkiUploadLabel, err,
			)
		}
		return newMunkiUploadOutput(object, action), nil
	})
}

func registerCompletePackageInstallerUploadRoute(
	humaAPI huma.API,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "complete-munki-package-installer-upload",
		Method:      http.MethodPut,
		Path:        munkiPackageInstallerPath + "/{id}",
		Tags:        []string{api.TagMunkiPackageInstallers},
		Summary:     "Complete a package installer upload",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageInstallerInput) (*munkiObjectOutput, error) {
		object, err := finalizeMunkiUpload(ctx, objects, packages.ObjectPrefix, input.ID)
		if err != nil {
			return nil, api.ResourceError(
				ctx, logger, "complete-munki-package-installer-upload", munkiUploadLabel, err,
				"object_id", input.ID,
			)
		}
		view := munkiObjectView(*object, contentURL(munkiPackageInstallerPath, object.ID))
		return &munkiObjectOutput{Body: view}, nil
	})
}

func registerDeletePackageInstallerUploadRoute(
	humaAPI huma.API,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "delete-munki-package-installer-upload",
		Method:        http.MethodDelete,
		Path:          munkiPackageInstallerPath + "/{id}",
		Tags:          []string{api.TagMunkiPackageInstallers},
		Summary:       "Delete a package installer upload",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *munkiPackageInstallerInput) (*struct{}, error) {
		if err := objects.Delete(ctx, input.ID, packages.ObjectPrefix); err != nil {
			return nil, api.ResourceError(
				ctx, logger, "delete-munki-package-installer-upload", munkiUploadLabel, err,
				"object_id", input.ID,
			)
		}
		return &struct{}{}, nil
	})
}

func registerSignPackageInstallerPartRoute(
	humaAPI huma.API,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "sign-munki-package-installer-part",
		Method:      http.MethodPost,
		Path:        munkiPackageInstallerPath + "/{id}/multipart/parts/{part_number}",
		Tags:        []string{api.TagMunkiPackageInstallers},
		Summary:     "Sign a multipart upload part",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageInstallerPartInput) (*munkiMultipartPartOutput, error) {
		target, err := objects.PresignMultipartPart(
			ctx, input.ID, packages.ObjectPrefix, input.PartNumber,
		)
		if err != nil {
			return nil, api.ResourceError(
				ctx, logger, "sign-munki-package-installer-part", munkiUploadLabel, err,
				"object_id", input.ID, "part_number", input.PartNumber,
			)
		}
		return &munkiMultipartPartOutput{Body: target}, nil
	})
}

func registerCompletePackageInstallerMultipartRoute(
	humaAPI huma.API,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "complete-munki-package-installer-multipart",
		Method:        http.MethodPut,
		Path:          munkiPackageInstallerPath + "/{id}/multipart",
		Tags:          []string{api.TagMunkiPackageInstallers},
		Summary:       "Complete a multipart upload",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageInstallerCompleteInput) (*struct{}, error) {
		if err := objects.CompleteMultipart(ctx, input.ID, packages.ObjectPrefix, input.Body.Parts); err != nil {
			return nil, api.ResourceError(
				ctx, logger, "complete-munki-package-installer-multipart", munkiUploadLabel, err,
				"object_id", input.ID,
			)
		}
		return &struct{}{}, nil
	})
}
