package handlers

import "testing"

func TestHostListInputRejectsNonNumericSoftwareFilter(t *testing.T) {
	input := hostListInput{SoftwareTitleID: "nope"}

	if _, err := input.params(); err == nil {
		t.Fatal("expected error for non-numeric software_title_id, got nil")
	}
}
