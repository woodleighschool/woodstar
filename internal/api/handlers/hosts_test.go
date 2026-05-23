package handlers

import (
	"testing"
)

func TestParseResourceID(t *testing.T) {
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
			got, err := parseResourceID(tt.input, hostResource)
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

func TestHostListInputParamsRejectInvalidSoftwareFilter(t *testing.T) {
	input := hostListInput{SoftwareTitleID: "nope"}

	if _, err := input.params(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHostBulkDeleteInputIDs(t *testing.T) {
	got, err := (bulkIDsBody{IDs: []int64{3, 1, 3}}).ids("host IDs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("ids = %#v, want [1 3]", got)
	}
}

func TestHostBulkDeleteInputRejectsEmptyIDs(t *testing.T) {
	if _, err := (bulkIDsBody{}).ids("host IDs"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
