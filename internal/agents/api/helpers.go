package agentapi

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/adminctx"
	"github.com/woodleighschool/woodstar/internal/api/apihelpers"
	"github.com/woodleighschool/woodstar/internal/scope"
	"github.com/woodleighschool/woodstar/internal/users"
)

// hostGetInput is reused by checks and queries for host-scoped list routes.
type hostGetInput struct {
	ID string `path:"id"`
}

// bulkIDsBody is the shared request body for bulk-delete operations.
type bulkIDsBody struct {
	IDs []int64 `json:"ids"`
}

// normalizeLabelScope cleans label IDs and applies the scope normalisation rules.
func normalizeLabelScope(s scope.LabelScope) (scope.LabelScope, error) {
	ids, err := apihelpers.ParseIDList(s.LabelIDs, "label_ids")
	if err != nil {
		return scope.LabelScope{}, err
	}
	return scope.NormalizeLabelScope(scope.LabelScope{Mode: s.Mode, LabelIDs: ids}), nil
}

func requireAdmin(ctx context.Context) (*users.User, error) {
	user, ok := adminctx.UserFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	if user.Role != users.RoleAdmin {
		return nil, huma.Error403Forbidden("admin role required")
	}
	return user, nil
}

func currentUserID(ctx context.Context) *int64 {
	user, ok := adminctx.UserFromContext(ctx)
	if !ok {
		return nil
	}
	return &user.ID
}
