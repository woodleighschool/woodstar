// Package clientresources owns Munki's deployed Managed Software Center resources.
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

// BannerFit controls whether the banner keeps its natural aspect ratio or fills the stage.
type BannerFit string

const (
	BannerFitHeight BannerFit = "height"
	BannerFitCover  BannerFit = "cover"
)

var bannerFitValues = []BannerFit{
	BannerFitHeight,
	BannerFitCover,
}

// Schema returns the OpenAPI schema for BannerFit.
func (BannerFit) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(bannerFitValues...)
}

// Link is one generated Managed Software Center navigation link.
type Link struct {
	Label         string `json:"label"           maxLength:"80"`
	Target        string `json:"target"          maxLength:"2048"`
	OpenInBrowser bool   `json:"open_in_browser"`
}

// Builder is the editable state used to generate a client resources archive.
type Builder struct {
	BannerObjectID int64     `json:"banner_object_id"`
	BannerFit      BannerFit `json:"banner_fit"`
	BannerFocalX   int       `json:"banner_focal_x" minimum:"0" maximum:"100"`
	Links          []Link    `json:"links"                      maxItems:"12"`
	FooterText     string    `json:"footer_text"                              maxLength:"500"`
	FooterLinks    []Link    `json:"footer_links"               maxItems:"12"`
}

// ClientResources is one deployed archive and its optional retained builder state.
type ClientResources struct {
	ID              int64     `json:"id"`
	ArchiveObjectID int64     `json:"-"`
	Custom          bool      `json:"custom"`
	Builder         *Builder  `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ClientResourcesMutation selects either builder-generated or uploaded client resources.
type ClientResourcesMutation struct {
	Builder         *Builder `json:"builder,omitempty"`
	ArchiveObjectID *int64   `json:"archive_object_id,omitempty" minimum:"1"`
}

func (m *ClientResourcesMutation) normalize() {
	if m.Builder != nil {
		m.Builder.normalize()
	}
}

func (m *ClientResourcesMutation) validate() error {
	switch {
	case m.Builder != nil && m.ArchiveObjectID != nil:
		return fmt.Errorf("%w: provide builder or archive_object_id, not both", dbutil.ErrInvalidInput)
	case m.Builder == nil && m.ArchiveObjectID == nil:
		return fmt.Errorf("%w: builder or archive_object_id is required", dbutil.ErrInvalidInput)
	case m.ArchiveObjectID != nil && *m.ArchiveObjectID <= 0:
		return fmt.Errorf("%w: archive_object_id must be positive", dbutil.ErrInvalidInput)
	case m.Builder != nil:
		return m.Builder.validate()
	default:
		return nil
	}
}

func (b *Builder) normalize() {
	b.FooterText = strings.TrimSpace(b.FooterText)
	for i := range b.Links {
		b.Links[i].normalize()
	}
	for i := range b.FooterLinks {
		b.FooterLinks[i].normalize()
	}
	if b.Links == nil {
		b.Links = []Link{}
	}
	if b.FooterLinks == nil {
		b.FooterLinks = []Link{}
	}
}

func (b *Builder) validate() error {
	if b.BannerObjectID <= 0 {
		return fmt.Errorf("%w: banner_object_id is required", dbutil.ErrInvalidInput)
	}
	if b.BannerFit != BannerFitHeight && b.BannerFit != BannerFitCover {
		return fmt.Errorf("%w: banner_fit must be height or cover", dbutil.ErrInvalidInput)
	}
	if b.BannerFocalX < 0 || b.BannerFocalX > 100 {
		return fmt.Errorf("%w: banner_focal_x must be between 0 and 100", dbutil.ErrInvalidInput)
	}
	if utf8.RuneCountInString(b.FooterText) > maxFooterTextLength {
		return fmt.Errorf("%w: footer_text is too long", dbutil.ErrInvalidInput)
	}
	if err := validateLinks("links", b.Links); err != nil {
		return err
	}
	return validateLinks("footer_links", b.FooterLinks)
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
	detected := mimetype.Lookup(contentType)
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
