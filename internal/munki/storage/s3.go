// Package storage signs Munki artifact storage URLs.
package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"github.com/woodleighschool/woodstar/internal/munki"
)

// S3Config contains the Munki artifact S3 settings.
type S3Config struct {
	Bucket         string
	Region         string
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	PathStyle      bool
	TTL            time.Duration
}

// S3Presigner creates presigned GET URLs for Munki artifacts.
type S3Presigner struct {
	bucket    string
	ttl       time.Duration
	client    *s3.Client
	presigner *s3.PresignClient
}

// NewS3Presigner returns an S3-backed Munki artifact presigner.
func NewS3Presigner(ctx context.Context, cfg S3Config) (*S3Presigner, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load munki s3 config: %w", err)
	}
	client := newS3Client(awsCfg, cfg.Endpoint, cfg.PathStyle)
	presignEndpoint := cfg.PublicEndpoint
	if presignEndpoint == "" {
		presignEndpoint = cfg.Endpoint
	}
	presignClient := newS3Client(awsCfg, presignEndpoint, cfg.PathStyle)
	return &S3Presigner{
		bucket:    cfg.Bucket,
		ttl:       cfg.TTL,
		client:    client,
		presigner: s3.NewPresignClient(presignClient),
	}, nil
}

func newS3Client(cfg aws.Config, endpoint string, pathStyle bool) *s3.Client {
	return s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.UsePathStyle = pathStyle
		if endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
	})
}

// PresignGet returns a temporary GET URL for artifact's storage key.
func (p *S3Presigner) PresignGet(ctx context.Context, artifact munki.Artifact) (string, error) {
	output, err := p.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(artifact.StorageKey),
	}, func(options *s3.PresignOptions) {
		options.Expires = p.ttl
	})
	if err != nil {
		return "", fmt.Errorf("presign munki artifact %d: %w", artifact.ID, err)
	}
	return output.URL, nil
}

// PresignPut returns a temporary PUT URL for storageKey.
func (p *S3Presigner) PresignPut(
	ctx context.Context,
	storageKey string,
	contentType string,
	sha256 string,
) (munki.ArtifactUploadURL, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(storageKey),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if sha256 != "" {
		input.Metadata = map[string]string{"woodstar-sha256": sha256}
	}
	output, err := p.presigner.PresignPutObject(ctx, input, func(options *s3.PresignOptions) {
		options.Expires = p.ttl
	})
	if err != nil {
		return munki.ArtifactUploadURL{}, fmt.Errorf("presign munki upload %q: %w", storageKey, err)
	}
	return munki.ArtifactUploadURL{
		URL:     output.URL,
		Headers: singleValueHeaders(output.SignedHeader),
	}, nil
}

// Stat returns object metadata for storageKey.
func (p *S3Presigner) Stat(ctx context.Context, storageKey string) (munki.ArtifactObject, error) {
	output, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(storageKey),
	})
	if s3NotFound(err) {
		return munki.ArtifactObject{}, munki.ErrNotFound
	}
	if err != nil {
		return munki.ArtifactObject{}, fmt.Errorf("stat munki artifact %q: %w", storageKey, err)
	}
	return munki.ArtifactObject{
		ContentType: aws.ToString(output.ContentType),
		SizeBytes:   aws.ToInt64(output.ContentLength),
		SHA256:      output.Metadata["woodstar-sha256"],
	}, nil
}

func s3NotFound(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "NotFound", "NoSuchKey", "404":
		return true
	default:
		return false
	}
}

func singleValueHeaders(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(headers))
	for key, values := range headers {
		if key == "Host" {
			continue
		}
		if len(values) > 0 {
			out[key] = values[0]
		}
	}
	return out
}
