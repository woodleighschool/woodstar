package destination

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

func TestCreationReviewPreservesScopeAndUnfamiliarFields(t *testing.T) {
	fields := map[string]json.RawMessage{
		"version": raw("2.0"), "receipts": json.RawMessage(`[{"package_id":"org.example.app","version":"2.0","optional":true}]`),
		"installs":           json.RawMessage(`[{"path":"/Applications/Example.app"}]`),
		"requires":           json.RawMessage(`[{"software_id":3,"package_id":4}]`),
		"postinstall_script": raw("#!/bin/sh\necho ready\n"), "future_field": raw(map[string]any{"enabled": false}), "notes": json.RawMessage(`null`),
	}
	before := string(raw(fields))
	review := strings.Join(creationReview("Package", fields, metadata{referenceNames: map[packages.PackageReferenceMutation]string{{SoftwareID: 3, PackageID: 4}: "Runtime · 1.0"}}), "\n")
	for _, want := range []string{"version: 2.0", "org.example.app · 2.0 (optional)", "/Applications/Example.app", "requires: Runtime · 1.0", "postinstall_script: 2 lines", "future_field: {\"enabled\":false}", "notes: null"} {
		if !strings.Contains(review, want) {
			t.Errorf("missing %q: %s", want, review)
		}
	}
	if string(raw(fields)) != before {
		t.Fatal("review altered detailed evidence")
	}
}

func TestCreationReviewUsesTheDetectionVersion(t *testing.T) {
	for _, tt := range []struct{ name, key, short, build, want string }{
		{"default", "", "2.0", "203", "/Applications/Example.app · 2.0"},
		{"build", "CFBundleVersion", "2.0", "203", "/Applications/Example.app · 203 (version key: CFBundleVersion)"},
		{"short", "CFBundleShortVersionString", "2.0", "203", "/Applications/Example.app · 2.0 (version key: CFBundleShortVersionString)"},
		{"missing selected version", "CFBundleVersion", "2.0", "", "/Applications/Example.app (version key: CFBundleVersion)"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			item := map[string]string{"path": "/Applications/Example.app"}
			if tt.key != "" {
				item["version_comparison_key"] = tt.key
			}
			if tt.short != "" {
				item["bundle_short_version"] = tt.short
			}
			if tt.build != "" {
				item["bundle_version"] = tt.build
			}
			lines := creationReview("Package", map[string]json.RawMessage{"installs": raw([]map[string]string{item})}, metadata{})
			got := strings.TrimSpace(lines[len(lines)-1])
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
