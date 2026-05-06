package handlers

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/models"
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
			got, err := parseHostID(tt.input)
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
		Platform:        " darwin ",
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
	if got.Platform != "darwin" || got.SoftwareTitleID != 12 || got.SoftwareID != 40 {
		t.Fatalf("params = %#v", got)
	}
}

func TestHostListInputParamsRejectInvalidSoftwareFilter(t *testing.T) {
	input := hostListInput{SoftwareTitleID: "nope"}

	if _, err := input.params(); err == nil {
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

func TestHostSoftwareResponseGroupsInstalledVersions(t *testing.T) {
	row := models.HostSoftwareRow{
		ID:           7,
		Name:         "Example",
		DisplayName:  "Example",
		Source:       "apps",
		ExtensionFor: "",
		InstalledVersions: []models.HostSoftwareInstalledVersion{{
			Version:          "1.2.3",
			BundleIdentifier: "com.example.app",
			InstalledPaths:   []string{"/Applications/Example.app"},
			SignatureInformation: []models.PathSignatureInformation{{
				InstalledPath:    "/Applications/Example.app",
				TeamIdentifier:   "ABCD123456",
				CDHashSHA256:     "cdhash",
				ExecutableSHA256: "executable",
				ExecutablePath:   "/Applications/Example.app/Contents/MacOS/Example",
			}},
		}},
	}

	got := hostSoftwareResponse(row)
	if got.ID != "7" || got.Status != nil || got.SoftwarePackage != nil || got.AppStoreApp != nil {
		t.Fatalf("unexpected top-level response: %#v", got)
	}
	if len(got.InstalledVersions) != 1 {
		t.Fatalf("installed_versions len = %d, want 1", len(got.InstalledVersions))
	}
	if got.InstalledVersions[0].InstalledPaths[0] != "/Applications/Example.app" {
		t.Fatalf("installed path = %#v", got.InstalledVersions[0].InstalledPaths)
	}
	if got.InstalledVersions[0].SignatureInformation[0].TeamIdentifier != "ABCD123456" {
		t.Fatalf("signature info = %#v", got.InstalledVersions[0].SignatureInformation)
	}
}
