package dbutil

import "strings"

const (
	defaultPerPage = 50
	maxPerPage     = 200
	orderAsc       = "asc"
	orderDesc      = "desc"
	orderSQLAsc    = "ASC"
	orderSQLDesc   = "DESC"
	OrderUpdatedAt = "updated_at"
)

// ListParams is the common query shape for paginated list endpoints.
// Page is 1-indexed: page 1 returns the first PerPage rows.
type ListParams struct {
	Q              string
	Page           int
	PerPage        int
	OrderKey       string
	OrderDirection string
}

func CleanListParams(params ListParams) ListParams {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage <= 0 {
		params.PerPage = defaultPerPage
	}
	if params.PerPage > maxPerPage {
		params.PerPage = maxPerPage
	}
	params.Q = strings.TrimSpace(params.Q)
	params.OrderKey = strings.TrimSpace(params.OrderKey)
	params.OrderDirection = strings.ToLower(strings.TrimSpace(params.OrderDirection))
	if params.OrderDirection != orderDesc {
		params.OrderDirection = orderAsc
	}
	return params
}

func SplitListValues(values []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(values))
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
