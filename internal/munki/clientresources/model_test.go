package clientresources

import (
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestClientResourcesMutationRequiresOneSource(t *testing.T) {
	t.Parallel()
	archiveObjectID := int64(7)
	tests := []struct {
		name     string
		mutation ClientResourcesMutation
		wantErr  bool
	}{
		{
			name:     "builder",
			mutation: ClientResourcesMutation{Builder: &Builder{BannerObjectID: 3, BannerFit: BannerFitHeight}},
		},
		{name: "archive", mutation: ClientResourcesMutation{ArchiveObjectID: &archiveObjectID}},
		{name: "missing", mutation: ClientResourcesMutation{}, wantErr: true},
		{
			name: "both",
			mutation: ClientResourcesMutation{
				Builder:         &Builder{BannerObjectID: 3, BannerFit: BannerFitHeight},
				ArchiveObjectID: &archiveObjectID,
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.mutation.normalize()
			err := test.mutation.validate()
			if test.wantErr && !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("validate() error = %v, want ErrInvalidInput", err)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("validate(): %v", err)
			}
		})
	}
}

func TestBuilderValidateLinks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		link    Link
		wantErr bool
	}{
		{name: "Munki route", link: Link{Label: "Updates", Target: "munki://updates"}},
		{name: "external in app", link: Link{Label: "Support", Target: "https://example.com/help"}},
		{
			name: "external in browser",
			link: Link{Label: "Support", Target: "https://example.com/help", OpenInBrowser: true},
		},
		{name: "mailto", link: Link{Label: "Email", Target: "mailto:help@example.com"}},
		{name: "relative", link: Link{Label: "Support", Target: "/help"}, wantErr: true},
		{name: "credentials", link: Link{Label: "Support", Target: "https://user:pass@example.com"}, wantErr: true}, //nolint:gosec // Invalid credential-bearing URL fixture.
		{
			name:    "Munki route in browser",
			link:    Link{Label: "Updates", Target: "munki://updates", OpenInBrowser: true},
			wantErr: true,
		},
		{name: "unsupported scheme", link: Link{Label: "File", Target: "file:///tmp/example"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			builder := Builder{
				BannerObjectID: 1,
				BannerFit:      BannerFitHeight,
				Links:          []Link{test.link},
			}
			builder.normalize()
			err := builder.validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestBuilderValidateBannerLayout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		fit       BannerFit
		focalX    int
		wantError bool
	}{
		{name: "height left", fit: BannerFitHeight},
		{name: "cover center", fit: BannerFitCover, focalX: 50},
		{name: "empty fit", wantError: true},
		{name: "unknown fit", fit: "stretch", wantError: true},
		{name: "focal point below range", fit: BannerFitCover, focalX: -1, wantError: true},
		{name: "focal point above range", fit: BannerFitCover, focalX: 101, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			builder := Builder{BannerObjectID: 1, BannerFit: test.fit, BannerFocalX: test.focalX}
			builder.normalize()
			err := builder.validate()
			if (err != nil) != test.wantError {
				t.Fatalf("validate() error = %v, wantError %v", err, test.wantError)
			}
		})
	}
}

func TestBuilderValidateTextLimitsCountUnicodeCharacters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		label   string
		target  string
		footer  string
		wantErr bool
	}{
		{
			name:   "multibyte text at limits",
			label:  strings.Repeat("界", maxLinkLabelLength),
			target: "https://example.com/" + strings.Repeat("界", maxLinkTargetLength-len("https://example.com/")),
			footer: strings.Repeat("界", maxFooterTextLength),
		},
		{
			name:    "label over limit",
			label:   strings.Repeat("界", maxLinkLabelLength+1),
			target:  "https://example.com",
			wantErr: true,
		},
		{
			name:    "footer over limit",
			label:   "Support",
			target:  "https://example.com",
			footer:  strings.Repeat("界", maxFooterTextLength+1),
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			builder := Builder{
				BannerObjectID: 1,
				BannerFit:      BannerFitHeight,
				Links:          []Link{{Label: test.label, Target: test.target}},
				FooterText:     test.footer,
			}
			builder.normalize()
			err := builder.validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
