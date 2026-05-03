package handlers

import (
	"testing"
	"time"
)

func TestSessionCookieUsesConfiguredPathAndSecureFlag(t *testing.T) {
	cookie := sessionCookie("token", time.Now().Add(time.Hour), CookieSettings{
		CookiePath:   "/woodstar",
		SecureCookie: true,
	})

	if cookie.Path != "/woodstar" {
		t.Fatalf("Path = %q, want /woodstar", cookie.Path)
	}
	if !cookie.Secure {
		t.Fatal("Secure = false, want true")
	}
}
