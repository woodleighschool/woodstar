// Package scope encodes label-based targeting shared by reports and checks.
package scope

import "slices"

// NormalizeLabelScope sorts label IDs, removes invalid duplicates, and collapses empty scopes.
func NormalizeLabelScope(s LabelScope) LabelScope {
	s.LabelIDs = cleanPositiveIDs(s.LabelIDs)
	switch s.Mode {
	case ScopeNone, ScopeIncludeAny, ScopeIncludeAll, ScopeExcludeAny:
	default:
		s.Mode = ScopeNone
	}
	if len(s.LabelIDs) == 0 {
		s.Mode = ScopeNone
	}
	return s
}

func cleanPositiveIDs(ids []int64) []int64 {
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
