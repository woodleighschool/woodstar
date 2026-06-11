package dbutil

import "strings"

const (
	defaultPageSize = 50
	maxPageSize     = 1000
	orderSQLAsc     = "ASC"
	orderSQLDesc    = "DESC"
	OrderUpdatedAt  = "updated_at"
)

// ListParams is the common query shape for paginated list endpoints.
// PageIndex is 0-indexed to match TanStack Table pagination state.
type ListParams struct {
	Q         string
	PageIndex int
	PageSize  int
	Sort      string
}

func CleanListParams(params ListParams) ListParams {
	if params.PageIndex < 0 {
		params.PageIndex = 0
	}
	if params.PageSize <= 0 {
		params.PageSize = defaultPageSize
	}
	if params.PageSize > maxPageSize {
		params.PageSize = maxPageSize
	}
	return params
}

func SplitListValues(values []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(values))
	for _, value := range values {
		for item := range strings.SplitSeq(value, ",") {
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
