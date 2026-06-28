package middleware

import "net/http"

// CrossOriginProtection rejects untrusted cross-origin browser mutations while
// allowing same-origin browser requests and non-browser API clients.
func CrossOriginProtection(trustedOrigins []string) func(http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()
	for _, origin := range trustedOrigins {
		// Config validation keeps this list compatible with net/http.
		_ = protection.AddTrustedOrigin(origin)
	}
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden origin"))
	}))
	return protection.Handler
}
