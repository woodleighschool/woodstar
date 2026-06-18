package adminapi

import (
	"errors"
	"net/http"
	"slices"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminctx"
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

func RequireHTTPAuth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			user, err := authService.Authenticate(req.Context(), req.Header.Get("Authorization"))
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, auth.ErrNotAuthenticated) {
					status = http.StatusUnauthorized
				}
				http.Error(w, http.StatusText(status), status)
				return
			}
			next.ServeHTTP(w, req.WithContext(adminctx.WithUser(req.Context(), user)))
		})
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

// RequireAdminForMutations restricts ordinary write operations to administrators.
func RequireAdminForMutations(api huma.API) func(*huma.Operation, func(*huma.Operation)) {
	return func(op *huma.Operation, next func(*huma.Operation)) {
		if mutationMethod(op.Method) {
			requireAdminOperation(api, op)
		}
		next(op)
	}
}

func requireAdminForAll(api huma.API) func(*huma.Operation, func(*huma.Operation)) {
	return func(op *huma.Operation, next func(*huma.Operation)) {
		requireAdminOperation(api, op)
		next(op)
	}
}

func requireAdminOperation(api huma.API, op *huma.Operation) {
	op.Middlewares = append(op.Middlewares, RequireAdmin(api))
	if !slices.Contains(op.Errors, http.StatusForbidden) {
		op.Errors = append(op.Errors, http.StatusForbidden)
	}
}

func mutationMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
