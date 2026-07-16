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
	"strings"

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

// MultipartUpload is the provider state Uppy needs to start uploading parts.
type MultipartUpload struct {
	UploadID string
	Key      string
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
		return nil, storage.UploadTarget{}, errors.Join(err, s.Delete(ctx, object.ID, prefix))
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
	if object.MultipartUploadID != nil {
		return nil, fmt.Errorf("%w: multipart upload must be completed before finalization", dbutil.ErrInvalidInput)
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
				s.Delete(ctx, object.ID, prefix),
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

// CreateMultipart creates or resumes the provider upload recorded for a pending object.
func (s *Service) CreateMultipart(
	ctx context.Context,
	objectID int64,
	prefix string,
) (MultipartUpload, error) {
	object, backend, err := s.multipartObject(ctx, objectID, prefix)
	if err != nil {
		return MultipartUpload{}, err
	}
	if object.MultipartUploadID != nil {
		return MultipartUpload{UploadID: *object.MultipartUploadID, Key: object.Key()}, nil
	}
	canonicalExists, err := s.canonicalObjectExists(ctx, object.Key())
	if err != nil {
		return MultipartUpload{}, err
	}
	if canonicalExists {
		return MultipartUpload{}, fmt.Errorf(
			"%w: multipart upload is already completed and ready to finalize",
			dbutil.ErrInvalidInput,
		)
	}
	uploadID, err := backend.CreateMultipartUpload(ctx, object.Key())
	if err != nil {
		return MultipartUpload{}, err
	}
	recordedID, created, err := s.objects.RecordMultipartUploadID(ctx, object.ID, uploadID)
	if err != nil {
		return MultipartUpload{}, errors.Join(err, backend.AbortMultipartUpload(ctx, object.Key(), uploadID))
	}
	if !created {
		abortErr := backend.AbortMultipartUpload(ctx, object.Key(), uploadID)
		if abortErr != nil && !errors.Is(abortErr, storage.ErrMultipartUploadNotFound) {
			return MultipartUpload{}, abortErr
		}
	}
	return MultipartUpload{UploadID: recordedID, Key: object.Key()}, nil
}

// PresignMultipartPart returns an S3 PUT target for a recorded multipart upload.
func (s *Service) PresignMultipartPart(
	ctx context.Context,
	objectID int64,
	prefix string,
	partNumber int32,
) (storage.UploadTarget, error) {
	if partNumber < 1 || partNumber > 10_000 {
		return storage.UploadTarget{}, fmt.Errorf("%w: part_number must be between 1 and 10000", dbutil.ErrInvalidInput)
	}
	object, backend, err := s.multipartObject(ctx, objectID, prefix)
	if err != nil {
		return storage.UploadTarget{}, err
	}
	if object.MultipartUploadID == nil {
		return storage.UploadTarget{}, fmt.Errorf("%w: multipart upload has not been created", dbutil.ErrInvalidInput)
	}
	return backend.PresignMultipartPart(ctx, object.Key(), *object.MultipartUploadID, partNumber, 0)
}

// CompleteMultipart assembles uploaded parts at the canonical object key.
func (s *Service) CompleteMultipart(
	ctx context.Context,
	objectID int64,
	prefix string,
	parts []storage.CompletedPart,
) error {
	if err := validateCompletedParts(parts); err != nil {
		return err
	}
	object, backend, err := s.multipartObject(ctx, objectID, prefix)
	if err != nil {
		return err
	}
	if object.MultipartUploadID == nil {
		exists, existsErr := s.canonicalObjectExists(ctx, object.Key())
		if existsErr != nil {
			return existsErr
		}
		if exists {
			return nil
		}
		return fmt.Errorf("%w: multipart upload has not been created", dbutil.ErrInvalidInput)
	}
	uploadID := *object.MultipartUploadID
	err = backend.CompleteMultipartUpload(ctx, object.Key(), uploadID, parts)
	if errors.Is(err, storage.ErrMultipartUploadNotFound) {
		exists, existsErr := s.canonicalObjectExists(ctx, object.Key())
		if existsErr != nil {
			return existsErr
		}
		if !exists {
			return err
		}
	} else if err != nil {
		return err
	}
	return s.objects.ClearMultipartUploadID(ctx, object.ID, uploadID)
}

// Delete removes a pending upload or a canonical object under prefix.
func (s *Service) Delete(ctx context.Context, objectID int64, prefix string) error {
	object, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return err
	}
	if object.Prefix != prefix {
		return fmt.Errorf("%w: object has the wrong storage prefix", dbutil.ErrInvalidInput)
	}
	if object.MultipartUploadID != nil {
		backend, err := s.multipartBackend()
		if err != nil {
			return err
		}
		if err := backend.AbortMultipartUpload(ctx, object.Key(), *object.MultipartUploadID); err != nil &&
			!errors.Is(err, storage.ErrMultipartUploadNotFound) {
			return err
		}
	}
	if !object.Available() {
		key := uploadKey(object.ID)
		if err := s.backend.Delete(ctx, key); err != nil {
			return fmt.Errorf("delete %q: %w", key, err)
		}
	}
	return s.objects.Delete(ctx, object.ID)
}

func (s *Service) multipartObject(
	ctx context.Context,
	objectID int64,
	prefix string,
) (*storage.Object, storage.MultipartBackend, error) {
	backend, err := s.multipartBackend()
	if err != nil {
		return nil, nil, err
	}
	object, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return nil, nil, err
	}
	if object.Prefix != prefix {
		return nil, nil, fmt.Errorf("%w: object has the wrong storage prefix", dbutil.ErrInvalidInput)
	}
	if object.Available() {
		return nil, nil, fmt.Errorf("%w: storage object is already finalized", dbutil.ErrInvalidInput)
	}
	return object, backend, nil
}

func (s *Service) multipartBackend() (storage.MultipartBackend, error) {
	backend, ok := s.backend.(storage.MultipartBackend)
	if !ok {
		return nil, fmt.Errorf("%w: multipart uploads require S3 storage", dbutil.ErrInvalidInput)
	}
	return backend, nil
}

func (s *Service) canonicalObjectExists(ctx context.Context, key string) (bool, error) {
	reader, _, err := s.backend.Open(ctx, key)
	if errors.Is(err, storage.ErrObjectNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := reader.Close(); err != nil {
		return false, err
	}
	return true, nil
}

func validateCompletedParts(parts []storage.CompletedPart) error {
	if len(parts) == 0 {
		return fmt.Errorf("%w: multipart completion requires at least one part", dbutil.ErrInvalidInput)
	}
	var previous int32
	for _, part := range parts {
		if part.PartNumber < 1 || part.PartNumber > 10_000 {
			return fmt.Errorf("%w: part_number must be between 1 and 10000", dbutil.ErrInvalidInput)
		}
		if part.PartNumber <= previous {
			return fmt.Errorf("%w: multipart parts must be strictly ascending", dbutil.ErrInvalidInput)
		}
		if strings.TrimSpace(part.ETag) == "" {
			return fmt.Errorf("%w: multipart part etag must not be blank", dbutil.ErrInvalidInput)
		}
		previous = part.PartNumber
	}
	return nil
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
