package adminapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	munkipackages "github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const munkiUploadLabel = "munki upload"

type munkiUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
}

type munkiUploadTarget struct {
	ObjectID  int64             `json:"object_id"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type munkiObjectView struct {
	ID          int64   `json:"id"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	SizeBytes   *int64  `json:"size_bytes,omitempty"`
	SHA256      *string `json:"sha256,omitempty"`
	ContentURL  string  `json:"content_url,omitempty"`
}

type munkiUploadOutput struct {
	Body munkiUploadTarget
}

type munkiObjectOutput struct {
	Body munkiObjectView
}

type munkiIconObjectsInput struct {
	apitypes.ListQueryInput
}

type munkiIconObjectsOutput struct {
	Body apitypes.Page[munkiObjectView]
}

type munkiPackageUploadInput struct {
	PackageID int64 `path:"id"`
	Body      munkiUploadRequest
}

type munkiPackageConfirmInput struct {
	PackageID int64 `path:"id"`
	ObjectID  int64 `path:"object_id"`
}

type munkiSoftwareUploadInput struct {
	SoftwareID int64 `path:"id"`
	Body       munkiUploadRequest
}

type munkiSoftwareConfirmInput struct {
	SoftwareID int64 `path:"id"`
	ObjectID   int64 `path:"object_id"`
}

type munkiIconContentInput struct {
	ObjectID int64 `path:"object_id"`
}

// registerMunkiUploadRoutes mounts Munki-owned object upload and picker routes.
func registerMunkiUploadRoutes(api huma.API, deps Dependencies) {
	munki := deps.Munki
	registerMunkiPackageUploadRoutes(api, munki)
	registerMunkiSoftwareUploadRoutes(api, munki)
	registerMunkiIconRoutes(api, munki)
}

func registerMunkiPackageUploadRoutes(api huma.API, munki MunkiDependencies) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-package-installer-upload",
		Method:        http.MethodPost,
		Path:          "/api/munki/packages/{id}/installer",
		Tags:          []string{"Munki"},
		Summary:       "Create an installer upload for a Munki package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiPackageUploadInput) (*munkiUploadOutput, error) {
		if _, err := munki.Packages.GetByID(ctx, input.PackageID); err != nil {
			return nil, apitypes.ResourceMutationError(munkiUploadLabel, err)
		}
		return createMunkiUpload(ctx, munki.Objects, munki.Store, munkipackages.ObjectPrefix, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "confirm-munki-package-installer-upload",
		Method:        http.MethodPost,
		Path:          "/api/munki/packages/{id}/installer/{object_id}/confirm",
		Tags:          []string{"Munki"},
		Summary:       "Confirm and attach an installer upload to a Munki package",
		DefaultStatus: http.StatusOK,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiPackageConfirmInput) (*munkiObjectOutput, error) {
		return confirmMunkiUpload(
			ctx,
			munki.Objects,
			munkipackages.ObjectPrefix,
			input.ObjectID,
			func(objectID int64) error {
				return munki.Packages.SetInstallerObject(ctx, input.PackageID, objectID)
			},
		)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-package-uninstaller-upload",
		Method:        http.MethodPost,
		Path:          "/api/munki/packages/{id}/uninstaller",
		Tags:          []string{"Munki"},
		Summary:       "Create an uninstaller upload for a Munki package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiPackageUploadInput) (*munkiUploadOutput, error) {
		if _, err := munki.Packages.GetByID(ctx, input.PackageID); err != nil {
			return nil, apitypes.ResourceMutationError(munkiUploadLabel, err)
		}
		return createMunkiUpload(ctx, munki.Objects, munki.Store, munkipackages.ObjectPrefix, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "confirm-munki-package-uninstaller-upload",
		Method:        http.MethodPost,
		Path:          "/api/munki/packages/{id}/uninstaller/{object_id}/confirm",
		Tags:          []string{"Munki"},
		Summary:       "Confirm and attach an uninstaller upload to a Munki package",
		DefaultStatus: http.StatusOK,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiPackageConfirmInput) (*munkiObjectOutput, error) {
		return confirmMunkiUpload(
			ctx,
			munki.Objects,
			munkipackages.ObjectPrefix,
			input.ObjectID,
			func(objectID int64) error {
				return munki.Packages.SetUninstallerObject(ctx, input.PackageID, objectID)
			},
		)
	})
}

func registerMunkiSoftwareUploadRoutes(api huma.API, munki MunkiDependencies) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-software-icon-upload",
		Method:        http.MethodPost,
		Path:          "/api/munki/software/{id}/icon",
		Tags:          []string{"Munki"},
		Summary:       "Create an icon upload for Munki software",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiSoftwareUploadInput) (*munkiUploadOutput, error) {
		if _, err := munki.Software.GetByID(ctx, input.SoftwareID); err != nil {
			return nil, apitypes.ResourceMutationError(munkiUploadLabel, err)
		}
		return createMunkiUpload(ctx, munki.Objects, munki.Store, munkisoftware.IconObjectPrefix, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "confirm-munki-software-icon-upload",
		Method:        http.MethodPost,
		Path:          "/api/munki/software/{id}/icon/{object_id}/confirm",
		Tags:          []string{"Munki"},
		Summary:       "Confirm and attach an icon upload to Munki software",
		DefaultStatus: http.StatusOK,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *munkiSoftwareConfirmInput) (*munkiObjectOutput, error) {
		return confirmMunkiUpload(
			ctx,
			munki.Objects,
			munkisoftware.IconObjectPrefix,
			input.ObjectID,
			func(objectID int64) error {
				return munki.Software.SetIcon(ctx, input.SoftwareID, objectID)
			},
		)
	})
}

func registerMunkiIconRoutes(api huma.API, munki MunkiDependencies) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-icons",
		Method:      http.MethodGet,
		Path:        "/api/munki/icons",
		Tags:        []string{"Munki"},
		Summary:     "List uploaded Munki icons",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *munkiIconObjectsInput) (*munkiIconObjectsOutput, error) {
		rows, count, err := munki.Objects.ListByPrefix(
			ctx,
			munkisoftware.IconObjectPrefix,
			input.ListQueryInput.Params(),
		)
		if err != nil {
			return nil, apitypes.ResourceMutationError("Munki icon", err)
		}
		views := make([]munkiObjectView, len(rows))
		for i, row := range rows {
			views[i] = viewMunkiObject(row)
		}
		return &munkiIconObjectsOutput{Body: apitypes.Page[munkiObjectView]{Items: views, Count: int32(count)}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-munki-icon-content",
		Method:      http.MethodGet,
		Path:        "/api/munki/icons/{object_id}/content",
		Tags:        []string{"Munki"},
		Summary:     "Serve an uploaded Munki icon",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiIconContentInput) (*huma.StreamResponse, error) {
		obj, err := munki.Objects.GetByID(ctx, input.ObjectID)
		if err != nil {
			return nil, apitypes.ResourceMutationError("Munki icon", err)
		}
		if obj.Prefix != munkisoftware.IconObjectPrefix {
			return nil, apitypes.ResourceMutationError(
				"Munki icon",
				fmt.Errorf("%w: object is not a Munki icon", dbutil.ErrInvalidInput),
			)
		}
		return &huma.StreamResponse{Body: func(hctx huma.Context) {
			serveObjectContent(hctx, munki.Store, obj.Key())
		}}, nil
	})
}

func createMunkiUpload(
	ctx context.Context,
	objects *storage.ObjectStore,
	store storage.Store,
	prefix string,
	req munkiUploadRequest,
) (*munkiUploadOutput, error) {
	obj, err := objects.CreatePending(ctx, prefix, req.Filename, req.ContentType)
	if err != nil {
		return nil, apitypes.ResourceMutationError(munkiUploadLabel, err)
	}

	// Presigning backends (S3) hand the client a direct upload URL; backends
	// without one (file) take the bytes through woodstar's streaming receiver.
	presigner, ok := store.(storage.Presigner)
	if !ok {
		return &munkiUploadOutput{Body: munkiUploadTarget{
			ObjectID:  obj.ID,
			UploadURL: fmt.Sprintf("/api/objects/%d/content", obj.ID),
			Method:    http.MethodPut,
		}}, nil
	}
	target, err := presigner.PresignPut(ctx, obj.Key(), 0, storage.PutOptions{ContentType: req.ContentType})
	if err != nil {
		return nil, err
	}
	return &munkiUploadOutput{Body: munkiUploadTarget{
		ObjectID:  obj.ID,
		UploadURL: target.URL,
		Method:    target.Method,
		Headers:   target.Headers,
	}}, nil
}

func confirmMunkiUpload(
	ctx context.Context,
	objects *storage.ObjectStore,
	prefix string,
	objectID int64,
	attach func(objectID int64) error,
) (*munkiObjectOutput, error) {
	obj, err := objects.GetByID(ctx, objectID)
	if err != nil {
		return nil, apitypes.ResourceMutationError(munkiUploadLabel, err)
	}
	if obj.Prefix != prefix {
		return nil, apitypes.ResourceMutationError(
			munkiUploadLabel,
			fmt.Errorf("%w: object has the wrong Munki prefix", dbutil.ErrInvalidInput),
		)
	}
	confirmed, err := objects.ConfirmUploaded(ctx, objectID)
	if errors.Is(err, storage.ErrObjectNotFound) {
		return nil, apitypes.ResourceMutationError(
			munkiUploadLabel,
			fmt.Errorf("%w: uploaded object does not exist", dbutil.ErrInvalidInput),
		)
	}
	if err != nil {
		return nil, apitypes.ResourceMutationError(munkiUploadLabel, err)
	}
	if err := attach(confirmed.ID); err != nil {
		_ = objects.DeleteUnreferenced(ctx, confirmed.ID)
		return nil, apitypes.ResourceMutationError(munkiUploadLabel, err)
	}
	return &munkiObjectOutput{Body: viewMunkiObject(*confirmed)}, nil
}

func viewMunkiObject(o storage.Object) munkiObjectView {
	view := munkiObjectView{
		ID:          o.ID,
		Filename:    o.Filename,
		ContentType: o.ContentType,
		SizeBytes:   o.SizeBytes,
		SHA256:      o.SHA256,
	}
	if o.Prefix == munkisoftware.IconObjectPrefix {
		view.ContentURL = fmt.Sprintf("/api/munki/icons/%d/content", o.ID)
	}
	return view
}

func serveObjectContent(ctx huma.Context, store storage.Store, key string) {
	if presigner, ok := store.(storage.Presigner); ok {
		url, err := presigner.PresignGet(ctx.Context(), key, 0)
		if err != nil {
			ctx.SetStatus(http.StatusInternalServerError)
			return
		}
		ctx.SetHeader("Location", url)
		ctx.SetStatus(http.StatusFound)
		return
	}
	reader, info, err := store.Open(ctx.Context(), key)
	if errors.Is(err, storage.ErrObjectNotFound) {
		ctx.SetStatus(http.StatusNotFound)
		return
	}
	if err != nil {
		ctx.SetStatus(http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	if info.ContentType != "" {
		ctx.SetHeader("Content-Type", info.ContentType)
	}
	ctx.SetStatus(http.StatusOK)
	_, _ = io.Copy(ctx.BodyWriter(), reader)
}
