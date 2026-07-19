package ctxkeys

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/directory"
)

type contextKey int

const principalContextKey contextKey = 0

// WithPrincipal attaches the authenticated app principal to ctx.
func WithPrincipal(ctx context.Context, principal *auth.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

// Principal returns the authenticated app principal from ctx.
func Principal(ctx context.Context) (*auth.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(*auth.Principal)
	return principal, ok && principal != nil
}

// RequirePrincipal returns the authenticated principal regardless of role.
func RequirePrincipal(ctx context.Context) (*auth.Principal, error) {
	principal, ok := Principal(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	return principal, nil
}

// RequireUserID returns the authenticated principal's persisted user ID.
func RequireUserID(ctx context.Context) (int64, error) {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return 0, err
	}
	if principal.UserID == nil {
		return 0, huma.Error404NotFound("account not found")
	}
	return *principal.UserID, nil
}

// RequireAdmin returns the authenticated administrator principal.
func RequireAdmin(ctx context.Context) (*auth.Principal, error) {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if principal.Role != directory.RoleAdmin {
		return nil, huma.Error403Forbidden("admin role required")
	}
	return principal, nil
}

// CurrentUserID returns the authenticated principal's persisted user ID.
func CurrentUserID(ctx context.Context) *int64 {
	principal, ok := Principal(ctx)
	if !ok {
		return nil
	}
	return principal.UserID
}
