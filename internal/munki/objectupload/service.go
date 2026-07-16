// Package objectupload ingests objects into the Munki repository.
package objectupload

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/gabriel-vasile/mimetype"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const uploadPrefix = "munki/.uploads/"

// Service classifies incoming Munki content before committing it to storage.
type Service struct {
	objects *storage.ObjectStore
	backend storage.Backend
}

// NewService returns a Munki object ingestion service.
func NewService(objects *storage.ObjectStore, backend storage.Backend) *Service {
	return &Service{objects: objects, backend: backend}
}

// Begin reserves an object and returns a direct target for its temporary bytes.
func (s *Service) Begin(
	ctx context.Context,
	prefix string,
	filename string,
) (*storage.Object, storage.UploadTarget, error) {
	object, err := s.objects.CreatePending(ctx, prefix, filename)
	if err != nil {
		return nil, storage.UploadTarget{}, err
	}
	target, err := s.backend.PresignPut(ctx, uploadKey(object.ID), 0)
	if err != nil {
		return nil, storage.UploadTarget{}, errors.Join(err, s.Delete(ctx, object.ID))
	}
	return object, target, nil
}

// Write ingests server-generated content into the Munki repository.
func (s *Service) Write(
	ctx context.Context,
	prefix string,
	filename string,
	contentType string,
	body []byte,
) (*storage.Object, error) {
	object, err := s.objects.CreatePending(ctx, prefix, filename)
	if err != nil {
		return nil, err
	}
	if err := s.backend.Put(
		ctx,
		object.Key(),
		bytes.NewReader(body),
		storage.PutOptions{ContentType: contentType},
	); err != nil {
		return nil, errors.Join(err, s.objects.Delete(ctx, object.ID))
	}
	hash := sha256.Sum256(body)
	available, err := s.objects.MarkAvailable(
		ctx,
		object.ID,
		int64(len(body)),
		contentType,
		hex.EncodeToString(hash[:]),
	)
	if err != nil {
		return nil, errors.Join(err, s.objects.Delete(ctx, object.ID))
	}
	return available, nil
}

// Finalize classifies and hashes landed bytes, then commits the canonical object.
func (s *Service) Finalize(
	ctx context.Context,
	objectID int64,
	prefix string,
) (*storage.Object, error) {
	object, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return nil, err
	}
	if object.Prefix != prefix {
		return nil, fmt.Errorf("%w: object has the wrong storage prefix", dbutil.ErrInvalidInput)
	}
	if object.Available() {
		return object, nil
	}

	sourceKey := object.Key()
	metadata, err := s.inspect(ctx, sourceKey)
	if errors.Is(err, storage.ErrObjectNotFound) {
		sourceKey = uploadKey(object.ID)
		metadata, err = s.inspect(ctx, sourceKey)
	}
	if err != nil {
		return nil, err
	}
	if sourceKey != object.Key() {
		moveErr := s.backend.Move(
			ctx,
			sourceKey,
			object.Key(),
			storage.PutOptions{ContentType: metadata.contentType},
		)
		if moveErr != nil && !errors.Is(moveErr, storage.ErrObjectNotFound) {
			return nil, moveErr
		}
		canonicalMetadata, err := s.inspect(ctx, object.Key())
		if err != nil {
			return nil, err
		}
		if canonicalMetadata != metadata {
			return nil, errors.Join(
				fmt.Errorf("%w: uploaded object changed during finalization", dbutil.ErrInvalidInput),
				s.Delete(ctx, object.ID),
			)
		}
		metadata = canonicalMetadata
	}
	return s.objects.MarkAvailable(
		ctx,
		object.ID,
		metadata.sizeBytes,
		metadata.contentType,
		metadata.sha256,
	)
}

// Delete removes a pending upload or a canonical object.
func (s *Service) Delete(ctx context.Context, objectID int64) error {
	object, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return err
	}
	if !object.Available() {
		key := uploadKey(object.ID)
		if err := s.backend.Delete(ctx, key); err != nil {
			return fmt.Errorf("delete %q: %w", key, err)
		}
	}
	return s.objects.Delete(ctx, object.ID)
}

type objectMetadata struct {
	sizeBytes   int64
	contentType string
	sha256      string
}

func (s *Service) inspect(ctx context.Context, key string) (objectMetadata, error) {
	reader, info, err := s.backend.Open(ctx, key)
	if err != nil {
		return objectMetadata{}, err
	}
	hash := sha256.New()
	var size byteCount
	metadata := io.MultiWriter(hash, &size)
	detected, err := mimetype.DetectReader(io.TeeReader(reader, metadata))
	if err == nil {
		_, err = io.Copy(metadata, reader)
	}
	closeErr := reader.Close()
	if err != nil {
		return objectMetadata{}, fmt.Errorf("read %q: %w", key, err)
	}
	if closeErr != nil {
		return objectMetadata{}, fmt.Errorf("close %q: %w", key, closeErr)
	}
	if int64(size) != info.Size {
		return objectMetadata{}, fmt.Errorf("read %q: backend size changed during ingestion", key)
	}
	return objectMetadata{
		sizeBytes:   int64(size),
		contentType: detected.String(),
		sha256:      hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func uploadKey(objectID int64) string {
	return fmt.Sprintf("%s%d", uploadPrefix, objectID)
}

type byteCount int64

func (c *byteCount) Write(p []byte) (int, error) {
	*c += byteCount(len(p))
	return len(p), nil
}
