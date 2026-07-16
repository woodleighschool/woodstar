package clientresources

import (
	"strings"
	"testing"
)

func TestMutationValidateLinks(t *testing.T) {
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
		{name: "credentials", link: Link{Label: "Support", Target: "https://user:pass@example.com"}, wantErr: true},
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
			mutation := Mutation{
				BannerObjectID:  1,
				BannerAlignment: BannerAlignmentLeft,
				Links:           []Link{test.link},
			}
			mutation.normalize()
			err := mutation.validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestMutationValidateBannerAlignment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		alignment BannerAlignment
		wantErr   bool
	}{
		{name: "left", alignment: BannerAlignmentLeft},
		{name: "center", alignment: BannerAlignmentCenter},
		{name: "empty", wantErr: true},
		{name: "right", alignment: "right", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mutation := Mutation{BannerObjectID: 1, BannerAlignment: test.alignment}
			mutation.normalize()
			err := mutation.validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestMutationValidateTextLimitsCountUnicodeCharacters(t *testing.T) {
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
			mutation := Mutation{
				BannerObjectID:  1,
				BannerAlignment: BannerAlignmentLeft,
				Links:           []Link{{Label: test.label, Target: test.target}},
				FooterText:      test.footer,
			}
			mutation.normalize()
			err := mutation.validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
