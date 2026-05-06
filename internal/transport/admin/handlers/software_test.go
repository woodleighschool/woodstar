package handlers

import (
	"reflect"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/models"
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

func TestSoftwareTitleResponseUsesFleetShape(t *testing.T) {
	updated := time.Date(2026, 5, 5, 1, 2, 3, 0, time.UTC)
	title := models.SoftwareTitle{
		ID:              12,
		Name:            "Example Extension",
		DisplayName:     "Example Extension",
		Source:          "chrome_extensions",
		ExtensionFor:    "arc",
		HostsCount:      3,
		VersionsCount:   2,
		CountsUpdatedAt: &updated,
		Versions: []models.SoftwareVersion{
			{ID: 40, Version: "1.0.0", HostsCount: 1},
			{ID: 41, Version: "1.1.0", HostsCount: 2},
		},
	}

	got := softwareTitleResponse(title)
	if got.ID != "12" || got.Browser != "arc" {
		t.Fatalf("unexpected response identifiers: %#v", got)
	}
	if got.SoftwarePackage != nil || got.AppStoreApp != nil {
		t.Fatalf("package/app store placeholders = %#v %#v, want nil", got.SoftwarePackage, got.AppStoreApp)
	}
	if got.HostsCount != 3 || got.VersionsCount != 2 || got.CountsUpdatedAt == nil {
		t.Fatalf("unexpected counts: %#v", got)
	}
	if len(got.Versions) != 2 || got.Versions[1].Version != "1.1.0" {
		t.Fatalf("versions = %#v", got.Versions)
	}
	if got.Versions[1].HostsCount != 2 {
		t.Fatalf("version hosts_count = %d, want 2", got.Versions[1].HostsCount)
	}
}
