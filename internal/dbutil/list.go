package dbutil

import (
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/validation"
)

const (
	defaultPageSize = 50
	orderSQLAsc     = "ASC"
	orderSQLDesc    = "DESC"
)

// ListParams is the common query shape for paginated list endpoints.
// PageIndex is 0-indexed to match TanStack Table pagination state.
type ListParams struct {
	Q         string
	PageIndex int32 `validate:"gte=0"`
	PageSize  int32 `validate:"gte=1,lte=1000"`
	Sort      string
}

// NormalizeListParams applies pagination defaults and trims text fields.
func NormalizeListParams(params ListParams) ListParams {
	params.Q = strings.TrimSpace(params.Q)
	params.Sort = strings.TrimSpace(params.Sort)
	if params.PageSize == 0 {
		params.PageSize = defaultPageSize
	}
	return params
}

// ValidateListParams checks pagination bounds after normalization.
func ValidateListParams(params ListParams) error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}
	return nil
}

// NormalizeListValues splits, trims, removes empty values, and deduplicates a list.
func NormalizeListValues(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		for item := range strings.SplitSeq(value, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			out = append(out, item)
		}
	}
	return out
}
