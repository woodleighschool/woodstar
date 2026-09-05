package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"

	"github.com/woodleighschool/goodies/bloby"
)

type munkiIconUploadInput struct {
	Body MunkiDirectUploadRequest
}

const munkiIconPath = "/api/munki/icons"

type munkiSoftwareIconPutInput struct {
	ID   int64 `path:"id"`
	Body MunkiObjectMutation
}

type munkiIconObjectsInput struct {
	api.ListQueryInput
}

type munkiIconObjectsOutput struct {
	Body api.Page[MunkiObjectView]
}

func registerIconRoutes(
	humaAPI huma.API,
	software *munkisoftware.Store,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	registerCreateIconUploadRoute(humaAPI, objects, logger)
	registerSetSoftwareIconRoute(humaAPI, software, objects, logger)
	registerListMunkiIconsRoute(humaAPI, objects, logger)
}

func registerCreateIconUploadRoute(
	humaAPI huma.API,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-munki-icon-upload",
		Method:        http.MethodPost,
		Path:          munkiIconPath,
		Tags:          []string{api.TagMunkiIcons},
		Summary:       "Create an icon upload",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *munkiIconUploadInput) (*munkiUploadOutput, error) {
		obj, target, err := objects.BeginDirect(
			ctx,
			munkisoftware.IconObjectPrefix,
			input.Body.Filename,
		)
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"create-munki-icon-upload",
				munkiUploadLabel,
				err,
			)
		}
		return newMunkiUploadOutput(obj, target), nil
	})
}

func registerSetSoftwareIconRoute(
	humaAPI huma.API,
	software *munkisoftware.Store,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "set-munki-software-icon",
		Method:        http.MethodPut,
		Path:          munkiSoftwareIDPath + "/icon",
		Tags:          []string{api.TagMunkiIcons},
		Summary:       "Set a software icon",
		DefaultStatus: http.StatusOK,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiSoftwareIconPutInput) (*munkiObjectOutput, error) {
		object, err := setMunkiObject(
			ctx,
			objects,
			munkisoftware.IconObjectPrefix,
			input.Body.ObjectID,
			func(objectID int64) error {
				return software.SetIcon(ctx, input.ID, objectID)
			},
		)
		if err != nil {
			return nil, api.ResourceError(
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
	humaAPI huma.API,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-munki-icons",
		Method:      http.MethodGet,
		Path:        munkiIconPath,
		Tags:        []string{api.TagMunkiIcons},
		Summary:     "List icons",
	}, func(ctx context.Context, input *munkiIconObjectsInput) (*munkiIconObjectsOutput, error) {
		params := input.Params()
		rows, count, err := objects.ListByPrefix(ctx, munkisoftware.IconObjectPrefix, bloby.ListOptions{
			Limit:  int(params.PageSize),
			Offset: int(params.PageIndex) * int(params.PageSize),
		})
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-munki-icons", "Munki icon", err)
		}
		views := make([]MunkiObjectView, len(rows))
		for i, row := range rows {
			views[i] = munkiObjectView(row, contentURL(munkiIconPath, row.ID))
		}
		return &munkiIconObjectsOutput{Body: api.Page[MunkiObjectView]{
			Items: views,
			Count: count,
		}}, nil
	})
}
