package middleware

import "net/http"

// CrossOriginProtection rejects cross-origin browser mutations while allowing
// same-origin browser requests and non-browser API clients.
func CrossOriginProtection() func(http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden origin"))
	}))
	return protection.Handler
}
