package handlers

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
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
	Method    string            `json:"method"     enum:"PUT"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type MunkiMultipartCompletedPart struct {
	PartNumber int32  `json:"part_number" minimum:"1" maximum:"10000"`
	ETag       string `json:"etag"                                    minLength:"1"`
}

type MunkiMultipartCompleteRequest struct {
	Parts []MunkiMultipartCompletedPart `json:"parts" minItems:"1"`
}

const (
	munkiUploadStrategyDirectPut = "direct-put"
	munkiUploadStrategyMultipart = "multipart"
)

type MunkiDirectUploadAction struct {
	Strategy string            `json:"strategy"          enum:"direct-put"`
	URL      string            `json:"url"`
	Method   string            `json:"method"            enum:"PUT"`
	Headers  map[string]string `json:"headers,omitempty"`
}

type MunkiMultipartUploadAction struct {
	Strategy string `json:"strategy" enum:"multipart"`
}

type MunkiUploadAction struct {
	Strategy string            `json:"strategy"`
	URL      string            `json:"url,omitempty"`
	Method   string            `json:"method,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

func (MunkiUploadAction) Schema(registry huma.Registry) *huma.Schema {
	return &huma.Schema{
		OneOf: []*huma.Schema{
			registry.Schema(reflect.TypeFor[MunkiDirectUploadAction](), true, "direct-put"),
			registry.Schema(reflect.TypeFor[MunkiMultipartUploadAction](), true, "multipart"),
		},
		Discriminator: &huma.Discriminator{
			PropertyName: "strategy",
			Mapping: map[string]string{
				munkiUploadStrategyDirectPut: "#/components/schemas/MunkiDirectUploadAction",
				munkiUploadStrategyMultipart: "#/components/schemas/MunkiMultipartUploadAction",
			},
		},
	}
}

type MunkiPackageInstallerUploadTarget struct {
	ObjectID int64             `json:"object_id"`
	Upload   MunkiUploadAction `json:"upload"`
}

type MunkiDirectUploadTarget struct {
	ObjectID int64                   `json:"object_id"`
	Upload   MunkiDirectUploadAction `json:"upload"`
}

type MunkiObjectView struct {
	ID          int64   `json:"id"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	SizeBytes   *int64  `json:"size_bytes,omitempty"`
	SHA256      *string `json:"sha256,omitempty"`
	ContentURL  string  `json:"content_url"`
}

type munkiPackageInstallerUploadOutput struct {
	Body MunkiPackageInstallerUploadTarget
}

type munkiDirectUploadOutput struct {
	Body MunkiDirectUploadTarget
}

type munkiObjectOutput struct {
	Body MunkiObjectView
}

func newMunkiDirectUploadOutput(
	obj *storage.Object,
	target storage.UploadTarget,
) *munkiDirectUploadOutput {
	return &munkiDirectUploadOutput{Body: MunkiDirectUploadTarget{
		ObjectID: obj.ID,
		Upload: MunkiDirectUploadAction{
			Strategy: munkiUploadStrategyDirectPut,
			URL:      target.URL,
			Method:   target.Method,
			Headers:  target.Headers,
		},
	}}
}

func newMunkiPackageInstallerDirectUploadOutput(
	obj *storage.Object,
	target storage.UploadTarget,
) *munkiPackageInstallerUploadOutput {
	return &munkiPackageInstallerUploadOutput{Body: MunkiPackageInstallerUploadTarget{
		ObjectID: obj.ID,
		Upload: MunkiUploadAction{
			Strategy: munkiUploadStrategyDirectPut,
			URL:      target.URL,
			Method:   target.Method,
			Headers:  target.Headers,
		},
	}}
}

func newMunkiPackageInstallerMultipartUploadOutput(
	obj *storage.Object,
) *munkiPackageInstallerUploadOutput {
	return &munkiPackageInstallerUploadOutput{Body: MunkiPackageInstallerUploadTarget{
		ObjectID: obj.ID,
		Upload: MunkiUploadAction{
			Strategy: munkiUploadStrategyMultipart,
		},
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
