package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/schema"
	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type MunkiUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
}

type MunkiUploadTransport storage.UploadTransport

var munkiUploadTransportValues = []MunkiUploadTransport{
	MunkiUploadTransport(storage.UploadTransportWoodstar),
	MunkiUploadTransport(storage.UploadTransportS3),
}

func (MunkiUploadTransport) Schema(_ huma.Registry) *huma.Schema {
	return schema.StringEnum(munkiUploadTransportValues...)
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
	ContentURL  string  `json:"content_url,omitempty"`
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
	presigner storage.Presigner,
	o storage.Object,
) (MunkiObjectView, error) {
	view := munkiObjectView(o)
	contentURL, err := munkiupload.ContentURL(ctx, presigner, o)
	if err != nil {
		return MunkiObjectView{}, err
	}
	view.ContentURL = contentURL
	return view, nil
}
