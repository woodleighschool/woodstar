package ids

import (
	"fmt"
	"slices"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func CleanLabelIDs(values []int64, name string) ([]int64, error) {
	ids, err := ParsePositive(values, name)
	if err != nil {
		return nil, err
	}
	slices.Sort(ids)
	return slices.Compact(ids), nil
}

func ParsePositive(values []int64, name string) ([]int64, error) {
	out := make([]int64, len(values))
	for i, id := range values {
		if id <= 0 {
			return nil, fmt.Errorf("%w: %s includes a non-positive ID", dbutil.ErrInvalidInput, name)
		}
		out[i] = id
	}
	return out, nil
}

func SameSet(a []int64, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[int64]struct{}, len(a))
	for _, id := range a {
		seen[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := seen[id]; !ok {
			return false
		}
	}
	return true
}
