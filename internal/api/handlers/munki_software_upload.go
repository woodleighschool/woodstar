package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type munkiSoftwareUploadInput struct {
	SoftwareID int64 `path:"id"`
	Body       MunkiUploadRequest
}

type munkiSoftwareConfirmInput struct {
	SoftwareID int64 `path:"id"`
	ObjectID   int64 `path:"object_id"`
}

type munkiIconObjectsInput struct {
	ListQueryInput
}

type munkiIconObjectsOutput struct {
	Body Page[MunkiObjectView]
}

func registerIconRoutes(
	api huma.API,
	software *munkisoftware.Store,
	objects *storage.ObjectStore,
	presigner storage.Presigner,
	logger *slog.Logger,
) {
	registerCreateSoftwareIconRoute(api, software, objects, presigner, logger)
	registerConfirmSoftwareIconRoute(api, software, objects, presigner, logger)
	registerListMunkiIconsRoute(api, objects, presigner, logger)
}

func registerCreateSoftwareIconRoute(
	api huma.API,
	software *munkisoftware.Store,
	objects *storage.ObjectStore,
	presigner storage.Presigner,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-software-icon-upload",
		Method:        http.MethodPost,
		Path:          munkiSoftwareIDPath + "/icon",
		Tags:          []string{munkiTag},
		Summary:       "Create an icon upload for Munki software",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiSoftwareUploadInput) (*munkiUploadOutput, error) {
		if _, err := software.GetByID(ctx, input.SoftwareID); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"create-munki-software-icon-upload",
				munkiupload.Label,
				err,
				"software_id",
				input.SoftwareID,
			)
		}
		obj, target, err := munkiupload.Create(
			ctx,
			objects,
			presigner,
			munkiupload.IconObjectPrefix,
			input.Body.Filename,
			input.Body.ContentType,
		)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"create-munki-software-icon-upload",
				munkiupload.Label,
				err,
				"software_id",
				input.SoftwareID,
			)
		}
		return munkiUploadOutputFromTarget(obj, target), nil
	})
}

func registerConfirmSoftwareIconRoute(
	api huma.API,
	software *munkisoftware.Store,
	objects *storage.ObjectStore,
	presigner storage.Presigner,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "confirm-munki-software-icon-upload",
		Method:        http.MethodPost,
		Path:          munkiSoftwareIDPath + "/icon/{object_id}/confirm",
		Tags:          []string{munkiTag},
		Summary:       "Confirm and attach an icon upload to Munki software",
		DefaultStatus: http.StatusOK,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiSoftwareConfirmInput) (*munkiObjectOutput, error) {
		obj, err := munkiupload.Confirm(
			ctx,
			objects,
			munkiupload.IconObjectPrefix,
			input.ObjectID,
			func(objectID int64) error {
				return software.SetIcon(ctx, input.SoftwareID, objectID)
			},
		)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"confirm-munki-software-icon-upload",
				munkiupload.Label,
				err,
				"software_id", input.SoftwareID,
				"object_id", input.ObjectID,
			)
		}
		view, err := munkiObjectViewWithContentURL(ctx, presigner, *obj)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"confirm-munki-software-icon-upload",
				munkiupload.Label,
				err,
				"software_id", input.SoftwareID,
				"object_id", input.ObjectID,
			)
		}
		return &munkiObjectOutput{Body: view}, nil
	})
}

func registerListMunkiIconsRoute(
	api huma.API,
	objects *storage.ObjectStore,
	presigner storage.Presigner,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-icons",
		Method:      http.MethodGet,
		Path:        "/api/munki/icons",
		Tags:        []string{munkiTag},
		Summary:     "List uploaded Munki icons",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *munkiIconObjectsInput) (*munkiIconObjectsOutput, error) {
		rows, count, err := objects.ListByPrefix(ctx, munkiupload.IconObjectPrefix, input.ListQueryInput.Params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-munki-icons", "Munki icon", err)
		}
		views := make([]MunkiObjectView, len(rows))
		for i, row := range rows {
			view, err := munkiObjectViewWithContentURL(ctx, presigner, row)
			if err != nil {
				return nil, resourceError(ctx, logger, "list-munki-icons", "Munki icon", err, "object_id", row.ID)
			}
			views[i] = view
		}
		return &munkiIconObjectsOutput{Body: Page[MunkiObjectView]{
			Items: views,
			Count: int32(count),
		}}, nil
	})
}
