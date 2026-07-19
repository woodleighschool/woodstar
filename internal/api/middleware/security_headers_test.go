package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersRestrictBrowserResponsesToTransferOrigin(t *testing.T) {
	t.Parallel()

	handler := SecurityHeaders("https://uploads.example")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<!doctype html>"))
		}),
	)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/hosts", nil))

	wantCSP := "default-src 'self'; base-uri 'none'; connect-src 'self' https://uploads.example; " +
		"font-src 'self'; form-action 'self'; frame-ancestors 'none'; frame-src 'none'; " +
		"img-src 'self' blob: https://uploads.example; object-src 'none'; script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'"
	if got := recorder.Header().Get("Content-Security-Policy"); got != wantCSP {
		t.Fatalf("Content-Security-Policy = %q, want %q", got, wantCSP)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := recorder.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q, want no-referrer", got)
	}
	if got := recorder.Header().Get("Permissions-Policy"); got != "camera=(), geolocation=(), microphone=()" {
		t.Fatalf("Permissions-Policy = %q, want minimal disabled browser capabilities", got)
	}
}
