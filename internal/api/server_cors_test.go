package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/config"
)

func TestCORSDisabledByDefault(t *testing.T) {
	t.Parallel()
	handler := corsMiddleware(config.Config{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://panel.example.com")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestCORSPreflightAllowsConfiguredOriginAndBlobHeaders(t *testing.T) {
	t.Parallel()
	handler := corsMiddleware(config.Config{
		CORSAllowedOrigins: []string{"https://panel.example.com"},
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/storage/munki/packages/1/Installer.pkg", nil)
	req.Header.Set("Origin", "https://panel.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	req.Header.Set("Access-Control-Request-Headers", "content-type,range")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://panel.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want configured origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "range") {
		t.Fatalf("Access-Control-Allow-Headers = %q, want range", got)
	}
}

func TestCompressionMiddlewareCompressesResponse(t *testing.T) {
	t.Parallel()
	compression, err := compressionMiddleware()
	if err != nil {
		t.Fatalf("compressionMiddleware returned error: %v", err)
	}
	handler := compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("api-bytes", 200)))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
}

func TestRequestTimeoutMiddlewareSetsConfiguredDeadline(t *testing.T) {
	t.Parallel()

	const timeout = time.Minute
	var gotTimeout time.Duration
	handler := requestTimeoutMiddleware(timeout)(
		http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			deadline, ok := req.Context().Deadline()
			if !ok {
				t.Fatal("request context has no deadline")
			}
			gotTimeout = time.Until(deadline)
		}),
	)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	if gotTimeout < timeout-time.Second || gotTimeout > timeout {
		t.Fatalf("context timeout = %s, want about %s", gotTimeout, timeout)
	}
}
