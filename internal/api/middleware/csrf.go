package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/csrf"

	"github.com/woodleighschool/woodstar/internal/config"
)

// CSRF returns the browser CSRF middleware.
func CSRF(cfg config.Config, sessionLifetime time.Duration) func(http.Handler) http.Handler {
	protect := csrf.Protect(
		[]byte(cfg.SessionSecret),
		csrf.Secure(cfg.IsHTTPS()),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.Path("/"),
		csrf.CookieName("woodstar_csrf"),
		csrf.MaxAge(int(sessionLifetime.Seconds())),
	)
	return func(next http.Handler) http.Handler {
		protected := protect(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if hasBearerToken(r.Header.Get("Authorization")) {
				next.ServeHTTP(w, r)
				return
			}
			if !cfg.IsHTTPS() {
				r = csrf.PlaintextHTTPRequest(r)
			}
			protected.ServeHTTP(w, r)
		})
	}
}

func hasBearerToken(authHeader string) bool {
	const prefix = "Bearer "
	if len(authHeader) <= len(prefix) {
		return false
	}
	if !strings.EqualFold(authHeader[:len(prefix)], prefix) {
		return false
	}
	return strings.TrimSpace(authHeader[len(prefix):]) != ""
}
