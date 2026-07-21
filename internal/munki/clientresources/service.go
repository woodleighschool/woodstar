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

	"github.com/gabriel-vasile/mimetype"

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
	PublishBuilder(ctx context.Context, builder storedBuilder) (*ClientResources, error)
	PublishArchive(ctx context.Context, archiveObjectID int64) (*ClientResources, error)
	Delete(ctx context.Context) error
}

// Service publishes generated or uploaded client resources archives.
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

// SaveBuilder validates builder input, compiles its archive, and deploys it.
func (s *Service) SaveBuilder(ctx context.Context, builder Builder) (*ClientResources, error) {
	builder.normalize()
	if err := builder.validate(); err != nil {
		return nil, err
	}

	banner, wasPending, err := s.prepareBanner(ctx, builder.BannerObjectID)
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
	archiveBody, err := Compile(builder, extension, bannerBody)
	if err != nil {
		return nil, errors.Join(err, cleanupBanner())
	}
	archive, err := s.storeArchive(ctx, archiveBody)
	if err != nil {
		return nil, errors.Join(err, cleanupBanner())
	}

	resource, err := s.resources.PublishBuilder(ctx, storedBuilder{
		Builder:         builder,
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

// PublishArchive finalizes and deploys an uploaded archive.
func (s *Service) PublishArchive(ctx context.Context, objectID int64) (*ClientResources, error) {
	archive, wasPending, err := s.prepareArchive(ctx, objectID)
	if err != nil {
		return nil, err
	}
	resource, err := s.resources.PublishArchive(ctx, archive.ID)
	if err != nil && wasPending {
		err = errors.Join(err, cleanupUploads(ctx, s.ingestor, ArchiveObjectPrefix, archive.ID))
	}
	return resource, err
}

func (s *Service) Delete(ctx context.Context) error {
	return s.resources.Delete(ctx)
}

func (s *Service) prepareBanner(ctx context.Context, objectID int64) (*storage.Object, bool, error) {
	banner, wasPending, err := s.finalizeObject(ctx, objectID, BannerObjectPrefix, "banner")
	if err != nil {
		return nil, wasPending, err
	}
	if err := validateBanner(banner.ContentType, banner.SizeBytesValue()); err != nil {
		if wasPending {
			err = errors.Join(err, cleanupUploads(ctx, s.ingestor, BannerObjectPrefix, banner.ID))
		}
		return nil, wasPending, err
	}
	return banner, wasPending, nil
}

func (s *Service) prepareArchive(ctx context.Context, objectID int64) (*storage.Object, bool, error) {
	return s.finalizeObject(ctx, objectID, ArchiveObjectPrefix, "archive")
}

func (s *Service) finalizeObject(
	ctx context.Context,
	objectID int64,
	prefix string,
	label string,
) (*storage.Object, bool, error) {
	object, err := s.objects.GetByID(ctx, objectID)
	if err != nil {
		return nil, false, err
	}
	if object.Prefix != prefix {
		return nil, false, fmt.Errorf(
			"%w: object_id must reference a client resources %s",
			dbutil.ErrInvalidInput,
			label,
		)
	}
	wasPending := !object.Available()
	if wasPending {
		object, err = s.ingestor.Finalize(ctx, objectID, prefix)
		if errors.Is(err, storage.ErrObjectNotFound) {
			return nil, true, errors.Join(
				fmt.Errorf("%w: uploaded %s does not exist", dbutil.ErrInvalidInput, label),
				cleanupUploads(ctx, s.ingestor, prefix, objectID),
			)
		}
		if err != nil {
			return nil, true, errors.Join(
				err,
				cleanupUploads(ctx, s.ingestor, prefix, objectID),
			)
		}
	}
	return object, wasPending, nil
}

func (s *Service) readBanner(ctx context.Context, banner storage.Object) ([]byte, error) {
	reader, _, err := s.backend.Open(ctx, banner.Key())
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
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
	detected := mimetype.Lookup(contentType)
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
