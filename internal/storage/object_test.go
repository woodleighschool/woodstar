package storage

import "testing"

func TestObjectKey(t *testing.T) {
	t.Parallel()
	obj := Object{ID: 42, Prefix: "munki/packages", Filename: "Firefox-120.0.dmg"}
	if got, want := obj.Key(), "munki/packages/42/Firefox-120.0.dmg"; got != want {
		t.Fatalf("Key() = %q, want %q", got, want)
	}
}

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Firefox 120.dmg":  "Firefox-120.dmg",
		"../../etc/passwd": "passwd",
		`weird*name?.png`:  "weird-name.png",
		"  spaced .icns ":  "spaced.icns",
		"":                 "file",
		"...":              "file",
		"a/b/c.pkg":        "c.pkg",
		"Acmé Café.png":    "Acm-Caf.png",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}
