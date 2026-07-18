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
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
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
	req := httptest.NewRequest(http.MethodOptions, "/storage/munki/packages/1/Installer.pkg", nil)
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

func TestCompressionMiddlewareBypassesStorageIO(t *testing.T) {
	t.Parallel()
	compression, err := compressionMiddleware()
	if err != nil {
		t.Fatalf("compressionMiddleware returned error: %v", err)
	}
	handler := compression(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("storage-bytes", 200)))
		}),
	)

	for _, path := range []string{
		"/storage/.uploads/1",
		"/munki/pkgs/Installer.pkg",
		"/munki/icons/app.png",
		"/munki/client_resources/serial.zip",
		"/api/munki/software/1/icon",
		"/api/munki/icons/1/content",
		"/api/munki/package-installers/1/content",
		"/api/munki/client-resources/banner/1/content",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Accept-Encoding", "gzip")
		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("Content-Encoding"); got != "" {
			t.Fatalf("%s Content-Encoding = %q, want empty for storage response", path, got)
		}
	}
}

func TestCompressionMiddlewareBypassesDistributionWebSocket(t *testing.T) {
	t.Parallel()
	compression, err := compressionMiddleware()
	if err != nil {
		t.Fatalf("compressionMiddleware returned error: %v", err)
	}
	handler := compression(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("websocket-bytes", 200)))
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/munki/distribution/connect", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Upgrade", "WebSocket")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty for WebSocket upgrade", got)
	}
}

func TestCompressionMiddlewareDoesNotTrustStreamingHeadersOnOrdinaryRoutes(t *testing.T) {
	t.Parallel()
	compression, err := compressionMiddleware()
	if err != nil {
		t.Fatalf("compressionMiddleware returned error: %v", err)
	}
	handler := compression(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("api-bytes", 200)))
	}))

	for _, header := range []struct{ name, value string }{
		{name: "Accept", value: "text/event-stream"},
		{name: "Upgrade", value: "websocket"},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/hosts", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set(header.name, header.value)
		handler.ServeHTTP(rec, req)
		if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("%s spoof Content-Encoding = %q, want gzip", header.name, got)
		}
	}
}

func TestRequestTimeoutMiddlewareExemptsLongLivedResponses(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		method      string
		path        string
		headers     map[string]string
		wantExpiry  bool
		wantTimeout time.Duration
	}{
		{name: "ordinary API", method: http.MethodGet, path: "/api/hosts", wantExpiry: true, wantTimeout: time.Minute},
		{name: "spoofed SSE", method: http.MethodPost, path: "/api/hosts", headers: map[string]string{"Accept": "text/event-stream"}, wantExpiry: true, wantTimeout: time.Minute},
		{name: "spoofed WebSocket", method: http.MethodPost, path: "/api/hosts", headers: map[string]string{"Upgrade": "websocket"}, wantExpiry: true, wantTimeout: time.Minute},
		{name: "storage", path: "/storage/munki/packages/1/Installer.pkg"},
		{name: "Munki package", path: "/munki/pkgs/Installer.pkg"},
		{name: "admin content", path: "/api/munki/package-installers/1/content"},
		{name: "SSE", method: http.MethodGet, path: "/api/live-queries/1/stream"},
		{name: "WebSocket", method: http.MethodGet, path: "/api/munki/distribution/connect", headers: map[string]string{"Upgrade": "websocket"}},
		{name: "package installer finalization", method: http.MethodPut, path: "/api/munki/package-installers/1", wantExpiry: true, wantTimeout: packageInstallerTimeout},
		{name: "package installer multipart completion", method: http.MethodPost, path: "/api/munki/package-installers/1/multipart/complete", wantExpiry: true, wantTimeout: packageInstallerTimeout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotExpiry bool
			var gotTimeout time.Duration
			handler := requestTimeoutMiddleware(
				time.Minute,
			)(
				http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
					deadline, ok := req.Context().Deadline()
					gotExpiry = ok
					if ok {
						gotTimeout = time.Until(deadline)
					}
				}),
			)
			method := tc.method
			if method == "" {
				method = http.MethodGet
			}
			req := httptest.NewRequest(method, tc.path, nil)
			for name, value := range tc.headers {
				req.Header.Set(name, value)
			}
			handler.ServeHTTP(httptest.NewRecorder(), req)
			if gotExpiry != tc.wantExpiry {
				t.Fatalf("context has deadline = %t, want %t", gotExpiry, tc.wantExpiry)
			}
			if tc.wantTimeout > 0 && (gotTimeout < tc.wantTimeout-time.Second || gotTimeout > tc.wantTimeout) {
				t.Fatalf("context timeout = %s, want about %s", gotTimeout, tc.wantTimeout)
			}
		})
	}
}
