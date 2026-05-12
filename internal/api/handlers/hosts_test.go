package handlers

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/api/apihelpers"
)

func TestParseHostID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{name: "positive", input: "12", want: 12},
		{name: "zero rejected", input: "0", wantErr: true},
		{name: "negative rejected", input: "-1", wantErr: true},
		{name: "non-numeric rejected", input: "abc", wantErr: true},
		{name: "empty rejected", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := apihelpers.ParseHostID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHostListInputParams(t *testing.T) {
	input := hostListInput{
		Q:               " laptop ",
		Page:            2,
		PerPage:         25,
		OrderKey:        "display_name",
		OrderDirection:  "desc",
		Status:          " online ",
		Platform:        " darwin ",
		LabelID:         "7",
		SoftwareTitleID: "12",
		SoftwareID:      "40",
	}

	got, err := input.params()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Q != "laptop" || got.Page != 2 || got.PerPage != 25 {
		t.Fatalf("list params = %#v", got.ListParams)
	}
	if got.OrderKey != "display_name" || got.OrderDirection != "desc" {
		t.Fatalf("sort params = %#v", got.ListParams)
	}
	if got.Status != "online" || got.Platform != "darwin" || got.LabelID != 7 ||
		got.SoftwareTitleID != 12 || got.SoftwareID != 40 {
		t.Fatalf("params = %#v", got)
	}
}

func TestHostListInputParamsRejectInvalidSoftwareFilter(t *testing.T) {
	input := hostListInput{SoftwareTitleID: "nope"}

	if _, err := input.params(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHostBulkDeleteInputIDs(t *testing.T) {
	input := hostBulkDeleteInput{Body: hostBulkIDsBody{IDs: []int64{3, 1, 3}}}

	got, err := input.ids()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("ids = %#v, want [1 3]", got)
	}
}

func TestHostBulkDeleteInputRejectsEmptyIDs(t *testing.T) {
	input := hostBulkDeleteInput{Body: hostBulkIDsBody{}}

	if _, err := input.ids(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHostSoftwareInputParams(t *testing.T) {
	input := hostSoftwareInput{
		ID:             "9",
		Q:              " 1Password ",
		Page:           1,
		PerPage:        100,
		OrderKey:       "name",
		OrderDirection: "asc",
		Source:         []string{"apps,chrome_extensions", "apps"},
	}

	id, params, err := input.params()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 9 {
		t.Fatalf("id = %d, want 9", id)
	}
	if params.Q != "1Password" || params.Page != 1 || params.PerPage != 100 {
		t.Fatalf("list params = %#v", params.ListParams)
	}
	if params.OrderKey != "name" || params.OrderDirection != "asc" {
		t.Fatalf("sort params = %#v", params.ListParams)
	}
	if len(params.SoftwareSources) != 2 || params.SoftwareSources[0] != "apps" ||
		params.SoftwareSources[1] != "chrome_extensions" {
		t.Fatalf("sources = %#v", params.SoftwareSources)
	}
}
