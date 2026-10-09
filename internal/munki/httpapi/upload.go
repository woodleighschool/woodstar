package httpapi

import (
	"context"
	"errors"

	"github.com/woodleighschool/goodies/bloby"
	blobyhuma "github.com/woodleighschool/goodies/bloby/huma"
)

const munkiUploadLabel = "Munki upload"

// MunkiUploadRequest names an upload and declares the bytes it will carry.
// Storage accepts no other content for the object.
type MunkiUploadRequest struct {
	bloby.Content

	Filename string `json:"filename"`
}

type MunkiObjectMutation struct {
	ObjectID int64 `json:"object_id" minimum:"1"`
}

// MunkiMultipartPartRequest declares the checksum of one multipart part.
type MunkiMultipartPartRequest struct {
	CRC64NVME string `json:"crc64nvme" pattern:"^[0-9a-f]{16}$"`
}

type MunkiUploadTarget struct {
	ObjectID int64                  `json:"object_id"`
	Upload   blobyhuma.UploadAction `json:"upload"`
}

// MunkiObjectView describes a published storage object.
type MunkiObjectView struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
	ContentURL  string `json:"content_url"`
}

type munkiUploadOutput struct {
	Body MunkiUploadTarget
}

type munkiObjectOutput struct {
	Body MunkiObjectView
}

func newMunkiUploadOutput(
	obj *bloby.Object,
	action bloby.UploadAction,
) *munkiUploadOutput {
	return &munkiUploadOutput{Body: MunkiUploadTarget{
		ObjectID: obj.ID,
		Upload:   blobyhuma.UploadAction(action),
	}}
}

func munkiObjectView(o bloby.Object, contentURL string) MunkiObjectView {
	return MunkiObjectView{
		ID:          o.ID,
		Filename:    o.Filename,
		ContentType: o.ContentType,
		SizeBytes:   o.SizeBytesValue(),
		SHA256:      o.SHA256Value(),
		ContentURL:  contentURL,
	}
}

// finalizeMunkiUpload publishes an upload, releasing one whose bytes are
// missing or differ from its declaration.
func finalizeMunkiUpload(
	ctx context.Context,
	objects *bloby.Service,
	prefix string,
	objectID int64,
) (*bloby.Object, error) {
	object, err := objects.Finalize(ctx, objectID, prefix)
	if errors.Is(err, bloby.ErrObjectNotFound) || errors.Is(err, bloby.ErrContentMismatch) {
		return nil, errors.Join(err, cleanupMunkiUpload(ctx, objects, objectID, prefix))
	}
	return object, err
}

func setMunkiObject(
	ctx context.Context,
	objects *bloby.Service,
	prefix string,
	objectID int64,
	set func(int64) error,
) (*bloby.Object, error) {
	object, err := finalizeMunkiUpload(ctx, objects, prefix, objectID)
	if err != nil {
		return nil, err
	}
	if err := set(object.ID); err != nil {
		return nil, errors.Join(err, cleanupMunkiUpload(ctx, objects, object.ID, prefix))
	}
	return object, nil
}

func cleanupMunkiUpload(
	ctx context.Context,
	objects *bloby.Service,
	objectID int64,
	prefix string,
) error {
	err := objects.Delete(ctx, objectID, prefix)
	if errors.Is(err, bloby.ErrConflict) || errors.Is(err, bloby.ErrNotFound) {
		return nil
	}
	return err
}
