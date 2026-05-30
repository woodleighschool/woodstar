// Package handlers registers Huma operations for the admin API.
package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

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

type Page[T any] struct {
	Items []T `json:"items"`
	Count int `json:"count"`
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

// normalizeLabelScope applies the scope normalisation rules.
func normalizeLabelScope(s scope.LabelScope) scope.LabelScope {
	return scope.NormalizeLabelScope(s)
}

// currentUserID returns the authenticated admin's user ID, or nil if anonymous.
func currentUserID(ctx context.Context) *int64 {
	user, ok := userFromContext(ctx)
	if !ok {
		return nil
	}
	return &user.ID
}
