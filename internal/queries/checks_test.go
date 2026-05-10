package queries

import "testing"

func TestCleanCheckCreate(t *testing.T) {
	got, err := cleanCheckCreate(CheckCreate{
		Name:        " Gatekeeper enabled ",
		Description: " Security check ",
		Query:       " select 1 from gatekeeper where assessments_enabled = 1; ",
		Platform:    new(" darwin "),
	})
	if err != nil {
		t.Fatalf("cleanCheckCreate returned error: %v", err)
	}
	if got.Name != "Gatekeeper enabled" {
		t.Fatalf("Name = %q, want Gatekeeper enabled", got.Name)
	}
	if got.Query != "select 1 from gatekeeper where assessments_enabled = 1;" {
		t.Fatalf("Query = %q, want trimmed SQL", got.Query)
	}
	assertStringPtr(t, "Platform", got.Platform, new("darwin"))
}
