package admin

import (
	"net/http"
	"time"

	"github.com/gorilla/csrf"

	"github.com/woodleighschool/woodstar/internal/config"
)

// PlaintextHTTP marks local HTTP requests so gorilla/csrf validates Origin correctly.
func PlaintextHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, csrf.PlaintextHTTPRequest(r))
	})
}

// CSRF returns the browser CSRF middleware.
func CSRF(cfg config.Config, sessionLifetime time.Duration) func(http.Handler) http.Handler {
	return csrf.Protect(
		[]byte(cfg.SessionSecret),
		csrf.Secure(cfg.IsHTTPS()),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.Path("/"),
		csrf.CookieName("woodstar_csrf"),
		csrf.MaxAge(int(sessionLifetime.Seconds())),
	)
}
