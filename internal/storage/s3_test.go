package storage

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

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
