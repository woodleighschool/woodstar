package storage

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestObjectKey(t *testing.T) {
	t.Parallel()
	obj := Object{ID: 42, Prefix: "munki/packages", Filename: "Firefox-120.0.dmg"}
	if got, want := obj.Key(), "munki/packages/42/Firefox-120.0.dmg"; got != want {
		t.Fatalf("Key() = %q, want %q", got, want)
	}
}

func TestCleanUploadFilename(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Firefox-120.0.dmg":   "Firefox-120.0.dmg",
		"/tmp/Firefox.dmg":    "Firefox.dmg",
		`C:\Users\me\App.pkg`: "App.pkg",
		"  Spaced.icns  ":     "Spaced.icns",
		"Acmé Café.png":       "Acmé Café.png",
		"sub/dir/file.pkg":    "file.pkg",
		"../../etc/passwd":    "passwd",
	}
	for in, want := range cases {
		got, err := cleanUploadFilename(in)
		if err != nil {
			t.Errorf("cleanUploadFilename(%q) error = %v, want nil", in, err)
			continue
		}
		if got != want {
			t.Errorf("cleanUploadFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCleanUploadFilenameRejects(t *testing.T) {
	t.Parallel()
	for _, in := range []string{
		"",
		"   ",
		".",
		"..",
		"/",
		"a/b/..",
		"with\x00null.pkg",
	} {
		if _, err := cleanUploadFilename(in); !errors.Is(err, dbutil.ErrInvalidInput) {
			t.Errorf("cleanUploadFilename(%q) error = %v, want ErrInvalidInput", in, err)
		}
	}
}
