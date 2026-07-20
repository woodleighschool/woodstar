//go:build integration

package storageintegration

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/testutil/storagecontract"
)

const (
	garageImage                 = "dxflrs/garage:v2.3.0@sha256:866bd13ed2038ba7e7190e840482bc27234c4afaf77be8cfa439ae088c1e4690"
	garagePort                  = "3900/tcp"
	garageRegion                = "garage"
	garageBucket                = "woodstar-integration"
	garageAccessKey             = "GK0123456789abcdef0123456789abcdef"
	garageSecretKey             = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	storageContainerTimeout     = 2 * time.Minute
	storageCleanupTimeout       = 20 * time.Second
	storageContainerStopTimeout = time.Second
	storageRequestTimeout       = 30 * time.Second
)

func TestS3StorageBackend(t *testing.T) {
	endpoint := startGarage(t)
	backend, err := storage.New(t.Context(), storage.Config{
		Kind:        storage.KindS3,
		TransferTTL: time.Minute,
		S3: storage.S3Config{
			Bucket:         garageBucket,
			Region:         garageRegion,
			Endpoint:       endpoint,
			PublicEndpoint: endpoint,
			AccessKey:      garageAccessKey,
			SecretKey:      garageSecretKey,
			PathStyle:      true,
		},
	})
	if err != nil {
		t.Fatalf("create S3 storage: %v", err)
	}

	t.Run("backend", func(t *testing.T) {
		storagecontract.Run(t, backend)
	})
	t.Run("presigned", func(t *testing.T) {
		testS3PresignedTransfers(t, backend, endpoint)
	})
	t.Run("move content type", func(t *testing.T) {
		testS3MoveContentType(t, backend, endpoint)
	})
	t.Run("multipart", func(t *testing.T) {
		testS3MultipartTransfers(t, backend, endpoint)
	})
}

func testS3MoveContentType(t *testing.T, backend storage.Backend, endpoint string) {
	t.Helper()

	const movedContentType = "application/x-woodstar-moved"
	client := &http.Client{Timeout: storageRequestTimeout}
	sourceKey := "move/source object with spaces.bin"
	destinationKey := "move/destination object with spaces.bin"
	want := []byte("bytes moved inside S3")
	if err := backend.Put(t.Context(), sourceKey, bytes.NewReader(want), storage.PutOptions{
		ContentType: "application/x-woodstar-source",
	}); err != nil {
		t.Fatalf("put move source: %v", err)
	}
	if err := backend.Move(t.Context(), sourceKey, destinationKey, storage.PutOptions{
		ContentType: movedContentType,
	}); err != nil {
		t.Fatalf("move S3 object: %v", err)
	}
	assertStorageObjectMissing(t, backend, sourceKey)

	getURL, err := backend.PresignGet(t.Context(), destinationKey, time.Minute, storage.GetOptions{})
	if err != nil {
		t.Fatalf("presign moved object: %v", err)
	}
	assertPresignedEndpoint(t, getURL, endpoint)
	response := getS3URL(t, client, getURL)
	defer func() { _ = response.Body.Close() }()
	if got := response.Header.Get("Content-Type"); got != movedContentType {
		t.Fatalf("moved object content type = %q, want %q", got, movedContentType)
	}
	got, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read moved object: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("moved object bytes = %q, want %q", got, want)
	}
}

func testS3PresignedTransfers(t *testing.T, backend storage.Backend, endpoint string) {
	t.Helper()

	const (
		uploadContentType   = "application/x-woodstar-upload"
		downloadContentType = "application/x-woodstar-download"
	)
	client := &http.Client{Timeout: storageRequestTimeout}
	object := storage.Object{
		ID:          42,
		Prefix:      "presigned",
		Filename:    "direct object with spaces.bin",
		ContentType: downloadContentType,
	}
	key := object.Key()
	want := []byte("bytes uploaded through an actual presigned S3 request")
	target, err := backend.PresignPut(t.Context(), key, time.Minute)
	if err != nil {
		t.Fatalf("presign put: %v", err)
	}
	assertS3UploadTarget(t, target, endpoint)
	putS3Target(t, client, target, want, uploadContentType)

	getURL, err := backend.PresignGet(t.Context(), key, time.Minute, storage.GetOptions{})
	if err != nil {
		t.Fatalf("presign get: %v", err)
	}
	assertPresignedEndpoint(t, getURL, endpoint)
	response := getS3URL(t, client, getURL)
	if got := response.Header.Get("Content-Type"); got != uploadContentType {
		_ = response.Body.Close()
		t.Fatalf("stored content type = %q, want %q", got, uploadContentType)
	}
	got, err := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if err != nil {
		t.Fatalf("read presigned get: %v", err)
	}
	if closeErr != nil {
		t.Fatalf("close presigned get: %v", closeErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("presigned get bytes = %q, want %q", got, want)
	}

	deliveryResponse := httptest.NewRecorder()
	deliveryRequest := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/content", nil)
	if err := storage.NewDelivery(backend).Deliver(deliveryResponse, deliveryRequest, object, storage.DeliveryOptions{
		CacheControl: "private, max-age=86400",
	}); err != nil {
		t.Fatalf("deliver S3 object: %v", err)
	}
	if deliveryResponse.Code != http.StatusFound {
		t.Fatalf("S3 delivery status = %d, want 302", deliveryResponse.Code)
	}
	deliveryURL := deliveryResponse.Header().Get("Location")
	assertPresignedEndpoint(t, deliveryURL, endpoint)
	parsedDeliveryURL, err := url.Parse(deliveryURL)
	if err != nil {
		t.Fatalf("parse S3 delivery URL: %v", err)
	}
	if got := parsedDeliveryURL.Query().Get("response-content-type"); got != downloadContentType {
		t.Fatalf("S3 delivery content type = %q, want %q", got, downloadContentType)
	}
	if got := parsedDeliveryURL.Query().Get("response-cache-control"); got != "private, max-age=86400" {
		t.Fatalf("S3 delivery cache control = %q, want private max-age", got)
	}

	getURL, err = backend.PresignGet(t.Context(), key, time.Minute, storage.GetOptions{
		ContentType: downloadContentType,
	})
	if err != nil {
		t.Fatalf("presign get with content type: %v", err)
	}
	response = getS3URL(t, client, getURL)
	defer func() { _ = response.Body.Close() }()
	if got := response.Header.Get("Content-Type"); got != downloadContentType {
		t.Fatalf("presigned response content type = %q, want %q", got, downloadContentType)
	}
}

func testS3MultipartTransfers(t *testing.T, backend storage.Backend, endpoint string) {
	t.Helper()

	multipart, ok := backend.(storage.MultipartBackend)
	if !ok {
		t.Fatal("S3 backend does not implement storage.MultipartBackend")
	}
	client := &http.Client{Timeout: storageRequestTimeout}
	key := "multipart/assembled object with spaces.bin"
	uploadID, err := multipart.CreateMultipartUpload(t.Context(), key)
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	first := bytes.Repeat([]byte("a"), 5*1024*1024)
	second := []byte("final multipart bytes")
	parts := []storage.CompletedPart{
		presignAndUploadPart(t, client, multipart, endpoint, key, uploadID, 1, first),
		presignAndUploadPart(t, client, multipart, endpoint, key, uploadID, 2, second),
	}
	if err := multipart.CompleteMultipartUpload(t.Context(), key, uploadID, parts); err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}
	want := make([]byte, 0, len(first)+len(second))
	want = append(want, first...)
	want = append(want, second...)
	assertStorageObject(t, backend, key, want)

	abortKey := "multipart/aborted object with spaces.bin"
	abortID, err := multipart.CreateMultipartUpload(t.Context(), abortKey)
	if err != nil {
		t.Fatalf("create multipart upload to abort: %v", err)
	}
	presignAndUploadPart(t, client, multipart, endpoint, abortKey, abortID, 1, []byte("discarded bytes"))
	if err := multipart.AbortMultipartUpload(t.Context(), abortKey, abortID); err != nil {
		t.Fatalf("abort multipart upload: %v", err)
	}
	if err := multipart.AbortMultipartUpload(t.Context(), abortKey, abortID); !errors.Is(
		err,
		storage.ErrMultipartUploadNotFound,
	) {
		t.Fatalf("abort removed multipart upload error = %v, want storage.ErrMultipartUploadNotFound", err)
	}
	assertStorageObjectMissing(t, backend, abortKey)
}

func presignAndUploadPart(
	t *testing.T,
	client *http.Client,
	multipart storage.MultipartBackend,
	endpoint string,
	key string,
	uploadID string,
	partNumber int32,
	body []byte,
) storage.CompletedPart {
	t.Helper()

	target, err := multipart.PresignMultipartPart(t.Context(), key, uploadID, partNumber, time.Minute)
	if err != nil {
		t.Fatalf("presign multipart part %d: %v", partNumber, err)
	}
	assertS3UploadTarget(t, target, endpoint)
	etag := putS3Target(t, client, target, body, "application/octet-stream")
	if etag == "" {
		t.Fatalf("multipart part %d returned an empty ETag", partNumber)
	}
	return storage.CompletedPart{PartNumber: partNumber, ETag: etag}
}

func assertStorageObject(t *testing.T, backend storage.Backend, key string, want []byte) {
	t.Helper()

	reader, info, err := backend.Open(t.Context(), key)
	if err != nil {
		t.Fatalf("open %q: %v", key, err)
	}
	got, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		t.Fatalf("read %q: %v", key, readErr)
	}
	if closeErr != nil {
		t.Fatalf("close %q: %v", key, closeErr)
	}
	if info.Size != int64(len(want)) {
		t.Fatalf("%q size = %d, want %d", key, info.Size, len(want))
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%q bytes = %q, want %q", key, got, want)
	}
}

func assertStorageObjectMissing(t *testing.T, backend storage.Backend, key string) {
	t.Helper()

	reader, _, err := backend.Open(t.Context(), key)
	if reader != nil {
		_ = reader.Close()
	}
	if !errors.Is(err, storage.ErrObjectNotFound) {
		t.Fatalf("open missing %q error = %v, want storage.ErrObjectNotFound", key, err)
	}
}

func assertS3UploadTarget(t *testing.T, target storage.UploadTarget, endpoint string) {
	t.Helper()

	if target.Method != http.MethodPut {
		t.Fatalf("upload method = %q, want PUT", target.Method)
	}
	assertPresignedEndpoint(t, target.URL, endpoint)
}

func assertPresignedEndpoint(t *testing.T, rawURL string, endpoint string) {
	t.Helper()

	targetURL, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse presigned URL: %v", err)
	}
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		t.Fatalf("parse S3 endpoint: %v", err)
	}
	if targetURL.Scheme != endpointURL.Scheme || targetURL.Host != endpointURL.Host {
		t.Fatalf(
			"presigned endpoint = %s://%s, want %s://%s",
			targetURL.Scheme,
			targetURL.Host,
			endpointURL.Scheme,
			endpointURL.Host,
		)
	}
}

func putS3Target(
	t *testing.T,
	client *http.Client,
	target storage.UploadTarget,
	body []byte,
	contentType string,
) string {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), target.Method, target.URL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create presigned PUT: %v", err)
	}
	for name, value := range target.Headers {
		request.Header.Set(name, value)
	}
	request.Header.Set("Content-Type", contentType)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("execute presigned PUT: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4*1024))
		t.Fatalf(
			"presigned PUT status = %d, want 200: %s",
			response.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}
	return response.Header.Get("ETag")
}

func getS3URL(t *testing.T, client *http.Client, rawURL string) *http.Response {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("create presigned GET: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("execute presigned GET: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		defer func() { _ = response.Body.Close() }()
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4*1024))
		t.Fatalf(
			"presigned GET status = %d, want 200: %s",
			response.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}
	return response
}

func startGarage(t *testing.T) string {
	t.Helper()

	configPath := t.TempDir() + "/garage.toml"
	if err := os.WriteFile(configPath, []byte(garageConfig()), 0o600); err != nil {
		t.Fatalf("write Garage config: %v", err)
	}
	startCtx, startCancel := context.WithTimeout(t.Context(), storageContainerTimeout)
	container, err := testcontainers.Run(
		startCtx,
		garageImage,
		testcontainers.WithExposedPorts(garagePort),
		testcontainers.WithEnv(map[string]string{
			"GARAGE_DEFAULT_ACCESS_KEY": garageAccessKey,
			"GARAGE_DEFAULT_SECRET_KEY": garageSecretKey,
			"GARAGE_DEFAULT_BUCKET":     garageBucket,
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      configPath,
			ContainerFilePath: "/etc/garage.toml",
			FileMode:          0o600,
		}),
		testcontainers.WithCmd("/garage", "server", "--single-node", "--default-bucket"),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/").
				WithPort(garagePort).
				WithStatusCodeMatcher(func(status int) bool { return status >= 200 && status < 500 }).
				WithStartupTimeout(storageContainerTimeout),
		),
	)
	startCancel()
	if err != nil {
		t.Fatalf("start Garage container: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), storageCleanupTimeout)
		defer cleanupCancel()
		if err := container.Terminate(cleanupCtx, testcontainers.StopTimeout(storageContainerStopTimeout)); err != nil {
			t.Errorf("terminate Garage container: %v", err)
		}
	})

	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("resolve Garage host: %v", err)
	}
	port, err := container.MappedPort(t.Context(), garagePort)
	if err != nil {
		t.Fatalf("resolve Garage port: %v", err)
	}
	return "http://" + host + ":" + port.Port()
}

func garageConfig() string {
	return `metadata_dir = "/tmp/garage-meta"
data_dir = "/tmp/garage-data"
db_engine = "sqlite"

replication_factor = 1
rpc_bind_addr = "[::]:3901"
rpc_public_addr = "127.0.0.1:3901"
rpc_secret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

[s3_api]
s3_region = "garage"
api_bind_addr = "[::]:3900"
root_domain = ".s3.garage.localhost"
`
}
