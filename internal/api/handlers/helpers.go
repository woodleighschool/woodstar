// Package handlers registers Huma operations for the admin API.
package handlers

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

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
	ids, err := dbutil.ParsePositiveIDs(values, name)
	if err != nil {
		return nil, huma.Error400BadRequest(name + " includes a non-positive ID")
	}
	return ids, nil
}

// cleanBulkIDs validates, deduplicates, and sorts IDs for bulk operations.
func cleanBulkIDs(values []int64, name string) ([]int64, error) {
	ids, err := dbutil.CleanPositiveIDList(values, name)
	if err != nil {
		return nil, huma.Error400BadRequest(name + " includes a non-positive ID")
	}
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
		return huma.Error400BadRequest(
			strings.TrimPrefix(err.Error(), dbutil.ErrInvalidInput.Error()+": "),
		)
	default:
		return err
	}
}

// bulkIDsBody is the shared request body for bulk-delete operations.
type bulkIDsBody struct {
	IDs []int64 `json:"ids"`
}

type paginatedBody[T any] struct {
	Items []T `json:"items"`
	Count int `json:"count"`
}

type itemsBody[T any] struct {
	Items []T `json:"items"`
}

type ListQueryInput struct {
	Q         string `query:"q,omitempty"`
	PageIndex int    `query:"page_index,omitempty" minimum:"0"`
	PageSize  int    `query:"page_size,omitempty"  minimum:"1" maximum:"1000"`
	Sort      string `query:"sort,omitempty"                                  example:"name.asc"`
}

func (input ListQueryInput) params() dbutil.ListParams {
	return dbutil.ListParams{
		Q:         input.Q,
		PageIndex: input.PageIndex,
		PageSize:  input.PageSize,
		Sort:      input.Sort,
	}
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
	user, ok := userFromContext(ctx)
	if !ok {
		return nil
	}
	return &user.ID
}
