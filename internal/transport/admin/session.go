package admin

import (
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/transport/admin/adminctx"
)

// RequireAuth attaches the signed-in user to protected admin operations.
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
