package objectupload

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const Label = "munki upload"

type MunkiUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
}

type MunkiUploadKind string

const (
	MunkiUploadKindProxy     MunkiUploadKind = "proxy"
	MunkiUploadKindPresigned MunkiUploadKind = "presigned"
)

var uploadKindValues = []MunkiUploadKind{
	MunkiUploadKindProxy,
	MunkiUploadKindPresigned,
}

func (MunkiUploadKind) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(uploadKindValues...)
}

type MunkiUploadTarget struct {
	ObjectID   int64             `json:"object_id"`
	UploadKind MunkiUploadKind   `json:"upload_kind"`
	UploadURL  string            `json:"upload_url"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type MunkiObjectView struct {
	ID          int64   `json:"id"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	SizeBytes   *int64  `json:"size_bytes,omitempty"`
	SHA256      *string `json:"sha256,omitempty"`
	ContentURL  string  `json:"content_url,omitempty"`
}

type UploadOutput struct {
	Body MunkiUploadTarget
}

type ObjectOutput struct {
	Body MunkiObjectView
}

func Create(
	ctx context.Context,
	objects *storage.ObjectStore,
	store storage.Store,
	prefix string,
	req MunkiUploadRequest,
) (*UploadOutput, error) {
	obj, err := objects.CreatePending(ctx, prefix, req.Filename, req.ContentType)
	if err != nil {
		return nil, err
	}

	presigner, ok := store.(storage.Presigner)
	if !ok {
		return &UploadOutput{Body: MunkiUploadTarget{
			ObjectID:   obj.ID,
			UploadKind: MunkiUploadKindProxy,
			UploadURL:  fmt.Sprintf("/api/objects/%d/content", obj.ID),
			Method:     http.MethodPut,
		}}, nil
	}
	target, err := presigner.PresignPut(ctx, obj.Key(), 0, storage.PutOptions{ContentType: req.ContentType})
	if err != nil {
		return nil, err
	}
	return &UploadOutput{Body: MunkiUploadTarget{
		ObjectID:   obj.ID,
		UploadKind: MunkiUploadKindPresigned,
		UploadURL:  target.URL,
		Method:     target.Method,
		Headers:    target.Headers,
	}}, nil
}

func Confirm(
	ctx context.Context,
	objects *storage.ObjectStore,
	prefix string,
	objectID int64,
	attach func(objectID int64) error,
) (*ObjectOutput, error) {
	obj, err := objects.GetByID(ctx, objectID)
	if err != nil {
		return nil, err
	}
	if obj.Prefix != prefix {
		return nil, fmt.Errorf("%w: object has the wrong Munki prefix", dbutil.ErrInvalidInput)
	}
	confirmed, err := objects.ConfirmUploaded(ctx, objectID)
	if errors.Is(err, storage.ErrObjectNotFound) {
		return nil, fmt.Errorf("%w: uploaded object does not exist", dbutil.ErrInvalidInput)
	}
	if err != nil {
		return nil, err
	}
	if err := attach(confirmed.ID); err != nil {
		_ = objects.DeleteUnreferenced(ctx, confirmed.ID)
		return nil, err
	}
	return &ObjectOutput{Body: ViewObject(*confirmed)}, nil
}

func ViewObject(o storage.Object) MunkiObjectView {
	return MunkiObjectView{
		ID:          o.ID,
		Filename:    o.Filename,
		ContentType: o.ContentType,
		SizeBytes:   o.SizeBytes,
		SHA256:      o.SHA256,
	}
}
