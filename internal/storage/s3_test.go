package storage

import (
	"net/url"
	"testing"
	"time"
)

func TestS3StoreTransferOriginMatchesPresignedPart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  S3Config
	}{
		{
			name: "public endpoint",
			cfg: S3Config{
				Bucket:         "woodstar",
				Region:         "ap-southeast-2",
				Endpoint:       "https://garage.internal.example",
				PublicEndpoint: "https://uploads.example",
				AccessKey:      "test-access-key",
				SecretKey:      "test-secret-key",
				PathStyle:      true,
				PresignTTL:     time.Minute,
			},
		},
		{
			name: "AWS endpoint",
			cfg: S3Config{
				Bucket:     "woodstar",
				Region:     "ap-southeast-2",
				AccessKey:  "test-access-key",
				SecretKey:  "test-secret-key",
				PresignTTL: time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store, err := newS3Store(t.Context(), tt.cfg)
			if err != nil {
				t.Fatalf("newS3Store: %v", err)
			}
			target, err := store.PresignMultipartPart(
				t.Context(),
				"munki/packages/42/Installer.pkg",
				"upload-id",
				1,
				time.Minute,
			)
			if err != nil {
				t.Fatalf("PresignMultipartPart: %v", err)
			}
			parsed, err := url.Parse(target.URL)
			if err != nil {
				t.Fatalf("parse target URL: %v", err)
			}
			if got, want := store.TransferOrigin(), parsed.Scheme+"://"+parsed.Host; got != want {
				t.Fatalf("transfer origin = %q, want presigned target origin %q", got, want)
			}
		})
	}
}
