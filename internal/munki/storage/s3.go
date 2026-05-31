// Package storage signs Munki artifact storage URLs.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/woodleighschool/woodstar/internal/munki"
)

// S3Config contains the Munki artifact S3 settings.
type S3Config struct {
	Bucket    string
	Region    string
	Endpoint  string
	AccessKey string
	SecretKey string
	PathStyle bool
	TTL       time.Duration
}

// S3Presigner creates presigned GET URLs for Munki artifacts.
type S3Presigner struct {
	bucket string
	ttl    time.Duration
	client *s3.PresignClient
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
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = cfg.PathStyle
		if cfg.Endpoint != "" {
			options.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	return &S3Presigner{
		bucket: cfg.Bucket,
		ttl:    cfg.TTL,
		client: s3.NewPresignClient(client),
	}, nil
}

// PresignGet returns a temporary GET URL for artifact's storage key.
func (p *S3Presigner) PresignGet(ctx context.Context, artifact munki.Artifact) (string, error) {
	output, err := p.client.PresignGetObject(ctx, &s3.GetObjectInput{
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
