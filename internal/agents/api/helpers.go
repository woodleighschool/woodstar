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

// labelScopeBody is the shared request/response shape for label scope fields.
type labelScopeBody struct {
	Mode     scope.LabelScopeMode `json:"mode,omitempty"      enum:"include_any,include_all,exclude_any"`
	LabelIDs []int64              `json:"label_ids,omitempty"`
}

func (body labelScopeBody) model() (scope.LabelScope, error) {
	ids, err := apihelpers.ParseIDList(body.LabelIDs, "label_ids")
	if err != nil {
		return scope.LabelScope{}, err
	}
	return scope.NormalizeLabelScope(scope.LabelScope{Mode: body.Mode, LabelIDs: ids}), nil
}

func labelScopeResponse(s scope.LabelScope) labelScopeBody {
	return labelScopeBody{Mode: s.Mode, LabelIDs: append([]int64{}, s.LabelIDs...)}
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
