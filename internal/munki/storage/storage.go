package storage

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/munki"
)

// Config selects the artifact storage backend for Munki artifacts.
type Config struct {
	Enabled bool
	S3      S3Config
}

// ArtifactStorage signs temporary URLs and reads uploaded object metadata.
type ArtifactStorage interface {
	PresignGet(context.Context, munki.Artifact) (string, error)
	PresignPut(context.Context, string, string, string) (munki.ArtifactUploadURL, error)
	Stat(context.Context, string) (munki.ArtifactObject, error)
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

func (disabledStorage) PresignGet(context.Context, munki.Artifact) (string, error) {
	return "", munki.ErrStorageUnavailable
}

func (disabledStorage) PresignPut(
	context.Context,
	string,
	string,
	string,
) (munki.ArtifactUploadURL, error) {
	return munki.ArtifactUploadURL{}, munki.ErrStorageUnavailable
}

func (disabledStorage) Stat(context.Context, string) (munki.ArtifactObject, error) {
	return munki.ArtifactObject{}, munki.ErrStorageUnavailable
}

var (
	_ ArtifactStorage = disabledStorage{}
	_ ArtifactStorage = (*S3Presigner)(nil)
)
