package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const munkiPackageInstallerPath = "/api/munki/package-installers"

type munkiPackageInstallerCreateInput struct {
	Body MunkiUploadRequest
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

type munkiMultipartUploadOutput struct {
	Body MunkiMultipartUpload
}

type munkiMultipartPartOutput struct {
	Body MunkiMultipartPartTarget
}

func registerPackageInstallerRoutes(
	api huma.API,
	longRunningAPI huma.API,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	registerCreatePackageInstallerUploadRoute(api, ingestor, logger)
	registerCompletePackageInstallerUploadRoute(longRunningAPI, ingestor, logger)
	registerDeletePackageInstallerUploadRoute(api, ingestor, logger)
	registerCreatePackageInstallerMultipartRoute(api, ingestor, logger)
	registerSignPackageInstallerPartRoute(api, ingestor, logger)
	registerCompletePackageInstallerMultipartRoute(longRunningAPI, ingestor, logger)
}

func registerCreatePackageInstallerUploadRoute(
	api huma.API,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-package-installer-upload",
		Method:        http.MethodPost,
		Path:          munkiPackageInstallerPath,
		Tags:          []string{munkiPackageInstallersTag},
		Summary:       "Create a package installer upload",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, func(
		ctx context.Context,
		input *munkiPackageInstallerCreateInput,
	) (*munkiPackageInstallerUploadOutput, error) {
		object, action, err := ingestor.Begin(
			ctx,
			packages.ObjectPrefix,
			input.Body.Filename,
		)
		if err != nil {
			return nil, resourceError(
				ctx, logger, "create-munki-package-installer-upload", munkiUploadLabel, err,
			)
		}
		switch action := action.(type) {
		case storage.DirectUploadAction:
			return newMunkiPackageInstallerDirectUploadOutput(object, action.Target), nil
		case storage.MultipartUploadAction:
			return newMunkiPackageInstallerMultipartUploadOutput(object), nil
		default:
			return nil, resourceError(
				ctx, logger, "create-munki-package-installer-upload", munkiUploadLabel,
				fmt.Errorf("storage returned unsupported upload action %T", action),
			)
		}
	})
}

func registerCompletePackageInstallerUploadRoute(
	api huma.API,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "complete-munki-package-installer-upload",
		Method:      http.MethodPut,
		Path:        munkiPackageInstallerPath + "/{id}",
		Tags:        []string{munkiPackageInstallersTag},
		Summary:     "Complete a package installer upload",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageInstallerInput) (*munkiObjectOutput, error) {
		object, err := finalizeMunkiUpload(ctx, ingestor, packages.ObjectPrefix, input.ID)
		if err != nil {
			return nil, resourceError(
				ctx, logger, "complete-munki-package-installer-upload", munkiUploadLabel, err,
				"object_id", input.ID,
			)
		}
		view := munkiObjectView(*object, munkiPackageInstallerContentURL(object.ID))
		return &munkiObjectOutput{Body: view}, nil
	})
}

func registerDeletePackageInstallerUploadRoute(
	api huma.API,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "delete-munki-package-installer-upload",
		Method:        http.MethodDelete,
		Path:          munkiPackageInstallerPath + "/{id}",
		Tags:          []string{munkiPackageInstallersTag},
		Summary:       "Delete a package installer upload",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *munkiPackageInstallerInput) (*struct{}, error) {
		if err := ingestor.Delete(ctx, input.ID, packages.ObjectPrefix); err != nil {
			return nil, resourceError(
				ctx, logger, "delete-munki-package-installer-upload", munkiUploadLabel, err,
				"object_id", input.ID,
			)
		}
		return &struct{}{}, nil
	})
}

func registerCreatePackageInstallerMultipartRoute(
	api huma.API,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "create-munki-package-installer-multipart",
		Method:      http.MethodPost,
		Path:        munkiPackageInstallerPath + "/{id}/multipart",
		Tags:        []string{munkiPackageInstallersTag},
		Summary:     "Create a multipart upload",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageInstallerInput) (*munkiMultipartUploadOutput, error) {
		upload, err := ingestor.CreateMultipart(ctx, input.ID, packages.ObjectPrefix)
		if err != nil {
			return nil, resourceError(
				ctx, logger, "create-munki-package-installer-multipart", munkiUploadLabel, err,
				"object_id", input.ID,
			)
		}
		return &munkiMultipartUploadOutput{Body: MunkiMultipartUpload{
			UploadID: upload.UploadID,
			Key:      upload.Key,
		}}, nil
	})
}

func registerSignPackageInstallerPartRoute(
	api huma.API,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "sign-munki-package-installer-part",
		Method:      http.MethodPost,
		Path:        munkiPackageInstallerPath + "/{id}/multipart/parts/{part_number}",
		Tags:        []string{munkiPackageInstallersTag},
		Summary:     "Sign a multipart upload part",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageInstallerPartInput) (*munkiMultipartPartOutput, error) {
		target, err := ingestor.PresignMultipartPart(
			ctx, input.ID, packages.ObjectPrefix, input.PartNumber,
		)
		if err != nil {
			return nil, resourceError(
				ctx, logger, "sign-munki-package-installer-part", munkiUploadLabel, err,
				"object_id", input.ID, "part_number", input.PartNumber,
			)
		}
		return &munkiMultipartPartOutput{Body: MunkiMultipartPartTarget{
			UploadURL: target.URL,
			Method:    target.Method,
			Headers:   target.Headers,
		}}, nil
	})
}

func registerCompletePackageInstallerMultipartRoute(
	api huma.API,
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "complete-munki-package-installer-multipart",
		Method:        http.MethodPut,
		Path:          munkiPackageInstallerPath + "/{id}/multipart",
		Tags:          []string{munkiPackageInstallersTag},
		Summary:       "Complete a multipart upload",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiPackageInstallerCompleteInput) (*struct{}, error) {
		parts := make([]storage.CompletedPart, len(input.Body.Parts))
		for i, part := range input.Body.Parts {
			parts[i] = storage.CompletedPart{PartNumber: part.PartNumber, ETag: part.ETag}
		}
		if err := ingestor.CompleteMultipart(ctx, input.ID, packages.ObjectPrefix, parts); err != nil {
			return nil, resourceError(
				ctx, logger, "complete-munki-package-installer-multipart", munkiUploadLabel, err,
				"object_id", input.ID,
			)
		}
		return &struct{}{}, nil
	})
}
