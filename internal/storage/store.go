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

// ErrMultipartUploadNotFound reports that a provider no longer has an upload ID.
var ErrMultipartUploadNotFound = errors.New("storage multipart upload not found")

// Store reads and writes blobs by key. Backends: file, s3.
type Store interface {
	Open(ctx context.Context, key string) (ObjectReader, ObjectInfo, error)
	Put(ctx context.Context, key string, r io.Reader, opts PutOptions) error
	Delete(ctx context.Context, key string) error
}

// ObjectReader is a backend object stream that supports HTTP range reads.
type ObjectReader interface {
	io.Reader
	io.Seeker
	io.Closer
}

// Backend is a configured storage backend. All runtime backends can read/write
// bytes and mint direct transfer capabilities.
type Backend interface {
	Store
	Presigner
	Move(ctx context.Context, sourceKey, destinationKey string, opts PutOptions) error
	PresignPut(ctx context.Context, key string, ttl time.Duration) (UploadTarget, error)
	deliveryMode() deliveryMode
}

// MultipartBackend is the optional S3 multipart transfer capability.
type MultipartBackend interface {
	CreateMultipartUpload(ctx context.Context, key string) (string, error)
	PresignMultipartPart(
		ctx context.Context,
		key, uploadID string,
		partNumber int32,
		ttl time.Duration,
	) (UploadTarget, error)
	CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []CompletedPart) error
	AbortMultipartUpload(ctx context.Context, key, uploadID string) error
}

// Presigner mints direct read URLs.
type Presigner interface {
	PresignGet(ctx context.Context, key string, ttl time.Duration, opts GetOptions) (string, error)
}

// ObjectInfo is backend metadata for stored bytes.
type ObjectInfo struct {
	Size int64
}

// PutOptions carries representation metadata to preserve with stored bytes.
type PutOptions struct {
	ContentType string
}

// GetOptions carries optional hints for a presigned read.
type GetOptions struct {
	ContentType  string
	CacheControl string
}

// UploadTransport tells clients which upload implementation should send bytes.
type UploadTransport string

const (
	UploadTransportWoodstar UploadTransport = "woodstar"
	UploadTransportS3       UploadTransport = "s3"
)

// UploadTarget tells a client where and how to upload an object's bytes.
type UploadTarget struct {
	URL       string
	Method    string
	Transport UploadTransport
	Headers   map[string]string
}

// CompletedPart identifies one uploaded S3 multipart part.
type CompletedPart struct {
	PartNumber int32
	ETag       string
}
