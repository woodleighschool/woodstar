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

	obj, target, err := Create(ctx, objects, presigner, "munki/icons", "icon.png", "image/png")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if presigner.key == "" {
		t.Fatal("PresignPut was not called")
	}
	if presigner.opts.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", presigner.opts.ContentType)
	}
	if obj.ID == 0 {
		t.Fatal("object ID was not assigned")
	}
	if target.URL != presigner.target.URL {
		t.Fatalf("upload URL = %q, want %q", target.URL, presigner.target.URL)
	}
	if target.Transport != storage.UploadTransportWoodstar {
		t.Fatalf("upload transport = %q, want woodstar", target.Transport)
	}
	if target.Headers["Content-Type"] != "image/png" {
		t.Fatalf("headers = %v, want content type header", target.Headers)
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
