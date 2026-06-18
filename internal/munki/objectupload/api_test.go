package objectupload

import (
	"context"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestCreateUsesPresignedUploadTarget(t *testing.T) {
	db, ctx := dbtest.Open(t)
	objects := storage.NewObjectStore(db, nil)
	presigner := &recordingPresigner{
		target: storage.UploadTarget{
			URL:       "https://woodstar.example/storage/munki/icons/1/icon.png?cap=test",
			Method:    "PUT",
			Transport: storage.UploadTransportWoodstar,
			Headers:   map[string]string{"Content-Type": "image/png"},
		},
	}

	out, err := Create(ctx, objects, presigner, "munki/icons", MunkiUploadRequest{
		Filename:    "icon.png",
		ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if presigner.key == "" {
		t.Fatal("PresignPut was not called")
	}
	if presigner.opts.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", presigner.opts.ContentType)
	}
	if out.Body.UploadURL != presigner.target.URL {
		t.Fatalf("upload URL = %q, want %q", out.Body.UploadURL, presigner.target.URL)
	}
	if out.Body.UploadTransport != MunkiUploadTransport(storage.UploadTransportWoodstar) {
		t.Fatalf("upload transport = %q, want woodstar", out.Body.UploadTransport)
	}
	if out.Body.Headers["Content-Type"] != "image/png" {
		t.Fatalf("headers = %v, want content type header", out.Body.Headers)
	}
}

type recordingPresigner struct {
	key    string
	opts   storage.PutOptions
	target storage.UploadTarget
}

func (p *recordingPresigner) PresignGet(
	context.Context,
	string,
	time.Duration,
	storage.GetOptions,
) (string, error) {
	return "", nil
}

func (p *recordingPresigner) PresignPut(
	_ context.Context,
	key string,
	_ time.Duration,
	opts storage.PutOptions,
) (storage.UploadTarget, error) {
	p.key = key
	p.opts = opts
	return p.target, nil
}
