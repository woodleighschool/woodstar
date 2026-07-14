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
	ID   int64 `path:"id"`
	Body MunkiUploadRequest
}

type munkiSoftwareConfirmInput struct {
	ID       int64 `path:"id"`
	ObjectID int64 `path:"object_id"`
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
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiSoftwareUploadInput) (*munkiUploadOutput, error) {
		if _, err := software.GetByID(ctx, input.ID); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"create-munki-software-icon-upload",
				munkiupload.Label,
				err,
				"software_id",
				input.ID,
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
				input.ID,
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
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiSoftwareConfirmInput) (*munkiObjectOutput, error) {
		return confirmMunkiObjectUpload(
			ctx,
			objects,
			presigner,
			logger,
			munkiUploadConfirm{
				Operation: "confirm-munki-software-icon-upload",
				Prefix:    munkiupload.IconObjectPrefix,
				ObjectID:  input.ObjectID,
				Attach: func(objectID int64) error {
					return software.SetIcon(ctx, input.ID, objectID)
				},
				Attrs: []any{"software_id", input.ID},
			},
		)
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
	}, func(ctx context.Context, input *munkiIconObjectsInput) (*munkiIconObjectsOutput, error) {
		rows, count, err := objects.ListByPrefix(ctx, munkiupload.IconObjectPrefix, input.ListQueryInput.params())
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
			Count: count,
		}}, nil
	})
}
