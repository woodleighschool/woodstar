package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type munkiSoftwareUploadInput struct {
	ID   int64 `path:"id"`
	Body MunkiUploadRequest
}

const munkiIconPath = "/api/munki/icons"

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
	ingestor *storage.Ingestor,
	logger *slog.Logger,
) {
	registerCreateSoftwareIconRoute(api, software, ingestor, logger)
	registerSetSoftwareIconRoute(api, software, ingestor, logger)
	registerListMunkiIconsRoute(api, objects, logger)
}

func registerCreateSoftwareIconRoute(
	api huma.API,
	software *munkisoftware.Store,
	ingestor *storage.Ingestor,
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
	}, func(ctx context.Context, input *munkiSoftwareUploadInput) (*munkiDirectUploadOutput, error) {
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
		obj, target, err := ingestor.BeginDirect(
			ctx,
			munkisoftware.IconObjectPrefix,
			input.Body.Filename,
		)
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
		return newMunkiDirectUploadOutput(obj, target), nil
	})
}

func registerSetSoftwareIconRoute(
	api huma.API,
	software *munkisoftware.Store,
	ingestor *storage.Ingestor,
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
		object, err := setMunkiObject(
			ctx,
			ingestor,
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
		return &munkiObjectOutput{Body: munkiObjectView(
			*object,
			munkisoftware.IconURL(&object.ID),
		)}, nil
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
		Path:        munkiIconPath,
		Tags:        []string{munkiTag},
		Summary:     "List uploaded Munki icons",
	}, func(ctx context.Context, input *munkiIconObjectsInput) (*munkiIconObjectsOutput, error) {
		rows, count, err := objects.ListByPrefix(ctx, munkisoftware.IconObjectPrefix, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-munki-icons", "Munki icon", err)
		}
		views := make([]MunkiObjectView, len(rows))
		for i, row := range rows {
			views[i] = munkiObjectView(row, munkiIconContentURL(row.ID))
		}
		return &munkiIconObjectsOutput{Body: Page[MunkiObjectView]{
			Items: views,
			Count: count,
		}}, nil
	})
}
