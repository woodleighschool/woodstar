package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCrossOriginProtectionAllowsTrustedOriginMutation(t *testing.T) {
	t.Parallel()
	handler := CrossOriginProtection(
		[]string{"https://panel.example.com"},
	)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/example", nil)
	req.Host = "api.example.com"
	req.Header.Set("Origin", "https://panel.example.com")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestCrossOriginProtectionRejectsUntrustedOriginMutation(t *testing.T) {
	t.Parallel()
	handler := CrossOriginProtection(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/example", nil)
	req.Host = "api.example.com"
	req.Header.Set("Origin", "https://panel.example.com")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
