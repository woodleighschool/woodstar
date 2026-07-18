package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
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

type MunkiMultipartUpload struct {
	UploadID string `json:"upload_id"`
	Key      string `json:"key"`
}

type MunkiMultipartPartTarget struct {
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type MunkiMultipartCompletedPart struct {
	PartNumber int32  `json:"part_number" minimum:"1" maximum:"10000"`
	ETag       string `json:"etag"                                    minLength:"1"`
}

type MunkiMultipartCompleteRequest struct {
	Parts []MunkiMultipartCompletedPart `json:"parts" minItems:"1"`
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

func munkiObjectView(o storage.Object, contentURL string) MunkiObjectView {
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
	ingestor *storage.Ingestor,
	prefix string,
	objectID int64,
) (*storage.Object, error) {
	object, err := ingestor.Finalize(ctx, objectID, prefix)
	if errors.Is(err, storage.ErrObjectNotFound) {
		return nil, errors.Join(
			fmt.Errorf("%w: uploaded object does not exist", dbutil.ErrInvalidInput),
			cleanupMunkiUpload(ctx, ingestor, objectID, prefix),
		)
	}
	return object, err
}

func setMunkiObject(
	ctx context.Context,
	ingestor *storage.Ingestor,
	prefix string,
	objectID int64,
	set func(int64) error,
) (*storage.Object, error) {
	object, err := finalizeMunkiUpload(ctx, ingestor, prefix, objectID)
	if err != nil {
		return nil, err
	}
	if err := set(object.ID); err != nil {
		return nil, errors.Join(err, cleanupMunkiUpload(ctx, ingestor, object.ID, prefix))
	}
	return object, nil
}

func cleanupMunkiUpload(
	ctx context.Context,
	ingestor *storage.Ingestor,
	objectID int64,
	prefix string,
) error {
	err := ingestor.Delete(ctx, objectID, prefix)
	if errors.Is(err, dbutil.ErrConflict) || errors.Is(err, dbutil.ErrNotFound) {
		return nil
	}
	return err
}
