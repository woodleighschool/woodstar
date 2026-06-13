package apitypes

import (
	"encoding/json"
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
	Items []T `json:"items" nullable:"false"`
	Count int `json:"count"`
}

// MarshalJSON renders an empty page as items: [] rather than null, matching the
// non-nullable items schema so the frontend never sees null for an empty list.
func (p Page[T]) MarshalJSON() ([]byte, error) {
	items := p.Items
	if items == nil {
		items = make([]T, 0)
	}
	return json.Marshal(struct {
		Items []T `json:"items"`
		Count int `json:"count"`
	}{Items: items, Count: p.Count})
}

// ListQueryInput is the shared query contract for paginated list endpoints. It
// carries optional `q`, 1-based `page`, `per_page`, and a single `sort` token
// such as `name.asc` or `last_seen_at.desc`. Per-resource filters are added as
// their own query fields, keyed by column ID.
type ListQueryInput struct {
	Q       string `query:"q,omitempty"`
	Page    int    `query:"page,omitempty"     minimum:"1"`
	PerPage int    `query:"per_page,omitempty" minimum:"1" maximum:"1000"`
	Sort    string `query:"sort,omitempty"`
}

func (input ListQueryInput) Params() dbutil.ListParams {
	pageIndex := 0
	if input.Page > 1 {
		pageIndex = input.Page - 1
	}
	return dbutil.ListParams{
		Q:         input.Q,
		PageIndex: pageIndex,
		PageSize:  input.PerPage,
		Sort:      input.Sort,
	}
}
