package destination

import (
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestAPICreateDoesNotRetry(t *testing.T) {
	var attempts atomic.Int32
	remote, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	if err := remote.request(t.Context(), http.MethodPost, "/api/munki/software", map[string]any{"name": "Example"}, nil); err == nil || attempts.Load() != 1 {
		t.Fatalf("non-idempotent create attempts=%d err=%v", attempts.Load(), err)
	}
}

func TestAPIResponsePreservesLargeIDs(t *testing.T) {
	remote, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":9007199254740993}`)
	}))
	var result map[string]any
	if err := remote.request(t.Context(), http.MethodGet, "/api/munki/software/9007199254740993", nil, &result); err != nil {
		t.Fatal(err)
	}
	if id, _ := result["id"].(json.Number); id.String() != "9007199254740993" {
		t.Fatalf("ID lost precision: %v", result)
	}
}

// testClient connects to a TLS server for handler and returns the server's origin.
func testClient(t *testing.T, handler http.Handler) (*Client, string) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	remote, err := New(Config{URL: server.URL, APIKey: "synthetic-key", CAFile: caPath})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = remote.Close() })
	return remote, server.URL
}
