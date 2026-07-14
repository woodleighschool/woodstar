package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestObjectKey(t *testing.T) {
	t.Parallel()
	obj := Object{ID: 42, Prefix: "munki/packages", Filename: "Firefox-120.0.dmg"}
	if got, want := obj.Key(), "munki/packages/42/Firefox-120.0.dmg"; got != want {
		t.Fatalf("Key() = %q, want %q", got, want)
	}
}

func TestNormalizeUploadFilename(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Firefox-120.0.dmg":   "Firefox-120.0.dmg",
		"/tmp/Firefox.dmg":    "Firefox.dmg",
		`C:\Users\me\App.pkg`: "App.pkg",
		"  Spaced.icns  ":     "Spaced.icns",
		"Acmé Café.png":       "Acmé Café.png",
		"sub/dir/file.pkg":    "file.pkg",
		"../../etc/passwd":    "passwd",
	}
	for in, want := range cases {
		got := normalizeUploadFilename(in)
		if got != want {
			t.Errorf("normalizeUploadFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateUploadFilenameRejects(t *testing.T) {
	t.Parallel()
	for _, in := range []string{
		"",
		"   ",
		".",
		"..",
		"/",
		"a/b/..",
		"with\x00null.pkg",
	} {
		name := normalizeUploadFilename(in)
		if err := validateUploadFilename(name); !errors.Is(err, dbutil.ErrInvalidInput) {
			t.Errorf("validateUploadFilename(%q) error = %v, want ErrInvalidInput", name, err)
		}
	}
}

func TestListByPrefixReturnsAvailableObjectsNewestFirst(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewObjectStore(db, nil)

	first, err := store.CreatePending(ctx, "munki/icons", "first.png", "image/png")
	if err != nil {
		t.Fatalf("create first object: %v", err)
	}
	if _, err := store.Confirm(ctx, first.ID, 1, "image/png", strings.Repeat("a", 64)); err != nil {
		t.Fatalf("confirm first object: %v", err)
	}
	second, err := store.CreatePending(ctx, "munki/icons", "second.png", "image/png")
	if err != nil {
		t.Fatalf("create second object: %v", err)
	}
	if _, err := store.Confirm(ctx, second.ID, 1, "image/png", strings.Repeat("b", 64)); err != nil {
		t.Fatalf("confirm second object: %v", err)
	}
	if _, err := store.CreatePending(ctx, "munki/icons", "pending.png", "image/png"); err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	other, err := store.CreatePending(ctx, "munki/packages", "other.pkg", "application/octet-stream")
	if err != nil {
		t.Fatalf("create other-prefix object: %v", err)
	}
	if _, err := store.Confirm(ctx, other.ID, 1, "application/octet-stream", strings.Repeat("c", 64)); err != nil {
		t.Fatalf("confirm other-prefix object: %v", err)
	}

	objects, count, err := store.ListByPrefix(ctx, "munki/icons", dbutil.ListParams{})
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if len(objects) != 2 || objects[0].ID != second.ID || objects[1].ID != first.ID {
		t.Fatalf("object IDs = %v, want [%d %d]", objectIDs(objects), second.ID, first.ID)
	}
}

func objectIDs(objects []Object) []int64 {
	ids := make([]int64, len(objects))
	for i, object := range objects {
		ids[i] = object.ID
	}
	return ids
}

func TestDeleteUnreferencedPreventsNewReferencesBeforeDeletingBytes(t *testing.T) {
	database, ctx := dbtest.Open(t)
	backend := newBlockingDeleteStore()
	store := NewObjectStore(database, backend)
	object, err := store.CreatePending(ctx, "munki/icons", "icon.png", "image/png")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}

	cleanupResult := make(chan error, 1)
	go func() {
		cleanupResult <- store.DeleteUnreferenced(ctx, object.ID)
	}()

	<-backend.deleteStarted
	_, referenceErr := database.Pool().Exec(ctx, `
INSERT INTO munki_software (name, display_name, icon_object_id)
VALUES ('Race App', 'Race App', $1)`, object.ID)
	backend.releaseDelete()

	cleanupErr := <-cleanupResult
	if referenceErr == nil {
		t.Fatal("new software reference succeeded after object deletion began")
	}
	if cleanupErr != nil {
		t.Fatalf("delete unreferenced object: %v", cleanupErr)
	}
	if _, err := store.GetByID(ctx, object.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get deleted object error = %v, want %v", err, dbutil.ErrNotFound)
	}
}

type blockingDeleteStore struct {
	deleteStarted chan struct{}
	deleteRelease chan struct{}
	releaseOnce   sync.Once
}

func newBlockingDeleteStore() *blockingDeleteStore {
	return &blockingDeleteStore{
		deleteStarted: make(chan struct{}),
		deleteRelease: make(chan struct{}),
	}
}

func (s *blockingDeleteStore) Open(context.Context, string) (ObjectReader, ObjectInfo, error) {
	return nil, ObjectInfo{}, errors.New("unexpected open")
}

func (s *blockingDeleteStore) Put(context.Context, string, io.Reader, PutOptions) error {
	return errors.New("unexpected put")
}

func (s *blockingDeleteStore) Delete(context.Context, string) error {
	close(s.deleteStarted)
	<-s.deleteRelease
	return nil
}

func (s *blockingDeleteStore) Stat(context.Context, string) (ObjectInfo, error) {
	return ObjectInfo{}, errors.New("unexpected stat")
}

func (s *blockingDeleteStore) releaseDelete() {
	s.releaseOnce.Do(func() { close(s.deleteRelease) })
}
