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

func TestInjectRuntimeUsesBaseURL(t *testing.T) {
	handler := NewHandler(HandlerOptions{
		FS:      testFS(),
		BaseURL: "https://woodstar.example.edu/woodstar",
		Version: "test",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/woodstar/", nil)

	handler.serveIndex(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `apiBaseURL:"https://woodstar.example.edu/woodstar"`) {
		t.Fatalf("runtime config did not include scoped apiBaseURL: %s", body)
	}
	if !strings.Contains(body, `baseURL:"https://woodstar.example.edu/woodstar"`) {
		t.Fatalf("runtime config did not include baseURL: %s", body)
	}
}

func TestServeAssetStripsBaseURL(t *testing.T) {
	handler := NewHandler(HandlerOptions{
		FS:      testFS(),
		BaseURL: "https://woodstar.example.edu/woodstar",
		Version: "test",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/woodstar/assets/app.js", nil)

	handler.serveAsset(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
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
