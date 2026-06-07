package artifacts

import (
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestBuildUploadTargetUsesCleanFilenameAndStorageKey(t *testing.T) {
	sha := strings.Repeat("a", 64)

	target, err := BuildUploadTarget(UploadTargetInput{
		Kind:      ArtifactKindIcon,
		Filename:  `C:\Downloads\GoogleChrome.png`,
		SizeBytes: 123,
		SHA256:    " " + sha + " ",
	})
	if err != nil {
		t.Fatalf("BuildUploadTarget: %v", err)
	}

	if target.Location != "aaaaaaaaaaaa/GoogleChrome.png" {
		t.Fatalf("Location = %q, want SHA-prefixed clean filename", target.Location)
	}
	if target.StorageKey != "icons/aaaaaaaaaaaa/GoogleChrome.png" {
		t.Fatalf("StorageKey = %q, want icon storage key", target.StorageKey)
	}
	if target.DisplayName != "GoogleChrome.png" {
		t.Fatalf("DisplayName = %q, want clean filename", target.DisplayName)
	}
	if target.ContentType != "image/png" {
		t.Fatalf("ContentType = %q, want image/png", target.ContentType)
	}
	if target.SHA256 != sha {
		t.Fatalf("SHA256 = %q, want trimmed SHA", target.SHA256)
	}
}

func TestBuildUploadTargetUsesPackagePrefixAndProvidedMetadata(t *testing.T) {
	sha := strings.Repeat("b", 64)

	target, err := BuildUploadTarget(UploadTargetInput{
		Kind:        ArtifactKindPackage,
		Filename:    "../Install.pkg",
		DisplayName: "Chrome Installer",
		ContentType: " application/x-xar ",
		SizeBytes:   456,
		SHA256:      sha,
	})
	if err != nil {
		t.Fatalf("BuildUploadTarget: %v", err)
	}

	if target.Location != "bbbbbbbbbbbb/Install.pkg" {
		t.Fatalf("Location = %q, want SHA-prefixed package filename", target.Location)
	}
	if target.StorageKey != "pkgs/bbbbbbbbbbbb/Install.pkg" {
		t.Fatalf("StorageKey = %q, want package storage key", target.StorageKey)
	}
	if target.DisplayName != "Chrome Installer" {
		t.Fatalf("DisplayName = %q, want provided display name", target.DisplayName)
	}
	if target.ContentType != "application/x-xar" {
		t.Fatalf("ContentType = %q, want trimmed provided type", target.ContentType)
	}
}

func TestBuildUploadTargetDefaultsUnknownContentType(t *testing.T) {
	target, err := BuildUploadTarget(UploadTargetInput{
		Kind:      ArtifactKindPackage,
		Filename:  "installer.unknownext",
		SizeBytes: 1,
		SHA256:    strings.Repeat("c", 64),
	})
	if err != nil {
		t.Fatalf("BuildUploadTarget: %v", err)
	}

	if target.ContentType != "application/octet-stream" {
		t.Fatalf("ContentType = %q, want octet-stream fallback", target.ContentType)
	}
}

func TestBuildUploadTargetRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name  string
		input UploadTargetInput
	}{
		{
			name: "missing filename",
			input: UploadTargetInput{
				Kind:      ArtifactKindIcon,
				Filename:  " / ",
				SizeBytes: 1,
				SHA256:    strings.Repeat("d", 64),
			},
		},
		{
			name: "bad SHA",
			input: UploadTargetInput{
				Kind:      ArtifactKindIcon,
				Filename:  "GoogleChrome.png",
				SizeBytes: 1,
				SHA256:    "not-a-sha",
			},
		},
		{
			name: "unsupported kind",
			input: UploadTargetInput{
				Kind:      ArtifactKind("script"),
				Filename:  "payload.sh",
				SizeBytes: 1,
				SHA256:    strings.Repeat("e", 64),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildUploadTarget(tc.input)
			if !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("BuildUploadTarget error = %v, want invalid input", err)
			}
		})
	}
}
