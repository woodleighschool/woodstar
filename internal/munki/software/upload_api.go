package software

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type munkiSoftwareUploadInput struct {
	SoftwareID int64 `path:"id"`
	Body       munkiupload.MunkiUploadRequest
}

type munkiSoftwareConfirmInput struct {
	SoftwareID int64 `path:"id"`
	ObjectID   int64 `path:"object_id"`
}

type munkiIconObjectsInput struct {
	apitypes.ListQueryInput
}

type munkiIconObjectsOutput struct {
	Body apitypes.Page[munkiupload.MunkiObjectView]
}

type munkiIconContentInput struct {
	ObjectID int64 `path:"object_id"`
}

func registerIconRoutes(api huma.API, software *Store, objects *storage.ObjectStore, store storage.Store) {
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
	}, func(ctx context.Context, input *munkiSoftwareUploadInput) (*munkiupload.UploadOutput, error) {
		if _, err := software.GetByID(ctx, input.SoftwareID); err != nil {
			return nil, apitypes.ResourceMutationError(munkiupload.Label, err)
		}
		out, err := munkiupload.Create(ctx, objects, store, IconObjectPrefix, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiupload.Label, err)
		}
		return out, nil
	})

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
	}, func(ctx context.Context, input *munkiSoftwareConfirmInput) (*munkiupload.ObjectOutput, error) {
		out, err := munkiupload.Confirm(
			ctx,
			objects,
			IconObjectPrefix,
			input.ObjectID,
			func(objectID int64) error {
				return software.SetIcon(ctx, input.SoftwareID, objectID)
			},
		)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiupload.Label, err)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-munki-icons",
		Method:      http.MethodGet,
		Path:        "/api/munki/icons",
		Tags:        []string{munkiTag},
		Summary:     "List uploaded Munki icons",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *munkiIconObjectsInput) (*munkiIconObjectsOutput, error) {
		rows, count, err := objects.ListByPrefix(ctx, IconObjectPrefix, input.ListQueryInput.Params())
		if err != nil {
			return nil, apitypes.ResourceMutationError("Munki icon", err)
		}
		views := make([]munkiupload.MunkiObjectView, len(rows))
		for i, row := range rows {
			views[i] = iconView(row)
		}
		return &munkiIconObjectsOutput{Body: apitypes.Page[munkiupload.MunkiObjectView]{
			Items: views,
			Count: int32(count),
		}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-munki-icon-content",
		Method:      http.MethodGet,
		Path:        "/api/munki/icons/{object_id}/content",
		Tags:        []string{munkiTag},
		Summary:     "Serve an uploaded Munki icon",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiIconContentInput) (*huma.StreamResponse, error) {
		obj, err := objects.GetByID(ctx, input.ObjectID)
		if err != nil {
			return nil, apitypes.ResourceMutationError("Munki icon", err)
		}
		if obj.Prefix != IconObjectPrefix {
			return nil, apitypes.ResourceMutationError(
				"Munki icon",
				fmt.Errorf("%w: object is not a Munki icon", dbutil.ErrInvalidInput),
			)
		}
		return &huma.StreamResponse{Body: func(hctx huma.Context) {
			storage.ServeContent(hctx, store, obj.Key())
		}}, nil
	})
}

func iconView(o storage.Object) munkiupload.MunkiObjectView {
	view := munkiupload.ViewObject(o)
	view.ContentURL = fmt.Sprintf("/api/munki/icons/%d/content", o.ID)
	return view
}
