package dbutil

import "testing"

func TestCleanListParamsNormalizesPaginationAndSearch(t *testing.T) {
	params := CleanListParams(ListParams{
		Q:         " mac ",
		PageIndex: -1,
		PageSize:  1200,
		Sort:      " last_seen_at.desc ",
	})

	if params.Q != "mac" {
		t.Fatalf("Q = %q, want mac", params.Q)
	}
	if params.PageIndex != 0 {
		t.Fatalf("PageIndex = %d, want 0", params.PageIndex)
	}
	if params.PageSize != 1000 {
		t.Fatalf("PageSize = %d, want 1000", params.PageSize)
	}
	if params.Sort != "last_seen_at.desc" {
		t.Fatalf("Sort = %q, want last_seen_at.desc", params.Sort)
	}
}
