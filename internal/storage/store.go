// Package storage is woodstar's blob store: a small backend-agnostic interface
// over local files or S3, plus a database registry of the objects it holds.
package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrObjectNotFound reports that a backend has no object for a key.
var ErrObjectNotFound = errors.New("storage object not found")

// Store reads and writes blobs by key. Backends: file, s3.
type Store interface {
	Open(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
	Put(ctx context.Context, key string, r io.Reader, opts PutOptions) error
	Delete(ctx context.Context, key string) error
	Stat(ctx context.Context, key string) (ObjectInfo, error)
}

// Presigner hands a client a URL to transfer bytes directly. The S3 backend
// implements it; the file backend does not, so callers type-assert for it and
// fall back to proxying through woodstar's own routes.
type Presigner interface {
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
	PresignPut(ctx context.Context, key string, ttl time.Duration, opts PutOptions) (UploadTarget, error)
}

// ObjectInfo is backend metadata for a stored object. ContentType is empty for
// backends that do not record it (the file backend); callers fall back to the
// content type declared when the object was created.
type ObjectInfo struct {
	Size        int64
	ContentType string
}

// PutOptions carries optional hints for a write or a presigned upload.
type PutOptions struct {
	ContentType string
}

// UploadTarget tells a client where and how to upload an object's bytes.
type UploadTarget struct {
	URL     string
	Method  string
	Headers map[string]string
}
