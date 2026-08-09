// Package listing defines the shared request semantics for searchable,
// sortable, paginated application lists.
package listing

import (
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/validation"
)

const defaultPageSize = 50

// Params is the common request shape for paginated lists.
// PageIndex is zero-indexed to match TanStack Table pagination state.
type Params struct {
	Q         string
	PageIndex int32 `validate:"gte=0"`
	PageSize  int32 `validate:"gte=1,lte=1000"`
	Sort      string
}

// Normalize applies pagination defaults and trims text fields.
func Normalize(params Params) Params {
	params.Q = strings.TrimSpace(params.Q)
	params.Sort = strings.TrimSpace(params.Sort)
	if params.PageSize == 0 {
		params.PageSize = defaultPageSize
	}
	return params
}

// Validate checks pagination bounds after normalization.
func Validate(params Params) error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return nil
}

// NormalizeValues splits comma-separated values, trims them, removes empty
// values, and preserves the first occurrence of each value.
func NormalizeValues[T ~string](values []T) []T {
	out := make([]T, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		for item := range strings.SplitSeq(string(value), ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			out = append(out, T(item))
		}
	}
	return out
}
