package dbutil

import (
	"fmt"
	"slices"
)

// ParsePositiveIDs validates values as positive database IDs and returns a copy.
func ParsePositiveIDs(values []int64, name string) ([]int64, error) {
	out := make([]int64, len(values))
	for i, id := range values {
		if id <= 0 {
			return nil, fmt.Errorf("%w: %s includes a non-positive ID", ErrInvalidInput, name)
		}
		out[i] = id
	}
	return out, nil
}

func CleanPositiveIDs(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// CleanPositiveIDList validates values as positive database IDs, then sorts and deduplicates them.
func CleanPositiveIDList(values []int64, name string) ([]int64, error) {
	ids, err := ParsePositiveIDs(values, name)
	if err != nil {
		return nil, err
	}
	slices.Sort(ids)
	return slices.Compact(ids), nil
}

func MergePositiveIDs(a, b []int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	out := make([]int64, 0, len(a)+len(b))
	for _, ids := range [][]int64{a, b} {
		for _, id := range ids {
			if id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

// SameIDSet reports whether a and b contain the same IDs, ignoring order.
func SameIDSet(a []int64, b []int64) bool {
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
