package storage

import (
	"context"
	"errors"

	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
)

var (
	// ErrUnavailable reports that Munki artifact storage is disabled or unusable.
	ErrUnavailable = errors.New("munki artifact storage unavailable")

	// ErrObjectNotFound reports that the configured storage backend has no object for a key.
	ErrObjectNotFound = errors.New("munki artifact object not found")
)

// Config selects the artifact storage backend for Munki artifacts.
type Config struct {
	Enabled bool
	S3      S3Config
}

// ArtifactStorage signs temporary URLs and reads uploaded object metadata.
type ArtifactStorage interface {
	PresignGet(context.Context, artifacts.Artifact) (string, error)
	PresignPut(context.Context, string, string, string) (artifacts.ArtifactUploadURL, error)
	Stat(context.Context, string) (artifacts.ArtifactObject, error)
}

// NewArtifactStorage returns the configured Munki artifact storage surface.
func NewArtifactStorage(ctx context.Context, cfg Config) (ArtifactStorage, error) {
	if !cfg.Enabled {
		return disabledStorage{}, nil
	}
	storage, err := NewS3Presigner(ctx, cfg.S3)
	if err != nil {
		return disabledStorage{}, err
	}
	return storage, nil
}

type disabledStorage struct{}

func (disabledStorage) PresignGet(context.Context, artifacts.Artifact) (string, error) {
	return "", ErrUnavailable
}

func (disabledStorage) PresignPut(
	context.Context,
	string,
	string,
	string,
) (artifacts.ArtifactUploadURL, error) {
	return artifacts.ArtifactUploadURL{}, ErrUnavailable
}

func (disabledStorage) Stat(context.Context, string) (artifacts.ArtifactObject, error) {
	return artifacts.ArtifactObject{}, ErrUnavailable
}

var (
	_ ArtifactStorage = disabledStorage{}
	_ ArtifactStorage = (*S3Presigner)(nil)
)
