package osquery

import (
	"testing"
	"time"
)

func TestParseSoftwareRows(t *testing.T) {
	rows := []map[string]string{
		{
			"name":                 "Safari",
			"version":              "26.0",
			"source":               "apps",
			"bundle_identifier":    "com.apple.Safari",
			"installed_path":       "/Applications/Safari.app",
			"last_opened_at":       "1745999192.82046",
			"bundle_short_version": "ignored",
		},
		{
			"name":              "node",
			"version":           "24.0.0",
			"source":            "homebrew_packages",
			"last_opened_at":    "",
			"bundle_identifier": "",
		},
		{"name": ""},
	}

	got := parseSoftwareRows(rows, softwareEnrichment{})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "Safari" || got[0].Source != "apps" || got[0].InstalledPath != "/Applications/Safari.app" {
		t.Fatalf("first row parsed incorrectly: %#v", got[0])
	}
	wantOpened := time.Unix(1745999192, 820460000).UTC()
	if got[0].LastOpenedAt == nil || !got[0].LastOpenedAt.Equal(wantOpened) {
		t.Fatalf("LastOpenedAt = %v, want %v", got[0].LastOpenedAt, wantOpened)
	}
	if got[1].LastOpenedAt != nil {
		t.Fatalf("second LastOpenedAt = %v, want nil", got[1].LastOpenedAt)
	}
}

func TestParseSoftwareRowsEnrichesInstalledPaths(t *testing.T) {
	rows := []map[string]string{
		{
			"name":              "Example",
			"version":           "1.2.3",
			"source":            "apps",
			"bundle_identifier": "com.example.app",
			"installed_path":    "/Applications/Example.app",
		},
	}
	enrichment := softwareEnrichmentByPath([]map[string]string{{
		"path":            "/Applications/Example.app",
		"team_identifier": "ABCD123456",
		"cdhash_sha256":   "cdhash",
	}}, []map[string]string{{
		"path":              "/Applications/Example.app",
		"executable_sha256": "executable-hash",
		"executable_path":   "/Applications/Example.app/Contents/MacOS/Example",
		"unrelated_ignored": "ignored",
	}})

	got := parseSoftwareRows(rows, enrichment)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].TeamIdentifier != "ABCD123456" || got[0].CDHashSHA256 != "cdhash" ||
		got[0].ExecutableSHA256 != "executable-hash" ||
		got[0].ExecutablePath != "/Applications/Example.app/Contents/MacOS/Example" {
		t.Fatalf("enrichment parsed incorrectly: %#v", got[0])
	}
}
