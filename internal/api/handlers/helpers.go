package handlers

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/adminctx"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

func parsePositiveIDValue(id string) (int64, bool) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	return parsed, err == nil && parsed > 0
}

// parseResourceID parses a resource path ID. Returns 404 with a resource-named
// message on failure so probes never see shape details for missing resources.
func parseResourceID(id string, resource string) (int64, error) {
	parsed, ok := parsePositiveIDValue(id)
	if !ok {
		return 0, huma.Error404NotFound(resource + " not found")
	}
	return parsed, nil
}

// parseOptionalPositiveID parses an optional positive integer query/path value.
// Empty input returns (0, nil).
func parseOptionalPositiveID(id string, name string) (int64, error) {
	if id == "" {
		return 0, nil
	}
	parsed, ok := parsePositiveIDValue(id)
	if !ok {
		return 0, huma.Error400BadRequest(name + " must be a positive integer")
	}
	return parsed, nil
}

// parseIDList validates every element as a positive int64.
func parseIDList(values []int64, name string) ([]int64, error) {
	ids := make([]int64, 0, len(values))
	for _, id := range values {
		if id <= 0 {
			return nil, huma.Error400BadRequest(name + " includes a non-positive ID")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// cleanBulkIDs validates, deduplicates, and sorts IDs for bulk operations.
func cleanBulkIDs(values []int64, name string) ([]int64, error) {
	ids, err := parseIDList(values, name)
	if err != nil {
		return nil, err
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) == 0 {
		return nil, huma.Error400BadRequest(name + " must include at least one ID")
	}
	return ids, nil
}

// resourceMutationError translates store errors into resource-shaped HTTP errors.
func resourceMutationError(resource string, err error) error {
	switch {
	case errors.Is(err, dbutil.ErrNotFound):
		return huma.Error404NotFound(resource + " not found")
	case errors.Is(err, dbutil.ErrAlreadyExists):
		return huma.Error409Conflict(resource + " already exists")
	case errors.Is(err, dbutil.ErrInvalidInput):
		return huma.Error400BadRequest(strings.TrimPrefix(err.Error(), dbutil.ErrInvalidInput.Error()+": "))
	default:
		return err
	}
}

// bulkIDsBody is the shared request body for bulk-delete operations.
type bulkIDsBody struct {
	IDs []int64 `json:"ids"`
}

func (body bulkIDsBody) ids(name string) ([]int64, error) {
	return cleanBulkIDs(body.IDs, name)
}

// normalizeLabelScope cleans label IDs and applies the scope normalisation rules.
func normalizeLabelScope(s scope.LabelScope) (scope.LabelScope, error) {
	ids, err := parseIDList(s.LabelIDs, "label_ids")
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
