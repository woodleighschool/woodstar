package listing

import "testing"

func TestNormalizeDefaultsPageSize(t *testing.T) {
	params := Normalize(Params{})
	if params.PageSize != 50 {
		t.Fatalf("PageSize = %d, want 50 default", params.PageSize)
	}
}

func TestValidateRejectsInvalidPagination(t *testing.T) {
	tests := []Params{
		{PageIndex: -1, PageSize: 50},
		{PageSize: -1},
		{PageSize: 1001},
	}
	for _, params := range tests {
		if err := Validate(params); err == nil {
			t.Fatalf("Validate(%+v) returned nil error", params)
		}
	}
}

func TestNormalizeValuesTrimsDropsEmptyAndDeduplicates(t *testing.T) {
	values := NormalizeValues([]string{" orbit, munki ", "", "orbit"})
	if len(values) != 2 || values[0] != "orbit" || values[1] != "munki" {
		t.Fatalf("NormalizeValues() = %#v", values)
	}
}
