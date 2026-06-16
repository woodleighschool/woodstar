package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// s3Store stores blobs in an S3-compatible bucket. It implements Store and
// Presigner, so transfers go directly between client and bucket.
type s3Store struct {
	bucket    string
	ttl       time.Duration
	client    *s3.Client
	presigner *s3.PresignClient
}

func newS3Store(ctx context.Context, cfg S3Config) (*s3Store, error) {
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
		return nil, fmt.Errorf("load storage s3 config: %w", err)
	}
	client := newS3Client(awsCfg, cfg.Endpoint, cfg.PathStyle)
	presignEndpoint := cfg.PublicEndpoint
	if presignEndpoint == "" {
		presignEndpoint = cfg.Endpoint
	}
	presignClient := newS3Client(awsCfg, presignEndpoint, cfg.PathStyle)
	return &s3Store{
		bucket:    cfg.Bucket,
		ttl:       cfg.PresignTTL,
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

func (s *s3Store) Open(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if s3NotFound(err) {
		return nil, ObjectInfo{}, ErrObjectNotFound
	}
	if err != nil {
		return nil, ObjectInfo{}, fmt.Errorf("get %q: %w", key, err)
	}
	return output.Body, ObjectInfo{
		Size:        aws.ToInt64(output.ContentLength),
		ContentType: aws.ToString(output.ContentType),
	}, nil
}

// Put buffers the body to make it seekable for signing. The presigned upload
// path is the norm for large objects; server-side Put is for modest writes.
func (s *s3Store) Put(ctx context.Context, key string, r io.Reader, opts PutOptions) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read body for %q: %w", key, err)
	}
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	}
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("put %q: %w", key, err)
	}
	return nil
}

func (s *s3Store) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("delete %q: %w", key, err)
	}
	return nil
}

func (s *s3Store) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	output, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if s3NotFound(err) {
		return ObjectInfo{}, ErrObjectNotFound
	}
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("stat %q: %w", key, err)
	}
	return ObjectInfo{
		Size:        aws.ToInt64(output.ContentLength),
		ContentType: aws.ToString(output.ContentType),
	}, nil
}

func (s *s3Store) PresignGet(
	ctx context.Context,
	key string,
	ttl time.Duration,
	_ GetOptions,
) (string, error) {
	output, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s.expires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign get %q: %w", key, err)
	}
	return output.URL, nil
}

func (s *s3Store) PresignPut(
	ctx context.Context,
	key string,
	ttl time.Duration,
	opts PutOptions,
) (UploadTarget, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	output, err := s.presigner.PresignPutObject(ctx, input, s.expires(ttl))
	if err != nil {
		return UploadTarget{}, fmt.Errorf("presign put %q: %w", key, err)
	}
	return UploadTarget{
		URL:       output.URL,
		Method:    http.MethodPut,
		Transport: UploadTransportS3,
		Headers:   singleValueHeaders(output.SignedHeader),
	}, nil
}

func (s *s3Store) expires(ttl time.Duration) func(*s3.PresignOptions) {
	if ttl <= 0 {
		ttl = s.ttl
	}
	return func(options *s3.PresignOptions) {
		options.Expires = ttl
	}
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

var (
	_ Store     = (*fileStore)(nil)
	_ Presigner = (*fileStore)(nil)
	_ Store     = (*s3Store)(nil)
	_ Presigner = (*s3Store)(nil)
)
