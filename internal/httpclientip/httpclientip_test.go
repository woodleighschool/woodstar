package httpclientip

import (
	"net/http"
	"testing"
)

func TestFromRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/osquery/config", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "192.0.2.10:1234"
	if got := FromRequest(req); got != "192.0.2.10" {
		t.Fatalf("FromRequest remote addr = %q, want 192.0.2.10", got)
	}

	req.RemoteAddr = "2001:db8::1"
	if got := FromRequest(req); got != "2001:db8::1" {
		t.Fatalf("FromRequest normalized IPv6 remote addr = %q, want 2001:db8::1", got)
	}

	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.20, 203.0.113.30")
	req.Header.Set("X-Real-IP", "2001:db8::1")
	if got := FromRequest(req); got != "192.0.2.10" {
		t.Fatalf("FromRequest should use RealIP-normalized remote addr, got %q", got)
	}
}
