package adminapi

import (
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/adminctx"
	"github.com/woodleighschool/woodstar/internal/auth"
)

// RequireAuth attaches the signed-in user to protected admin Huma operations.
// Accepts an "Authorization: Bearer <api-key>" header first and falls back to
// the scs session cookie when no Bearer token is present.
func RequireAuth(api huma.API, authService *auth.Service) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		user, err := authService.Authenticate(ctx.Context(), ctx.Header("Authorization"))
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

// RequireAdmin rejects authenticated users that are not administrators.
func RequireAdmin(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if _, err := adminctx.RequireAdmin(ctx.Context()); err != nil {
			var statusErr huma.StatusError
			if errors.As(err, &statusErr) {
				_ = huma.WriteErr(api, ctx, statusErr.GetStatus(), err.Error())
				return
			}
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "request failed")
			return
		}
		next(ctx)
	}
}
