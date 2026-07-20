// Package middleware provides authentication, logging, and browser HTTP policy.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/directory"
)

// Authenticator resolves a browser session or API key into a Woodstar user.
type Authenticator interface {
	Authenticate(ctx context.Context, authHeader string) (*directory.User, error)
}

// OptionalHumaAuth attaches a user to the Huma context when credentials are
// present and valid. Missing credentials are allowed; invalid or broken
// credentials keep their normal auth failure semantics.
func OptionalHumaAuth(api huma.API, authenticator Authenticator) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		user, err := authenticator.Authenticate(ctx.Context(), ctx.Header("Authorization"))
		if err == nil {
			next(huma.WithContext(ctx, ctxkeys.WithUser(ctx.Context(), user)))
			return
		}
		if errors.Is(err, auth.ErrNotAuthenticated) {
			next(ctx)
			return
		}
		_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "request failed")
	}
}

// RequireHumaAuth attaches the authenticated user to protected Huma operations.
func RequireHumaAuth(api huma.API, authenticator Authenticator) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		user, err := authenticator.Authenticate(ctx.Context(), ctx.Header("Authorization"))
		if err != nil {
			if errors.Is(err, auth.ErrNotAuthenticated) {
				_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "not authenticated")
				return
			}
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "request failed")
			return
		}

		next(huma.WithContext(ctx, ctxkeys.WithUser(ctx.Context(), user)))
	}
}

// ProtectedOperation declares the authentication contract shared by protected
// Huma operations. Runtime authentication is applied separately by
// RequireHumaAuth.
func ProtectedOperation(api huma.API) func(*huma.Operation, func(*huma.Operation)) {
	return func(op *huma.Operation, next func(*huma.Operation)) {
		op.Security = []map[string][]string{
			{"cookieAuth": {}},
			{"bearerAuth": {}},
		}
		declareErrorResponse(api, op, http.StatusUnauthorized)
		next(op)
	}
}

// RequireHTTPAuth attaches the authenticated user to raw HTTP routes.
func RequireHTTPAuth(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			user, err := authenticator.Authenticate(req.Context(), req.Header.Get("Authorization"))
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, auth.ErrNotAuthenticated) {
					status = http.StatusUnauthorized
				}
				http.Error(w, http.StatusText(status), status)
				return
			}
			next.ServeHTTP(w, req.WithContext(ctxkeys.WithUser(req.Context(), user)))
		})
	}
}

// requireAdmin rejects authenticated users that are not administrators.
func requireAdmin(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if _, err := ctxkeys.RequireAdmin(ctx.Context()); err != nil {
			if statusErr, ok := errors.AsType[huma.StatusError](err); ok {
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

// RequireAdminForAll restricts every operation in a group to administrators.
func RequireAdminForAll(api huma.API) func(*huma.Operation, func(*huma.Operation)) {
	return func(op *huma.Operation, next func(*huma.Operation)) {
		requireAdminOperation(api, op)
		next(op)
	}
}

func requireAdminOperation(api huma.API, op *huma.Operation) {
	op.Middlewares = append(op.Middlewares, requireAdmin(api))
	declareErrorResponse(api, op, http.StatusForbidden)
}

func declareErrorResponse(api huma.API, op *huma.Operation, status int) {
	key := strconv.Itoa(status)
	if op.Responses[key] != nil {
		return
	}
	op.Responses[key] = &huma.Response{
		Description: http.StatusText(status),
		Content: map[string]*huma.MediaType{
			"application/problem+json": {
				Schema: api.OpenAPI().Components.Schemas.Schema(
					reflect.TypeFor[huma.ErrorModel](),
					true,
					"Error",
				),
			},
		},
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
