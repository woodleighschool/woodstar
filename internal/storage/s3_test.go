package storage

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestS3MultipartCreateSignCompleteAndAbort(t *testing.T) {
	t.Parallel()
	type requestRecord struct {
		method string
		query  string
		body   string
	}
	var mu sync.Mutex
	var requests []requestRecord
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		requests = append(requests, requestRecord{method: r.Method, query: r.URL.RawQuery, body: string(body)})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/xml")
		switch {
		case r.Method == http.MethodPost && r.URL.Query().Has("uploads"):
			_, _ = io.WriteString(
				w,
				`<CreateMultipartUploadResult><Bucket>test</Bucket><Key>munki/packages/42/Installer.pkg</Key><UploadId>upload-42</UploadId></CreateMultipartUploadResult>`,
			)
		case r.Method == http.MethodPost && r.URL.Query().Get("uploadId") == "upload-42":
			_, _ = io.WriteString(
				w,
				`<CompleteMultipartUploadResult><Bucket>test</Bucket><Key>munki/packages/42/Installer.pkg</Key><ETag>whole</ETag></CompleteMultipartUploadResult>`,
			)
		case r.Method == http.MethodDelete && r.URL.Query().Get("uploadId") == "upload-42":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	client := newS3Client(aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("access", "secret", ""),
		HTTPClient:  server.Client(),
	}, server.URL, true)
	store := &s3Store{bucket: "test", client: client, presigner: s3.NewPresignClient(client)}
	key := "munki/packages/42/Installer.pkg"
	uploadID, err := store.CreateMultipartUpload(t.Context(), key)
	if err != nil {
		t.Fatalf("CreateMultipartUpload: %v", err)
	}
	if uploadID != "upload-42" {
		t.Fatalf("upload ID = %q, want upload-42", uploadID)
	}

	target, err := store.PresignMultipartPart(t.Context(), key, uploadID, 7, time.Minute)
	if err != nil {
		t.Fatalf("PresignMultipartPart: %v", err)
	}
	parsed, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("parse part URL: %v", err)
	}
	if target.Method != http.MethodPut || parsed.Query().Get("partNumber") != "7" ||
		parsed.Query().Get("uploadId") != uploadID {
		t.Fatalf("part target = %+v, want signed PUT for part 7", target)
	}

	parts := []CompletedPart{
		{PartNumber: 1, ETag: `"etag-1"`},
		{PartNumber: 7, ETag: `"etag-7"`},
	}
	if err := store.CompleteMultipartUpload(t.Context(), key, uploadID, parts); err != nil {
		t.Fatalf("CompleteMultipartUpload: %v", err)
	}
	if err := store.AbortMultipartUpload(t.Context(), key, uploadID); err != nil {
		t.Fatalf("AbortMultipartUpload: %v", err)
	}

	mu.Lock()
	got := append([]requestRecord(nil), requests...)
	mu.Unlock()
	if len(got) != 3 {
		t.Fatalf("requests = %#v, want create, complete, abort", got)
	}
	completion := got[1].body
	first := strings.Index(completion, "<PartNumber>1</PartNumber>")
	second := strings.Index(completion, "<PartNumber>7</PartNumber>")
	if first < 0 || second <= first {
		t.Fatalf("completion body = %q, want parts in ascending order", completion)
	}
}

func TestServeObjectReadsS3Ranges(t *testing.T) {
	t.Parallel()
	const body = "0123456789"
	var mu sync.Mutex
	var requestedRanges []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		mu.Lock()
		requestedRanges = append(requestedRanges, rangeHeader)
		mu.Unlock()

		offset := 0
		if rangeHeader != "" {
			value := strings.TrimSuffix(strings.TrimPrefix(rangeHeader, "bytes="), "-")
			parsed, err := strconv.Atoi(value)
			if err != nil {
				http.Error(w, "invalid range", http.StatusBadRequest)
				return
			}
			offset = parsed
			w.Header().Set("Content-Range", "bytes "+value+"-9/10")
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)-offset))
		if rangeHeader != "" {
			w.WriteHeader(http.StatusPartialContent)
		}
		_, _ = w.Write([]byte(body[offset:]))
	}))
	t.Cleanup(server.Close)

	client := newS3Client(aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("access", "secret", ""),
		HTTPClient:  server.Client(),
	}, server.URL, true)
	store := &s3Store{bucket: "test", client: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/icon", nil)
	request.Header.Set("Range", "bytes=2-5")

	if err := ServeObject(recorder, request, store, "munki/icons/7/icon.png", ServeOptions{}); err != nil {
		t.Fatalf("serve object: %v", err)
	}
	if recorder.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusPartialContent)
	}
	if recorder.Body.String() != "2345" {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), "2345")
	}

	mu.Lock()
	ranges := append([]string(nil), requestedRanges...)
	mu.Unlock()
	if len(ranges) < 2 || ranges[len(ranges)-1] != "bytes=2-" {
		t.Fatalf("S3 ranges = %q, want final request from byte 2", ranges)
	}
}

func TestS3PresignGetOverridesBackendContentType(t *testing.T) {
	t.Parallel()
	client := newS3Client(aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("access", "secret", ""),
	}, "https://storage.example", true)
	store := &s3Store{
		bucket:    "test",
		presigner: s3.NewPresignClient(client),
	}

	rawURL, err := store.PresignGet(
		t.Context(),
		"munki/icons/7/icon.png",
		time.Minute,
		GetOptions{ContentType: "image/png"},
	)
	if err != nil {
		t.Fatalf("presign get: %v", err)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if got := parsed.Query().Get("response-content-type"); got != "image/png" {
		t.Fatalf("response content type = %q, want image/png", got)
	}
}

func TestS3StoreMovePreservesContentTypeAndDeletesSource(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var requests []struct {
		method            string
		path              string
		copySource        string
		contentType       string
		metadataDirective string
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, struct {
			method            string
			path              string
			copySource        string
			contentType       string
			metadataDirective string
		}{
			r.Method,
			r.URL.Path,
			r.Header.Get("X-Amz-Copy-Source"),
			r.Header.Get("Content-Type"),
			r.Header.Get("X-Amz-Metadata-Directive"),
		})
		mu.Unlock()
		if r.Method == http.MethodPut {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<CopyObjectResult><ETag>"etag"</ETag></CopyObjectResult>`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client := newS3Client(aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("access", "secret", ""),
		HTTPClient:  server.Client(),
	}, server.URL, true)
	store := &s3Store{bucket: "test", client: client}
	sourceKey := ".uploads/42"
	destinationKey := "munki/packages/42/Installer.pkg"
	if err := store.Move(
		t.Context(),
		sourceKey,
		destinationKey,
		PutOptions{ContentType: "application/octet-stream"},
	); err != nil {
		t.Fatalf("Move: %v", err)
	}

	mu.Lock()
	got := append([]struct {
		method            string
		path              string
		copySource        string
		contentType       string
		metadataDirective string
	}(nil), requests...)
	mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("requests = %#v, want copy and delete", got)
	}
	decodedSource, err := url.PathUnescape(got[0].copySource)
	if err != nil {
		t.Fatalf("decode copy source: %v", err)
	}
	if got[0].method != http.MethodPut || got[0].path != "/test/"+destinationKey || decodedSource != "test/"+sourceKey {
		t.Fatalf("copy request = %#v, decoded source %q", got[0], decodedSource)
	}
	if got[0].contentType != "application/octet-stream" || got[0].metadataDirective != "REPLACE" {
		t.Fatalf("copy metadata = %#v, want content type replacement", got[0])
	}
	if got[1].method != http.MethodDelete || got[1].path != "/test/"+sourceKey {
		t.Fatalf("delete request = %#v", got[1])
	}
}
