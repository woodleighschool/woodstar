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
			"source":               "app",
			"bundle_identifier":    "com.apple.Safari",
			"path":                 "/Applications/Safari.app",
			"last_opened_time":     "1745999192",
			"bundle_short_version": "ignored",
		},
		{
			"name":              "node",
			"version":           "24.0.0",
			"source":            "brew",
			"last_opened_time":  "",
			"bundle_identifier": "",
		},
		{"name": ""},
	}

	got := parseSoftwareRows(rows)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "Safari" || got[0].Source != "app" || got[0].InstalledPath != "/Applications/Safari.app" {
		t.Fatalf("first row parsed incorrectly: %#v", got[0])
	}
	wantOpened := time.Unix(1745999192, 0).UTC()
	if got[0].LastOpenedAt == nil || !got[0].LastOpenedAt.Equal(wantOpened) {
		t.Fatalf("LastOpenedAt = %v, want %v", got[0].LastOpenedAt, wantOpened)
	}
	if got[1].LastOpenedAt != nil {
		t.Fatalf("second LastOpenedAt = %v, want nil", got[1].LastOpenedAt)
	}
}
