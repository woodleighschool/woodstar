package middleware

import (
	"net/http"
	"strings"
)

const (
	permissionsPolicy = "camera=(), geolocation=(), microphone=()"
	referrerPolicy    = "no-referrer"
)

// SecurityHeaders restricts browser capabilities to Woodstar and its storage transfer origin.
func SecurityHeaders(transferOrigin string) func(http.Handler) http.Handler {
	connectSources := []string{"'self'"}
	imageSources := []string{"'self'", "blob:"}
	if transferOrigin != "" {
		connectSources = append(connectSources, transferOrigin)
		imageSources = append(imageSources, transferOrigin)
	}
	csp := strings.Join([]string{
		"default-src 'self'",
		"base-uri 'none'",
		"connect-src " + strings.Join(connectSources, " "),
		"font-src 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
		"frame-src 'none'",
		"img-src " + strings.Join(imageSources, " "),
		"object-src 'none'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
	}, "; ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", csp)
			w.Header().Set("Permissions-Policy", permissionsPolicy)
			w.Header().Set("Referrer-Policy", referrerPolicy)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			next.ServeHTTP(w, r)
		})
	}
}
