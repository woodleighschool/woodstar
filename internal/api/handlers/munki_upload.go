package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const munkiUploadLabel = "Munki upload"

type MunkiUploadRequest struct {
	Filename string `json:"filename"`
}

type MunkiObjectMutation struct {
	ObjectID int64 `json:"object_id" minimum:"1"`
}

type MunkiUploadTransport storage.UploadTransport

var munkiUploadTransportValues = []MunkiUploadTransport{
	MunkiUploadTransport(storage.UploadTransportWoodstar),
	MunkiUploadTransport(storage.UploadTransportS3),
}

func (MunkiUploadTransport) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(munkiUploadTransportValues...)
}

type MunkiUploadTarget struct {
	ObjectID        int64                `json:"object_id"`
	UploadURL       string               `json:"upload_url"`
	Method          string               `json:"method"`
	UploadTransport MunkiUploadTransport `json:"upload_transport"`
	Headers         map[string]string    `json:"headers,omitempty"`
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

func munkiUploadOutputFromTarget(obj *storage.Object, target storage.UploadTarget) *munkiUploadOutput {
	return &munkiUploadOutput{Body: MunkiUploadTarget{
		ObjectID:        obj.ID,
		UploadURL:       target.URL,
		Method:          target.Method,
		UploadTransport: MunkiUploadTransport(target.Transport),
		Headers:         target.Headers,
	}}
}

func munkiObjectView(o storage.Object) MunkiObjectView {
	return MunkiObjectView{
		ID:          o.ID,
		Filename:    o.Filename,
		ContentType: o.ContentType,
		SizeBytes:   o.SizeBytes,
		SHA256:      o.SHA256,
	}
}

func munkiObjectViewWithContentURL(
	ctx context.Context,
	objects *storage.ObjectStore,
	o storage.Object,
) (MunkiObjectView, error) {
	view := munkiObjectView(o)
	contentURL, err := objects.ContentURL(ctx, o)
	if err != nil {
		return MunkiObjectView{}, err
	}
	view.ContentURL = contentURL
	return view, nil
}

func finalizeMunkiUpload(
	ctx context.Context,
	uploads *munkiupload.Service,
	prefix string,
	objectID int64,
) (*storage.Object, error) {
	object, err := uploads.Finalize(ctx, objectID, prefix)
	if errors.Is(err, storage.ErrObjectNotFound) {
		return nil, errors.Join(
			fmt.Errorf("%w: uploaded object does not exist", dbutil.ErrInvalidInput),
			cleanupMunkiUpload(ctx, uploads, objectID),
		)
	}
	return object, err
}

func setMunkiObject(
	ctx context.Context,
	objects *storage.ObjectStore,
	uploads *munkiupload.Service,
	prefix string,
	objectID int64,
	set func(int64) error,
) (MunkiObjectView, error) {
	object, err := finalizeMunkiUpload(ctx, uploads, prefix, objectID)
	if err != nil {
		return MunkiObjectView{}, err
	}
	if err := set(object.ID); err != nil {
		return MunkiObjectView{}, errors.Join(err, cleanupMunkiUpload(ctx, uploads, object.ID))
	}
	return munkiObjectViewWithContentURL(ctx, objects, *object)
}

func cleanupMunkiUpload(ctx context.Context, uploads *munkiupload.Service, objectID int64) error {
	err := uploads.Delete(ctx, objectID)
	if errors.Is(err, dbutil.ErrConflict) || errors.Is(err, dbutil.ErrNotFound) {
		return nil
	}
	return err
}
