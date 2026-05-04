// Package adminctx stores admin request-scoped values.
package adminctx

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/models"
)

type key int

const userKey key = 0

// WithUser returns a copy of ctx carrying user.
func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// UserFromContext returns the authenticated user attached to ctx.
func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userKey).(*models.User)
	return user, ok && user != nil
}
