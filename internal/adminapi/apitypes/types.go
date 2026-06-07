package apitypes

import (
	"errors"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// ResourceMutationError translates store errors into resource-shaped HTTP errors.
func ResourceMutationError(resource string, err error) error {
	switch {
	case errors.Is(err, dbutil.ErrNotFound):
		return huma.Error404NotFound(resource + " not found")
	case errors.Is(err, dbutil.ErrAlreadyExists):
		return huma.Error409Conflict(resource + " already exists")
	case errors.Is(err, dbutil.ErrConflict):
		return huma.Error409Conflict(
			strings.TrimPrefix(err.Error(), dbutil.ErrConflict.Error()+": "),
		)
	case errors.Is(err, dbutil.ErrInvalidInput):
		return huma.Error400BadRequest(
			strings.TrimPrefix(err.Error(), dbutil.ErrInvalidInput.Error()+": "),
		)
	default:
		return err
	}
}

// BulkIDsBody is the shared request body for bulk-delete operations.
type BulkIDsBody struct {
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

func (input ListQueryInput) Params() dbutil.ListParams {
	return dbutil.ListParams{
		Q:         input.Q,
		PageIndex: input.PageIndex,
		PageSize:  input.PageSize,
		Sort:      input.Sort,
	}
}
