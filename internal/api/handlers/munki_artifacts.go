package handlers

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki"
)

const (
	munkiArtifactPath        = "/api/munki/artifacts"
	munkiArtifactContentPath = "/api/munki/artifacts/{id}/content"
	munkiArtifactUploadPath  = "/api/munki/artifact-uploads"
	munkiArtifactLabel       = "Munki artifact"
)

type munkiArtifactStorage interface {
	PresignGet(context.Context, munki.Artifact) (string, error)
	PresignPut(context.Context, string, string, string) (munki.ArtifactUploadURL, error)
	Stat(context.Context, string) (munki.ArtifactObject, error)
}

type munkiArtifactCreateInput struct {
	Body munkiArtifactMutation
}

type munkiArtifactUploadInput struct {
	Body munkiArtifactUploadMutation
}

type munkiArtifactContentInput struct {
	ID int64 `path:"id"`
}

type munkiArtifactOutput struct {
	Body munkiArtifact
}

type munkiArtifactUploadOutput struct {
	Body munkiArtifactUpload
}

type munkiArtifactContentOutput struct {
	Status   int    `json:"-" default:"302"`
	Location string `                       header:"Location"`
}

type munkiArtifact struct {
	ID          int64              `json:"id"`
	Kind        munki.ArtifactKind `json:"kind"`
	DisplayName string             `json:"display_name"`
	Location    string             `json:"location"`
	ContentType string             `json:"content_type"`
	SizeBytes   int64              `json:"size_bytes"`
	SHA256      string             `json:"sha256"`
	StorageKey  string             `json:"storage_key"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type munkiArtifactMutation struct {
	Kind        munki.ArtifactKind `json:"kind"`
	DisplayName string             `json:"display_name,omitempty"`
	Location    string             `json:"location"`
	ContentType string             `json:"content_type,omitempty"`
	SizeBytes   int64              `json:"size_bytes"`
	SHA256      string             `json:"sha256"`
	StorageKey  string             `json:"storage_key"`
}

type munkiArtifactUploadMutation struct {
	Kind        munki.ArtifactKind `json:"kind"`
	Filename    string             `json:"filename"`
	DisplayName string             `json:"display_name,omitempty"`
	ContentType string             `json:"content_type,omitempty"`
	SizeBytes   int64              `json:"size_bytes"`
	SHA256      string             `json:"sha256"`
}

type munkiArtifactUpload struct {
	UploadURL string                `json:"upload_url"`
	Headers   map[string]string     `json:"headers,omitempty"`
	Artifact  munkiArtifactMutation `json:"artifact"`
}

func registerMunkiArtifacts(api huma.API, store *munki.Store, artifactStorage munkiArtifactStorage) {
	registerCreateMunkiArtifact(api, store, artifactStorage)
	registerCreateMunkiArtifactUpload(api, artifactStorage)
	registerGetMunkiArtifactContent(api, store, artifactStorage)
}

func registerCreateMunkiArtifact(api huma.API, store *munki.Store, artifactStorage munkiArtifactStorage) {
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
		mutation := input.Body.domain()
		if err := verifyMunkiArtifactObject(ctx, artifactStorage, mutation); err != nil {
			return nil, err
		}
		artifact, err := store.CreateArtifact(ctx, mutation)
		if err != nil {
			return nil, resourceMutationError(munkiArtifactLabel, err)
		}
		return &munkiArtifactOutput{Body: munkiArtifactFromDomain(*artifact)}, nil
	})
}

func registerCreateMunkiArtifactUpload(api huma.API, uploads munkiArtifactStorage) {
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
		target, err := input.Body.target()
		if err != nil {
			return nil, resourceMutationError(munkiArtifactLabel, err)
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

func registerGetMunkiArtifactContent(api huma.API, store *munki.Store, artifactStorage munkiArtifactStorage) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-artifact-content",
		Method:      http.MethodGet,
		Path:        munkiArtifactContentPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki artifact content URL",
		Errors: []int{
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, input *munkiArtifactContentInput) (*munkiArtifactContentOutput, error) {
		if artifactStorage == nil {
			return nil, munkiArtifactStorageUnavailable()
		}
		artifact, err := store.GetArtifact(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiArtifactLabel, err)
		}
		location, err := artifactStorage.PresignGet(ctx, *artifact)
		if err != nil {
			return nil, munkiArtifactStorageError(err)
		}
		return &munkiArtifactContentOutput{Status: http.StatusFound, Location: location}, nil
	})
}

func munkiArtifactFromDomain(artifact munki.Artifact) munkiArtifact {
	return munkiArtifact{
		ID:          artifact.ID,
		Kind:        artifact.Kind,
		DisplayName: artifact.DisplayName,
		Location:    artifact.Location,
		ContentType: artifact.ContentType,
		SizeBytes:   artifact.SizeBytes,
		SHA256:      artifact.SHA256,
		StorageKey:  artifact.StorageKey,
		CreatedAt:   artifact.CreatedAt,
		UpdatedAt:   artifact.UpdatedAt,
	}
}

func verifyMunkiArtifactObject(
	ctx context.Context,
	artifactStorage munkiArtifactStorage,
	mutation munki.ArtifactMutation,
) error {
	if artifactStorage == nil {
		return munkiArtifactStorageUnavailable()
	}
	object, err := artifactStorage.Stat(ctx, mutation.StorageKey)
	if errors.Is(err, munki.ErrNotFound) {
		return resourceMutationError(
			munkiArtifactLabel,
			fmt.Errorf("%w: uploaded object does not exist", dbutil.ErrInvalidInput),
		)
	}
	if err != nil {
		return munkiArtifactStorageError(err)
	}
	if object.SizeBytes != mutation.SizeBytes {
		return resourceMutationError(
			munkiArtifactLabel,
			fmt.Errorf("%w: uploaded object size does not match artifact metadata", dbutil.ErrInvalidInput),
		)
	}
	if object.SHA256 != "" && object.SHA256 != mutation.SHA256 {
		return resourceMutationError(
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
	if errors.Is(err, munki.ErrStorageUnavailable) {
		return munkiArtifactStorageUnavailable()
	}
	return err
}

func (body munkiArtifactMutation) domain() munki.ArtifactMutation {
	return munki.ArtifactMutation{
		Kind:        body.Kind,
		DisplayName: body.DisplayName,
		Location:    body.Location,
		ContentType: body.ContentType,
		SizeBytes:   body.SizeBytes,
		SHA256:      body.SHA256,
		StorageKey:  body.StorageKey,
	}
}

func (body munkiArtifactUploadMutation) target() (munkiArtifactMutation, error) {
	filename := cleanArtifactFilename(body.Filename)
	if filename == "" {
		return munkiArtifactMutation{}, fmt.Errorf("%w: filename is required", dbutil.ErrInvalidInput)
	}
	contentType := strings.TrimSpace(body.ContentType)
	if contentType == "" {
		contentType = artifactContentType(filename)
	}
	target := munkiArtifactMutation{
		Kind:        body.Kind,
		DisplayName: body.DisplayName,
		Location:    artifactUploadLocation(body.SHA256, filename),
		ContentType: contentType,
		SizeBytes:   body.SizeBytes,
		SHA256:      strings.TrimSpace(body.SHA256),
	}
	target.StorageKey = artifactStorageKey(target.Kind, target.Location)
	if target.DisplayName == "" {
		target.DisplayName = filename
	}
	if err := target.domain().Validate(); err != nil {
		return munkiArtifactMutation{}, err
	}
	return target, nil
}

func cleanArtifactFilename(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, `\`, "/"))
	filename = path.Base(filename)
	if filename == "." || filename == "/" || filename == "" {
		return ""
	}
	return filename
}

func artifactUploadLocation(sha256 string, filename string) string {
	sha256 = strings.TrimSpace(sha256)
	if len(sha256) >= 12 {
		return sha256[:12] + "/" + filename
	}
	return filename
}

func artifactStorageKey(kind munki.ArtifactKind, location string) string {
	switch kind {
	case munki.ArtifactKindPackage:
		return "pkgs/" + location
	case munki.ArtifactKindIcon:
		return "icons/" + location
	default:
		return string(kind) + "/" + location
	}
}

func artifactContentType(filename string) string {
	if contentType := mime.TypeByExtension(path.Ext(filename)); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}
