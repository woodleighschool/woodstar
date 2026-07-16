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

type munkiSoftwareIconPutInput struct {
	ID   int64 `path:"id"`
	Body MunkiObjectMutation
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
	uploads *munkiupload.Service,
	logger *slog.Logger,
) {
	registerCreateSoftwareIconRoute(api, software, uploads, logger)
	registerSetSoftwareIconRoute(api, software, objects, uploads, logger)
	registerListMunkiIconsRoute(api, objects, logger)
}

func registerCreateSoftwareIconRoute(
	api huma.API,
	software *munkisoftware.Store,
	uploads *munkiupload.Service,
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
				munkiUploadLabel,
				err,
				"software_id",
				input.ID,
			)
		}
		obj, target, err := uploads.Begin(ctx, munkisoftware.IconObjectPrefix, input.Body.Filename)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"create-munki-software-icon-upload",
				munkiUploadLabel,
				err,
				"software_id",
				input.ID,
			)
		}
		return munkiUploadOutputFromTarget(obj, target), nil
	})
}

func registerSetSoftwareIconRoute(
	api huma.API,
	software *munkisoftware.Store,
	objects *storage.ObjectStore,
	uploads *munkiupload.Service,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "set-munki-software-icon",
		Method:        http.MethodPut,
		Path:          munkiSoftwareIDPath + "/icon",
		Tags:          []string{munkiTag},
		Summary:       "Set the uploaded icon for Munki software",
		DefaultStatus: http.StatusOK,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiSoftwareIconPutInput) (*munkiObjectOutput, error) {
		view, err := setMunkiObject(
			ctx,
			objects,
			uploads,
			munkisoftware.IconObjectPrefix,
			input.Body.ObjectID,
			func(objectID int64) error {
				return software.SetIcon(ctx, input.ID, objectID)
			},
		)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"set-munki-software-icon",
				munkiUploadLabel,
				err,
				"software_id", input.ID,
				"object_id", input.Body.ObjectID,
			)
		}
		return &munkiObjectOutput{Body: view}, nil
	})
}

func registerListMunkiIconsRoute(
	api huma.API,
	objects *storage.ObjectStore,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-icons",
		Method:      http.MethodGet,
		Path:        "/api/munki/icons",
		Tags:        []string{munkiTag},
		Summary:     "List uploaded Munki icons",
	}, func(ctx context.Context, input *munkiIconObjectsInput) (*munkiIconObjectsOutput, error) {
		rows, count, err := objects.ListByPrefix(ctx, munkisoftware.IconObjectPrefix, input.ListQueryInput.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-munki-icons", "Munki icon", err)
		}
		views := make([]MunkiObjectView, len(rows))
		for i, row := range rows {
			view, err := munkiObjectViewWithContentURL(ctx, objects, row)
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
