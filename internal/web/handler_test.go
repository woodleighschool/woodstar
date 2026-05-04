package web

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestInjectRuntimeIncludesPublicURLAndCSRF(t *testing.T) {
	handler := NewHandler(HandlerOptions{
		FS:        testFS(),
		PublicURL: "https://example.com",
		Version:   "test",
		CSRFToken: func(*http.Request) string { return "csrf-token-value" },
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	handler.serveIndex(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `"apiBaseURL":"https://example.com"`) {
		t.Fatalf("runtime config missing apiBaseURL: %s", body)
	}
	if !strings.Contains(body, `"csrfToken":"csrf-token-value"`) {
		t.Fatalf("runtime config missing csrfToken: %s", body)
	}
	if !strings.Contains(body, `"version":"test"`) {
		t.Fatalf("runtime config missing version: %s", body)
	}
}

func TestServeAssetReturnsAsset(t *testing.T) {
	handler := NewHandler(HandlerOptions{
		FS:        testFS(),
		PublicURL: "https://example.com",
		Version:   "test",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/assets/app.js", nil)

	handler.serveAsset(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatalf("missing immutable cache header: %s", rec.Header().Get("Cache-Control"))
	}
}

func testFS() fs.FS {
	return fstest.MapFS{
		"index.html":    {Data: []byte("<html><head></head><body></body></html>")},
		"assets/app.js": {Data: []byte("console.log('ok')")},
		"favicon.ico":   {Data: []byte("ico")},
		"favicon.png":   {Data: []byte("png")},
	}
}
