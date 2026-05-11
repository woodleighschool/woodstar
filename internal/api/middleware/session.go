package middleware

import (
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/adminctx"
	"github.com/woodleighschool/woodstar/internal/auth"
)

// RequireAuth attaches the signed-in user to protected admin Huma operations.
func RequireAuth(api huma.API, authService *auth.Service) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		user, err := authService.CurrentUser(ctx.Context())
		if err != nil {
			if errors.Is(err, auth.ErrNotAuthenticated) {
				_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "not authenticated")
				return
			}
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "request failed")
			return
		}

		next(huma.WithContext(ctx, adminctx.WithUser(ctx.Context(), user)))
	}
}

// RequireAuthChi is the Chi-compatible counterpart to RequireAuth, used for
// non-Huma routes (the SSE stream handler) that share the same session.
func RequireAuthChi(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := authService.CurrentUser(r.Context())
			if err != nil {
				if errors.Is(err, auth.ErrNotAuthenticated) {
					http.Error(w, "not authenticated", http.StatusUnauthorized)
					return
				}
				http.Error(w, "request failed", http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(w, r.WithContext(adminctx.WithUser(r.Context(), user)))
		})
	}
}
