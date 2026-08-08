package software

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/text/unicode/norm"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
)

// InstallationDetectorVersionSource selects the application bundle version
// property used to compare installed software with a Munki package version.
type InstallationDetectorVersionSource string

const (
	InstallationVersionSourceBundleShortVersion InstallationDetectorVersionSource = "bundle_short_version"
	InstallationVersionSourceBundleVersion      InstallationDetectorVersionSource = "bundle_version"
)

var installationDetectorVersionSourceValues = []InstallationDetectorVersionSource{
	InstallationVersionSourceBundleShortVersion,
	InstallationVersionSourceBundleVersion,
}

func (InstallationDetectorVersionSource) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(installationDetectorVersionSourceValues...)
}

// InstallationDetector identifies an installed application and the version
// property Woodstar should compare with the selected Munki package.
type InstallationDetector struct {
	BundleIdentifier string                            `json:"bundle_identifier" validate:"required,notblank" minLength:"1"`
	ExpectedPath     string                            `json:"expected_path,omitempty"`
	VersionSource    InstallationDetectorVersionSource `json:"version_source" validate:"required,oneof=bundle_short_version bundle_version"`
}

func (d *InstallationDetector) normalize() {
	if d == nil {
		return
	}
	d.BundleIdentifier = strings.TrimSpace(d.BundleIdentifier)
	d.ExpectedPath = strings.TrimSpace(d.ExpectedPath)
	d.VersionSource = InstallationDetectorVersionSource(strings.TrimSpace(string(d.VersionSource)))
}

// IconURL returns the icon resource URL when software has an attached icon.
func IconURL(iconObjectID *int64) string {
	if iconObjectID == nil || *iconObjectID <= 0 {
		return ""
	}
	return "/api/munki/icons/" + strconv.FormatInt(*iconObjectID, 10) + "/content"
}

// CreateMutation is the input shape for creating Munki software.
type CreateMutation struct {
	Name                 string                `json:"name"                     minLength:"1" validate:"required,notblank"`
	DisplayName          string                `json:"display_name,omitempty"`
	Description          string                `json:"description,omitempty"`
	Category             string                `json:"category,omitempty"`
	Developer            string                `json:"developer,omitempty"`
	IconObjectID         *int64                `json:"icon_object_id,omitempty"               validate:"omitempty,gt=0"    minimum:"1"`
	Targets              Targets               `json:"targets"                                                                         nullable:"false"`
	InstallationDetector *InstallationDetector `json:"installation_detector,omitempty"`
}

func (m *CreateMutation) validate() error {
	if err := validation.Struct(m); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	if err := validateName(m.Name); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func (m *CreateMutation) normalize() {
	m.Name = norm.NFC.String(strings.TrimSpace(m.Name))
	m.DisplayName = strings.TrimSpace(m.DisplayName)
	if m.DisplayName == m.Name {
		m.DisplayName = ""
	}
	m.Description = strings.TrimSpace(m.Description)
	m.Category = strings.TrimSpace(m.Category)
	m.Developer = strings.TrimSpace(m.Developer)
	m.Targets = normalizeTargets(m.Targets)
	m.InstallationDetector.normalize()
}

// UpdateMutation is the input shape for updating mutable Munki software metadata.
type UpdateMutation struct {
	DisplayName          string                `json:"display_name,omitempty"`
	Description          string                `json:"description,omitempty"`
	Category             string                `json:"category,omitempty"`
	Developer            string                `json:"developer,omitempty"`
	IconObjectID         *int64                `json:"icon_object_id,omitempty" validate:"omitempty,gt=0" minimum:"1"`
	Targets              Targets               `json:"targets"                                                        nullable:"false"`
	InstallationDetector *InstallationDetector `json:"installation_detector,omitempty"`
}

func (m *UpdateMutation) validate() error {
	if err := validation.Struct(m); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func (m *UpdateMutation) normalize(name string) {
	m.DisplayName = strings.TrimSpace(m.DisplayName)
	if m.DisplayName == name {
		m.DisplayName = ""
	}
	m.Description = strings.TrimSpace(m.Description)
	m.Category = strings.TrimSpace(m.Category)
	m.Developer = strings.TrimSpace(m.Developer)
	m.Targets = normalizeTargets(m.Targets)
	m.InstallationDetector.normalize()
}

func validateName(name string) error {
	if strings.Contains(name, "/") {
		return errors.New("name must not contain a slash")
	}
	for _, delimiter := range []string{"--", "-"} {
		parts := strings.Split(name, delimiter)
		last := parts[len(parts)-1]
		first, _ := utf8.DecodeRuneInString(last)
		if len(parts) > 1 && first >= '0' && first <= '9' {
			return errors.New("name must not end with a Munki version suffix")
		}
	}
	return nil
}

// Software is Woodstar-managed metadata for a Munki software item.
type Software struct {
	ID                   int64                 `json:"id"`
	Name                 string                `json:"name"`
	DisplayName          *string               `json:"display_name,omitempty"`
	Description          string                `json:"description"`
	Category             string                `json:"category"`
	Developer            string                `json:"developer"`
	IconObjectID         *int64                `json:"icon_object_id,omitempty"`
	IconFile             *IconFile             `json:"icon_file,omitempty"`
	IconURL              string                `json:"icon_url,omitempty"`
	InstallationDetector *InstallationDetector `json:"installation_detector,omitempty"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`

	InstallationDetectorAutomatic bool `json:"-"`
}

// IconFile describes the object attached as a software icon.
type IconFile struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}
