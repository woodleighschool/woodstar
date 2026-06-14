package adminapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const munkiUploadLabel = "munki upload"

type munkiUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
}

type munkiUploadTarget struct {
	ObjectID  int64             `json:"object_id"`
	Key       string            `json:"key"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type munkiUploadOutput struct {
	Body munkiUploadTarget
}

type munkiPackageUploadInput struct {
	PackageID int64 `path:"id"`
	Body      munkiUploadRequest
}

type munkiSoftwareUploadInput struct {
	SoftwareID int64 `path:"id"`
	Body       munkiUploadRequest
}

// registerMunkiUploadRoutes mounts the attach endpoints that create a pending
// storage object, link it to a munki resource, and hand back an upload target.
func registerMunkiUploadRoutes(api huma.API, deps Dependencies) {
	munki := deps.Munki

	huma.Register(api, huma.Operation{
		OperationID:   "attach-munki-package-installer",
		Method:        http.MethodPost,
		Path:          "/api/munki/packages/{id}/installer",
		Tags:          []string{"Munki"},
		Summary:       "Attach an installer upload to a Munki package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *munkiPackageUploadInput) (*munkiUploadOutput, error) {
		return attachMunkiUpload(
			ctx,
			munki.Objects,
			munki.Store,
			"munki/packages",
			input.Body,
			func(objectID int64) error {
				return munki.Packages.SetInstallerObject(ctx, input.PackageID, objectID)
			},
		)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "attach-munki-package-uninstaller",
		Method:        http.MethodPost,
		Path:          "/api/munki/packages/{id}/uninstaller",
		Tags:          []string{"Munki"},
		Summary:       "Attach an uninstaller upload to a Munki package",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *munkiPackageUploadInput) (*munkiUploadOutput, error) {
		return attachMunkiUpload(
			ctx,
			munki.Objects,
			munki.Store,
			"munki/packages",
			input.Body,
			func(objectID int64) error {
				return munki.Packages.SetUninstallerObject(ctx, input.PackageID, objectID)
			},
		)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "attach-munki-software-icon",
		Method:        http.MethodPost,
		Path:          "/api/munki/software/{id}/icon",
		Tags:          []string{"Munki"},
		Summary:       "Attach an icon upload to Munki software",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *munkiSoftwareUploadInput) (*munkiUploadOutput, error) {
		return attachMunkiUpload(
			ctx,
			munki.Objects,
			munki.Store,
			"munki/icons",
			input.Body,
			func(objectID int64) error {
				return munki.Software.SetIcon(ctx, input.SoftwareID, objectID)
			},
		)
	})
}

func attachMunkiUpload(
	ctx context.Context,
	objects *storage.ObjectStore,
	store storage.Store,
	prefix string,
	req munkiUploadRequest,
	link func(objectID int64) error,
) (*munkiUploadOutput, error) {
	obj, err := objects.CreatePending(ctx, prefix, req.Filename, req.ContentType)
	if err != nil {
		return nil, apitypes.ResourceMutationError(munkiUploadLabel, err)
	}
	if err := link(obj.ID); err != nil {
		return nil, apitypes.ResourceMutationError(munkiUploadLabel, err)
	}
	presigner, ok := store.(storage.Presigner)
	if !ok {
		return nil, huma.Error503ServiceUnavailable("uploads require a presigning backend")
	}
	target, err := presigner.PresignPut(ctx, obj.Key(), 0, storage.PutOptions{ContentType: req.ContentType})
	if err != nil {
		return nil, err
	}
	return &munkiUploadOutput{Body: munkiUploadTarget{
		ObjectID:  obj.ID,
		Key:       obj.Key(),
		UploadURL: target.URL,
		Method:    target.Method,
		Headers:   target.Headers,
	}}, nil
}
