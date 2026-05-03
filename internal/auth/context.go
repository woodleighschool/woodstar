package auth

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/models"
)

type ctxKey int

const userKey ctxKey = 0

// ContextWithUser returns a copy of ctx that carries user as the authenticated actor.
func ContextWithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// UserFromContext returns the authenticated user attached by RequireAuth, if any.
func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userKey).(*models.User)
	return user, ok && user != nil
}
