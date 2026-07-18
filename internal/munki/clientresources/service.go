package clientresources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const (
	archiveContentType = "application/zip"
	maxRasterPixels    = 100_000_000
)

type registry interface {
	GetByID(ctx context.Context, objectID int64) (*storage.Object, error)
}

type objectIngestor interface {
	Finalize(ctx context.Context, objectID int64, prefix string) (*storage.Object, error)
	Write(ctx context.Context, prefix, filename, contentType string, body []byte) (*storage.Object, error)
	Delete(ctx context.Context, objectID int64, prefix string) error
}

type resourceStore interface {
	Get(ctx context.Context) (*ClientResources, error)
	Upsert(ctx context.Context, mutation storedMutation) (*ClientResources, error)
	Delete(ctx context.Context) error
}

// Service validates builder input, compiles the archive, and publishes the singleton.
type Service struct {
	resources resourceStore
	objects   registry
	ingestor  objectIngestor
	backend   storage.Store
}

func NewService(
	resources resourceStore,
	objects registry,
	ingestor objectIngestor,
	backend storage.Store,
) *Service {
	return &Service{resources: resources, objects: objects, ingestor: ingestor, backend: backend}
}

func (s *Service) Get(ctx context.Context) (*ClientResources, error) {
	return s.resources.Get(ctx)
}

func (s *Service) Save(ctx context.Context, mutation Mutation) (*ClientResources, error) {
	mutation.normalize()
	if err := mutation.validate(); err != nil {
		return nil, err
	}

	banner, wasPending, err := s.prepareBanner(ctx, mutation.BannerObjectID)
	if err != nil {
		return nil, err
	}
	cleanupBanner := func() error {
		if wasPending {
			return cleanupUploads(ctx, s.ingestor, BannerObjectPrefix, banner.ID)
		}
		return nil
	}

	bannerBody, err := s.readBanner(ctx, *banner)
	if err != nil {
		return nil, errors.Join(err, cleanupBanner())
	}
	extension, _ := bannerExtension(banner.ContentType)
	archiveBody, err := Compile(mutation, extension, bannerBody)
	if err != nil {
		return nil, errors.Join(err, cleanupBanner())
	}
	archive, err := s.storeArchive(ctx, archiveBody)
	if err != nil {
		return nil, errors.Join(err, cleanupBanner())
	}

	resource, err := s.resources.Upsert(ctx, storedMutation{
		Mutation:        mutation,
		ArchiveObjectID: archive.ID,
	})
	if err != nil {
		return nil, errors.Join(
			err,
			cleanupUploads(ctx, s.ingestor, ArchiveObjectPrefix, archive.ID),
			cleanupBanner(),
		)
	}
	return resource, nil
}

func (s *Service) Delete(ctx context.Context) error {
	return s.resources.Delete(ctx)
}

func (s *Service) prepareBanner(ctx context.Context, objectID int64) (*storage.Object, bool, error) {
	banner, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return nil, false, err
	}
	if banner.Prefix != BannerObjectPrefix {
		return nil, false, fmt.Errorf(
			"%w: banner_object_id must reference a client resources banner",
			dbutil.ErrInvalidInput,
		)
	}
	wasPending := !banner.Available()
	if wasPending {
		banner, err = s.ingestor.Finalize(ctx, objectID, BannerObjectPrefix)
		if errors.Is(err, storage.ErrObjectNotFound) {
			return nil, true, errors.Join(
				fmt.Errorf("%w: uploaded banner does not exist", dbutil.ErrInvalidInput),
				cleanupUploads(ctx, s.ingestor, BannerObjectPrefix, objectID),
			)
		}
		if err != nil {
			return nil, true, errors.Join(
				err,
				cleanupUploads(ctx, s.ingestor, BannerObjectPrefix, objectID),
			)
		}
	}
	if err := validateBanner(banner.ContentType, banner.SizeBytesValue()); err != nil {
		if wasPending {
			err = errors.Join(err, cleanupUploads(ctx, s.ingestor, BannerObjectPrefix, banner.ID))
		}
		return nil, wasPending, err
	}
	return banner, wasPending, nil
}

func (s *Service) readBanner(ctx context.Context, banner storage.Object) ([]byte, error) {
	reader, _, err := s.backend.Open(ctx, banner.Key())
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	body, err := io.ReadAll(io.LimitReader(reader, MaxBannerSizeBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read banner: %w", err)
	}
	if int64(len(body)) != banner.SizeBytesValue() || len(body) > MaxBannerSizeBytes {
		return nil, fmt.Errorf("%w: stored banner size does not match its registry record", dbutil.ErrInvalidInput)
	}
	if err := validateBannerBody(banner.ContentType, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (s *Service) storeArchive(ctx context.Context, body []byte) (*storage.Object, error) {
	return s.ingestor.Write(ctx, ArchiveObjectPrefix, archiveFilename, archiveContentType, body)
}

func validateBannerBody(contentType string, body []byte) error {
	detected := lookupContentType(contentType)
	if detected == nil {
		return fmt.Errorf("%w: unsupported banner content type", dbutil.ErrInvalidInput)
	}
	switch {
	case detected.Is("image/jpeg"), detected.Is("image/png"):
		var config image.Config
		var err error
		if detected.Is("image/jpeg") {
			config, err = jpeg.DecodeConfig(bytes.NewReader(body))
		} else {
			config, err = png.DecodeConfig(bytes.NewReader(body))
		}
		if err != nil {
			return fmt.Errorf("%w: decode banner: %w", dbutil.ErrInvalidInput, err)
		}
		if config.Width <= 0 || config.Height <= 0 ||
			int64(config.Width) > maxRasterPixels/int64(config.Height) {
			return fmt.Errorf("%w: banner dimensions are invalid or too large", dbutil.ErrInvalidInput)
		}
		return nil
	default:
		return fmt.Errorf("%w: unsupported banner content type", dbutil.ErrInvalidInput)
	}
}
