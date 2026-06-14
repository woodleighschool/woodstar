package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	storageTag        = "Storage"
	objectsPath       = "/api/storage/objects"
	objectContentPath = "/api/storage/objects/{id}/content"
	objectConfirmPath = "/api/storage/objects/{id}/confirm"
	objectLabel       = "storage object"
)

// ObjectView is the API representation of a stored object.
type ObjectView struct {
	ID          int64   `json:"id"`
	Key         string  `json:"key"`
	Prefix      string  `json:"prefix"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	SizeBytes   *int64  `json:"size_bytes,omitempty"`
	SHA256      *string `json:"sha256,omitempty"`
	Available   bool    `json:"available"`
	ContentURL  string  `json:"content_url"`
}

func viewOf(o Object) ObjectView {
	return ObjectView{
		ID:          o.ID,
		Key:         o.Key(),
		Prefix:      o.Prefix,
		Filename:    o.Filename,
		ContentType: o.ContentType,
		SizeBytes:   o.SizeBytes,
		SHA256:      o.SHA256,
		Available:   o.Available(),
		ContentURL:  fmt.Sprintf("/api/storage/objects/%d/content", o.ID),
	}
}

// RegisterAdminRoutes mounts the generic storage object endpoints.
func RegisterAdminRoutes(api huma.API, objects *ObjectStore, store Store) {
	registerListObjects(api, objects)
	registerObjectContent(api, objects, store)
	registerConfirmObject(api, objects, store)
}

type listObjectsInput struct {
	apitypes.ListQueryInput
	Prefix string `query:"prefix"`
}

type listObjectsOutput struct {
	Body apitypes.Page[ObjectView]
}

func registerListObjects(api huma.API, objects *ObjectStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-storage-objects",
		Method:      http.MethodGet,
		Path:        objectsPath,
		Tags:        []string{storageTag},
		Summary:     "List storage objects under a prefix",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *listObjectsInput) (*listObjectsOutput, error) {
		rows, count, err := objects.ListByPrefix(ctx, strings.Trim(input.Prefix, "/"), input.ListQueryInput.Params())
		if err != nil {
			return nil, apitypes.ResourceMutationError(objectLabel, err)
		}
		views := make([]ObjectView, len(rows))
		for i, row := range rows {
			views[i] = viewOf(row)
		}
		return &listObjectsOutput{Body: apitypes.Page[ObjectView]{Items: views, Count: int32(count)}}, nil
	})
}

type objectContentInput struct {
	ID int64 `path:"id"`
}

// registerObjectContent serves an object's bytes: a redirect to a presigned URL
// when the backend supports it (S3), otherwise a direct stream (file).
func registerObjectContent(api huma.API, objects *ObjectStore, store Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-storage-object-content",
		Method:      http.MethodGet,
		Path:        objectContentPath,
		Tags:        []string{storageTag},
		Summary:     "Serve a storage object's content",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *objectContentInput) (*huma.StreamResponse, error) {
		obj, err := objects.GetByID(ctx, input.ID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(objectLabel, err)
		}
		return &huma.StreamResponse{Body: func(hctx huma.Context) {
			serveContent(hctx, store, obj.Key())
		}}, nil
	})
}

// serveContent redirects to a presigned URL (S3) or streams bytes from the
// backend (file).
func serveContent(ctx huma.Context, store Store, key string) {
	if presigner, ok := store.(Presigner); ok {
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
	if errors.Is(err, ErrObjectNotFound) {
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

type confirmObjectInput struct {
	ID   int64 `path:"id"`
	Body struct {
		SHA256 string `json:"sha256"`
	}
}

type objectOutput struct {
	Body ObjectView
}

func registerConfirmObject(api huma.API, objects *ObjectStore, store Store) {
	huma.Register(api, huma.Operation{
		OperationID: "confirm-storage-object",
		Method:      http.MethodPost,
		Path:        objectConfirmPath,
		Tags:        []string{storageTag},
		Summary:     "Confirm an uploaded storage object",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *confirmObjectInput) (*objectOutput, error) {
		obj, err := objects.GetByID(ctx, input.ID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(objectLabel, err)
		}
		info, err := store.Stat(ctx, obj.Key())
		if errors.Is(err, ErrObjectNotFound) {
			return nil, apitypes.ResourceMutationError(
				objectLabel,
				fmt.Errorf("%w: uploaded object does not exist", dbutil.ErrInvalidInput),
			)
		}
		if err != nil {
			return nil, err
		}
		confirmed, err := objects.Confirm(ctx, obj.ID, info.Size, info.ContentType, input.Body.SHA256)
		if err != nil {
			return nil, apitypes.ResourceMutationError(objectLabel, err)
		}
		return &objectOutput{Body: viewOf(*confirmed)}, nil
	})
}
