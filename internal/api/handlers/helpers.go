package handlers

import (
	"context"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/adminctx"
	"github.com/woodleighschool/woodstar/internal/api/apihelpers"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// parseOptionalPositiveID parses an optional positive integer query/path value.
// Empty input returns (0, nil).
func parseOptionalPositiveID(id string, name string) (int64, error) {
	if id == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, huma.Error400BadRequest(name + " must be a positive integer")
	}
	return parsed, nil
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

// currentUserID returns the authenticated admin's user ID, or nil if anonymous.
func currentUserID(ctx context.Context) *int64 {
	user, ok := adminctx.UserFromContext(ctx)
	if !ok {
		return nil
	}
	return &user.ID
}
