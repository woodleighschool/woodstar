package middleware

import (
	"net/http"
	"strings"

	"github.com/woodleighschool/woodstar/internal/auth"
)

// RequireAuth protects admin API routes while leaving setup, login, and docs public.
func RequireAuth(basePath string, authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions || publicPath(basePath, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(auth.SessionCookieName)
			if err != nil {
				writeUnauthorized(w)
				return
			}
			user, err := authService.CurrentUser(r.Context(), cookie.Value)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.ContextWithUser(r.Context(), user)))
		})
	}
}

func publicPath(basePath string, path string) bool {
	if basePath != "" && basePath != "/" {
		path = strings.TrimPrefix(path, basePath)
	}
	if !strings.HasPrefix(path, "/api/") {
		return true
	}

	apiPath := strings.TrimPrefix(path, "/api")
	if strings.HasPrefix(apiPath, "/schemas/") {
		return true
	}

	switch apiPath {
	case "/healthz", "/readyz", "/version",
		"/setup/status", "/setup", "/auth/login", "/auth/logout",
		"/docs", "/openapi.json", "/openapi.yaml",
		"/openapi-3.0.json", "/openapi-3.0.yaml":
		return true
	default:
		return false
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"not authenticated"}`))
}
