// Package adminctx stores admin request-scoped values.
package adminctx

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/users"
)

type key int

const userKey key = 0

// WithUser returns a copy of ctx carrying user.
func WithUser(ctx context.Context, user *users.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// UserFromContext returns the authenticated user attached to ctx.
func UserFromContext(ctx context.Context) (*users.User, bool) {
	user, ok := ctx.Value(userKey).(*users.User)
	return user, ok && user != nil
}
