package artifacts

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	munkiTag                 = "Munki"
	munkiArtifactPath        = "/api/munki/artifacts"
	munkiArtifactIDPath      = "/api/munki/artifacts/{id}"
	munkiArtifactContentPath = "/api/munki/artifacts/{id}/content"
	munkiArtifactUploadPath  = "/api/munki/artifact-uploads"
	munkiArtifactLabel       = "Munki artifact"
)

type munkiArtifactCreateInput struct {
	Body ArtifactMutation
}

type munkiArtifactListInput struct {
	apitypes.ListQueryInput
}

type munkiArtifactGetInput struct {
	ArtifactID int64 `path:"id"`
}

type munkiArtifactUploadInput struct {
	Body munkiArtifactUploadMutation
}

type munkiArtifactContentInput struct {
	ArtifactID int64 `path:"id"`
}

type munkiArtifactDeleteInput struct {
	ArtifactID int64 `path:"id"`
}

type munkiArtifactListOutput struct {
	Body apitypes.Page[Artifact]
}

type munkiArtifactOutput struct {
	Body Artifact
}

type munkiArtifactUploadOutput struct {
	Body munkiArtifactUpload
}

type munkiArtifactContentOutput struct {
	Status   int    `json:"-" default:"302"`
	Location string `                       header:"Location"`
}

type munkiArtifactUploadMutation struct {
	Kind        ArtifactKind `json:"kind"`
	Filename    string       `json:"filename"`
	DisplayName string       `json:"display_name,omitempty"`
	ContentType string       `json:"content_type,omitempty"`
	SizeBytes   int64        `json:"size_bytes"`
	SHA256      string       `json:"sha256"`
}

type munkiArtifactUpload struct {
	UploadURL string            `json:"upload_url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Artifact  ArtifactMutation  `json:"artifact"`
}

// RegisterAdminRoutes registers Munki artifact admin endpoints.
func RegisterAdminRoutes(api huma.API, store *Store, artifactStorage ArtifactStorage) {
	registerListMunkiArtifacts(api, store)
	registerCreateMunkiArtifact(api, store, artifactStorage)
	registerCreateMunkiArtifactUpload(api, artifactStorage)
	registerGetMunkiArtifact(api, store)
	registerGetMunkiArtifactContent(api, store, artifactStorage)
	registerDeleteMunkiArtifact(api, store)
}

func registerListMunkiArtifacts(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-artifacts",
		Method:      http.MethodGet,
		Path:        munkiArtifactPath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki artifacts",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *munkiArtifactListInput) (*munkiArtifactListOutput, error) {
		rows, count, err := store.List(ctx, input.ListQueryInput.Params())
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiArtifactLabel, err)
		}
		return &munkiArtifactListOutput{
			Body: apitypes.Page[Artifact]{Items: rows, Count: count},
		}, nil
	})
}

func registerCreateMunkiArtifact(api huma.API, store *Store, artifactStorage ArtifactStorage) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-artifact",
		Method:        http.MethodPost,
		Path:          munkiArtifactPath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki artifact",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusConflict,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *munkiArtifactCreateInput) (*munkiArtifactOutput, error) {
		mutation := input.Body
		if err := verifyMunkiArtifactObject(ctx, artifactStorage, mutation); err != nil {
			return nil, err
		}
		artifact, err := store.Create(ctx, mutation)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiArtifactLabel, err)
		}
		return &munkiArtifactOutput{Body: *artifact}, nil
	})
}

func registerCreateMunkiArtifactUpload(api huma.API, uploads ArtifactStorage) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-artifact-upload",
		Method:        http.MethodPost,
		Path:          munkiArtifactUploadPath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki artifact upload URL",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *munkiArtifactUploadInput) (*munkiArtifactUploadOutput, error) {
		if uploads == nil {
			return nil, munkiArtifactStorageUnavailable()
		}
		target, err := BuildUploadTarget(UploadTargetInput{
			Kind:        input.Body.Kind,
			Filename:    input.Body.Filename,
			DisplayName: input.Body.DisplayName,
			ContentType: input.Body.ContentType,
			SizeBytes:   input.Body.SizeBytes,
			SHA256:      input.Body.SHA256,
		})
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiArtifactLabel, err)
		}
		upload, err := uploads.PresignPut(ctx, target.StorageKey, target.ContentType, target.SHA256)
		if err != nil {
			return nil, munkiArtifactStorageError(err)
		}
		return &munkiArtifactUploadOutput{
			Body: munkiArtifactUpload{
				UploadURL: upload.URL,
				Headers:   upload.Headers,
				Artifact:  target,
			},
		}, nil
	})
}

func registerGetMunkiArtifact(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-artifact",
		Method:      http.MethodGet,
		Path:        munkiArtifactIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki artifact",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiArtifactGetInput) (*munkiArtifactOutput, error) {
		artifact, err := store.GetByID(ctx, input.ArtifactID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiArtifactLabel, err)
		}
		return &munkiArtifactOutput{Body: *artifact}, nil
	})
}

func registerGetMunkiArtifactContent(api huma.API, store *Store, artifactStorage ArtifactStorage) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-artifact-content",
		Method:      http.MethodGet,
		Path:        munkiArtifactContentPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki artifact content URL",
		Errors: []int{
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *munkiArtifactContentInput) (*munkiArtifactContentOutput, error) {
		if artifactStorage == nil {
			return nil, munkiArtifactStorageUnavailable()
		}
		artifact, err := store.GetByID(ctx, input.ArtifactID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(munkiArtifactLabel, err)
		}
		location, err := artifactStorage.PresignGet(ctx, *artifact)
		if err != nil {
			return nil, munkiArtifactStorageError(err)
		}
		return &munkiArtifactContentOutput{Status: http.StatusFound, Location: location}, nil
	})
}

func registerDeleteMunkiArtifact(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-munki-artifact",
		Method:      http.MethodDelete,
		Path:        munkiArtifactIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Delete a Munki artifact",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *munkiArtifactDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ArtifactID); err != nil {
			return nil, apitypes.ResourceMutationError(munkiArtifactLabel, err)
		}
		return &struct{}{}, nil
	})
}

func verifyMunkiArtifactObject(
	ctx context.Context,
	artifactStorage ArtifactStorage,
	mutation ArtifactMutation,
) error {
	if artifactStorage == nil {
		return munkiArtifactStorageUnavailable()
	}
	object, err := artifactStorage.Stat(ctx, mutation.StorageKey)
	if errors.Is(err, ErrObjectNotFound) {
		return apitypes.ResourceMutationError(
			munkiArtifactLabel,
			fmt.Errorf("%w: uploaded object does not exist", dbutil.ErrInvalidInput),
		)
	}
	if err != nil {
		return munkiArtifactStorageError(err)
	}
	if object.SizeBytes != mutation.SizeBytes {
		return apitypes.ResourceMutationError(
			munkiArtifactLabel,
			fmt.Errorf("%w: uploaded object size does not match artifact metadata", dbutil.ErrInvalidInput),
		)
	}
	if object.SHA256 != "" && object.SHA256 != mutation.SHA256 {
		return apitypes.ResourceMutationError(
			munkiArtifactLabel,
			fmt.Errorf("%w: uploaded object checksum does not match artifact metadata", dbutil.ErrInvalidInput),
		)
	}
	return nil
}

func munkiArtifactStorageUnavailable() error {
	return huma.Error503ServiceUnavailable("Munki artifact storage is not configured")
}

func munkiArtifactStorageError(err error) error {
	if errors.Is(err, ErrUnavailable) {
		return munkiArtifactStorageUnavailable()
	}
	return err
}
