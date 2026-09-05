package httpapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/woodleighschool/goodies/bloby"
	blobyhuma "github.com/woodleighschool/goodies/bloby/huma"
	"github.com/woodleighschool/woodstar/internal/fault"
)

const munkiUploadLabel = "Munki upload"

type MunkiDirectUploadRequest struct {
	Filename string `json:"filename"`
}

type MunkiPackageInstallerUploadRequest struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes" minimum:"0"`
}

type MunkiObjectMutation struct {
	ObjectID int64 `json:"object_id" minimum:"1"`
}

type MunkiMultipartCompleteRequest struct {
	Parts []bloby.CompletedPart `json:"parts" minItems:"1"`
}

type MunkiUploadTarget struct {
	ObjectID int64                  `json:"object_id"`
	Upload   blobyhuma.UploadAction `json:"upload"`
}

type MunkiObjectView struct {
	ID          int64   `json:"id"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	SizeBytes   *int64  `json:"size_bytes,omitempty"`
	SHA256      *string `json:"sha256,omitempty"`
	ContentURL  string  `json:"content_url"`
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
		SizeBytes:   o.SizeBytes,
		SHA256:      o.SHA256,
		ContentURL:  contentURL,
	}
}

func finalizeMunkiUpload(
	ctx context.Context,
	objects *bloby.Service,
	prefix string,
	objectID int64,
) (*bloby.Object, error) {
	object, err := objects.Finalize(ctx, objectID, prefix)
	if errors.Is(err, bloby.ErrObjectNotFound) {
		return nil, errors.Join(
			fmt.Errorf("%w: uploaded object does not exist", fault.ErrInvalidInput),
			cleanupMunkiUpload(ctx, objects, objectID, prefix),
		)
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
