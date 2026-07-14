package storage

import (
	"context"
	"fmt"
	"time"
)

// Kind selects a storage backend.
type Kind string

// Supported backends.
const (
	KindFile Kind = "file"
	KindS3   Kind = "s3"
)

// Config selects and configures a backend.
type Config struct {
	Kind          Kind
	FileRoot      string
	BaseURL       string
	CapabilityKey []byte
	PresignTTL    time.Duration
	S3            S3Config
}

// S3Config holds the settings for the S3 backend.
type S3Config struct {
	Bucket         string
	Region         string
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	PathStyle      bool
	PresignTTL     time.Duration
}

// New builds the configured backend.
func New(ctx context.Context, cfg Config) (Backend, error) {
	switch cfg.Kind {
	case KindFile:
		return newFileStore(cfg.FileRoot, cfg.BaseURL, cfg.CapabilityKey, cfg.PresignTTL)
	case KindS3:
		return newS3Store(ctx, cfg.S3)
	default:
		return nil, fmt.Errorf("unknown storage kind %q", cfg.Kind)
	}
}
