// Package clientresources owns Munki's singleton Managed Software Center branding archive.
package clientresources

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gabriel-vasile/mimetype"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/openapischema"
)

const (
	BannerObjectPrefix  = "munki/clientresources/banners"
	ArchiveObjectPrefix = "munki/clientresources/archives"
	MaxBannerSizeBytes  = 5 * 1024 * 1024
	maxLinks            = 12
	maxLinkLabelLength  = 80
	maxLinkTargetLength = 2048
	maxFooterTextLength = 500
)

// BannerAlignment controls which part of a fixed-height banner remains anchored as the window resizes.
type BannerAlignment string

const (
	BannerAlignmentLeft   BannerAlignment = "left"
	BannerAlignmentCenter BannerAlignment = "center"
)

var bannerAlignmentValues = []BannerAlignment{
	BannerAlignmentLeft,
	BannerAlignmentCenter,
}

// Schema returns the OpenAPI schema for BannerAlignment.
func (BannerAlignment) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(bannerAlignmentValues...)
}

// Link is one generated Managed Software Center navigation link.
type Link struct {
	Label         string `json:"label"           maxLength:"80"`
	Target        string `json:"target"          maxLength:"2048"`
	OpenInBrowser bool   `json:"open_in_browser"`
}

// Mutation is the complete builder state accepted by Save.
type Mutation struct {
	BannerObjectID  int64           `json:"banner_object_id"`
	BannerAlignment BannerAlignment `json:"banner_alignment"`
	Links           []Link          `json:"links"            maxItems:"12"`
	FooterText      string          `json:"footer_text"                    maxLength:"500"`
	FooterLinks     []Link          `json:"footer_links"     maxItems:"12"`
}

// ClientResources is the configured singleton and its stored source and archive objects.
type ClientResources struct {
	Mutation

	ArchiveObjectID int64     `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type storedMutation struct {
	Mutation

	ArchiveObjectID int64
}

func (m *Mutation) normalize() {
	m.FooterText = strings.TrimSpace(m.FooterText)
	for i := range m.Links {
		m.Links[i].normalize()
	}
	for i := range m.FooterLinks {
		m.FooterLinks[i].normalize()
	}
	if m.Links == nil {
		m.Links = []Link{}
	}
	if m.FooterLinks == nil {
		m.FooterLinks = []Link{}
	}
}

func (m *Mutation) validate() error {
	if m.BannerObjectID <= 0 {
		return fmt.Errorf("%w: banner_object_id is required", dbutil.ErrInvalidInput)
	}
	if m.BannerAlignment != BannerAlignmentLeft && m.BannerAlignment != BannerAlignmentCenter {
		return fmt.Errorf("%w: banner_alignment must be left or center", dbutil.ErrInvalidInput)
	}
	if utf8.RuneCountInString(m.FooterText) > maxFooterTextLength {
		return fmt.Errorf("%w: footer_text is too long", dbutil.ErrInvalidInput)
	}
	if err := validateLinks("links", m.Links); err != nil {
		return err
	}
	return validateLinks("footer_links", m.FooterLinks)
}

func (l *Link) normalize() {
	l.Label = strings.TrimSpace(l.Label)
	l.Target = strings.TrimSpace(l.Target)
}

func validateLinks(field string, links []Link) error {
	if len(links) > maxLinks {
		return fmt.Errorf("%w: %s cannot contain more than %d links", dbutil.ErrInvalidInput, field, maxLinks)
	}
	labels := make(map[string]struct{}, len(links))
	for i, link := range links {
		if link.Label == "" {
			return fmt.Errorf("%w: %s[%d].label is required", dbutil.ErrInvalidInput, field, i)
		}
		if utf8.RuneCountInString(link.Label) > maxLinkLabelLength {
			return fmt.Errorf("%w: %s[%d].label is too long", dbutil.ErrInvalidInput, field, i)
		}
		labelKey := strings.ToLower(link.Label)
		if _, exists := labels[labelKey]; exists {
			return fmt.Errorf("%w: %s contains duplicate label %q", dbutil.ErrInvalidInput, field, link.Label)
		}
		labels[labelKey] = struct{}{}
		if err := validateLinkTarget(link); err != nil {
			return fmt.Errorf("%w: %s[%d].target %w", dbutil.ErrInvalidInput, field, i, err)
		}
	}
	return nil
}

func validateLinkTarget(link Link) error {
	if link.Target == "" {
		return errors.New("is required")
	}
	if utf8.RuneCountInString(link.Target) > maxLinkTargetLength {
		return errors.New("is too long")
	}
	target, err := url.ParseRequestURI(link.Target)
	if err != nil || target.Scheme == "" {
		return errors.New("must be an absolute URL or Munki route")
	}
	switch strings.ToLower(target.Scheme) {
	case "http", "https":
		if target.Host == "" || target.User != nil {
			return errors.New("must be a valid HTTP URL without credentials")
		}
	case "mailto", "munki":
		if link.OpenInBrowser {
			return errors.New("can only open HTTP URLs in the browser")
		}
	default:
		return fmt.Errorf("uses unsupported scheme %q", target.Scheme)
	}
	return nil
}

func bannerExtension(contentType string) (string, bool) {
	detected := lookupContentType(contentType)
	if detected == nil {
		return "", false
	}
	switch {
	case detected.Is("image/jpeg"):
		return "jpg", true
	case detected.Is("image/png"):
		return "png", true
	default:
		return "", false
	}
}

func validateBanner(contentType string, sizeBytes int64) error {
	if _, ok := bannerExtension(contentType); !ok {
		return fmt.Errorf("%w: banner must be a JPEG or PNG image", dbutil.ErrInvalidInput)
	}
	if sizeBytes <= 0 || sizeBytes > MaxBannerSizeBytes {
		return fmt.Errorf("%w: banner must be between 1 byte and 5 MiB", dbutil.ErrInvalidInput)
	}
	return nil
}

func lookupContentType(contentType string) *mimetype.MIME {
	return mimetype.Lookup(contentType)
}
