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

func TestValidateFilenameAccepts(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"Firefox-120.0.dmg",
		"Acmé Café.png",
		"name with spaces.pkg",
		"weird+name(1).png",
	} {
		if err := validateFilename(name); err != nil {
			t.Errorf("validateFilename(%q) = %v, want nil", name, err)
		}
	}
}

func TestValidateFilenameRejects(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"",
		"   ",
		" leading.dmg",
		"trailing.dmg ",
		".",
		"..",
		"a/b/c.pkg",
		`a\b.pkg`,
		"../../etc/passwd",
		"with\x00null.pkg",
	} {
		if err := validateFilename(name); !errors.Is(err, dbutil.ErrInvalidInput) {
			t.Errorf("validateFilename(%q) = %v, want ErrInvalidInput", name, err)
		}
	}
}
