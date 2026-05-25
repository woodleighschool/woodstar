package dbutil

import "testing"

func TestCleanListParamsClampsPagination(t *testing.T) {
	params := CleanListParams(ListParams{
		PageIndex: -1,
		PageSize:  1200,
	})

	if params.PageIndex != 0 {
		t.Fatalf("PageIndex = %d, want 0", params.PageIndex)
	}
	if params.PageSize != 1000 {
		t.Fatalf("PageSize = %d, want 1000", params.PageSize)
	}
}

func TestCleanListParamsDefaultsPageSize(t *testing.T) {
	params := CleanListParams(ListParams{})
	if params.PageSize != 50 {
		t.Fatalf("PageSize = %d, want 50 default", params.PageSize)
	}
}
