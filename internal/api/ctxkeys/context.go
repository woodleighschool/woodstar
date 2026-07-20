// Package ctxkeys defines typed request-context keys shared by API layers.
package ctxkeys

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/directory"
)

type contextKey int

const userContextKey contextKey = 0

// WithUser attaches the authenticated app API user to ctx.
func WithUser(ctx context.Context, user *directory.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// User returns the authenticated app API user from ctx.
func User(ctx context.Context) (*directory.User, bool) {
	user, ok := ctx.Value(userContextKey).(*directory.User)
	return user, ok && user != nil
}

// RequireUser returns the authenticated user regardless of role.
func RequireUser(ctx context.Context) (*directory.User, error) {
	user, ok := User(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	return user, nil
}

// RequireAdmin returns the authenticated administrator.
func RequireAdmin(ctx context.Context) (*directory.User, error) {
	user, err := RequireUser(ctx)
	if err != nil {
		return nil, err
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
