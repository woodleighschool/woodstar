package dbutil

import "testing"

func TestNormalizeListParamsDefaultsPageSize(t *testing.T) {
	params := NormalizeListParams(ListParams{})
	if params.PageSize != 50 {
		t.Fatalf("PageSize = %d, want 50 default", params.PageSize)
	}
}

func TestValidateListParamsRejectsInvalidPagination(t *testing.T) {
	tests := []ListParams{
		{PageIndex: -1, PageSize: 50},
		{PageSize: -1},
		{PageSize: 1001},
	}
	for _, params := range tests {
		if err := ValidateListParams(params); err == nil {
			t.Fatalf("ValidateListParams(%+v) returned nil error", params)
		}
	}
}

func TestNormalizeListValuesTrimsDropsEmptyAndDeduplicates(t *testing.T) {
	values := NormalizeListValues([]string{" orbit, munki ", "", "orbit"})
	if len(values) != 2 || values[0] != "orbit" || values[1] != "munki" {
		t.Fatalf("NormalizeListValues() = %#v", values)
	}
}
