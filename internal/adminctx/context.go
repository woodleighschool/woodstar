package adminctx

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/directory"
)

type contextKey int

const userContextKey contextKey = 0

// WithUser attaches the authenticated admin API user to ctx.
func WithUser(ctx context.Context, user *directory.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// User returns the authenticated admin API user from ctx.
func User(ctx context.Context) (*directory.User, bool) {
	user, ok := ctx.Value(userContextKey).(*directory.User)
	return user, ok && user != nil
}

// RequireUser returns the authenticated user from ctx regardless of role.
func RequireUser(ctx context.Context) (*directory.User, error) {
	user, ok := User(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	return user, nil
}

// RequireAdmin returns the authenticated admin user from ctx.
func RequireAdmin(ctx context.Context) (*directory.User, error) {
	user, ok := User(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	if user.Role == nil || *user.Role != directory.RoleAdmin {
		return nil, huma.Error403Forbidden("admin role required")
	}
	return user, nil
}

// CurrentUserID returns the authenticated user's ID, or nil if anonymous.
func CurrentUserID(ctx context.Context) *int64 {
	user, ok := User(ctx)
	if !ok {
		return nil
	}
	return &user.ID
}
