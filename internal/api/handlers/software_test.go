package handlers

import (
	"reflect"
	"testing"
)

func TestSoftwareListInputParams(t *testing.T) {
	input := softwareListInput{
		Q:              " 1Password ",
		Page:           3,
		PerPage:        100,
		OrderKey:       "hosts_count",
		OrderDirection: "desc",
		Source:         []string{"apps,chrome_extensions", "apps"},
	}

	got := input.params()
	if got.Q != "1Password" || got.Page != 3 || got.PerPage != 100 {
		t.Fatalf("list params = %#v", got.ListParams)
	}
	if got.OrderKey != "hosts_count" || got.OrderDirection != "desc" {
		t.Fatalf("sort params = %#v", got.ListParams)
	}
	if !reflect.DeepEqual(got.SoftwareSources, []string{"apps", "chrome_extensions"}) {
		t.Fatalf("sources = %#v", got.SoftwareSources)
	}
}
