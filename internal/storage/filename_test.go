package storage

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestNormalizeUploadFilename(t *testing.T) {
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
		got := normalizeUploadFilename(in)
		if got != want {
			t.Errorf("normalizeUploadFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateUploadFilenameRejects(t *testing.T) {
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
		name := normalizeUploadFilename(in)
		if err := validateUploadFilename(name); !errors.Is(err, dbutil.ErrInvalidInput) {
			t.Errorf("validateUploadFilename(%q) error = %v, want ErrInvalidInput", name, err)
		}
	}
}
