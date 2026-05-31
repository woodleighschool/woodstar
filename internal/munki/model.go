package munki

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
)

// ArtifactKind describes how Munki clients consume an artifact.
type ArtifactKind string

const (
	// ArtifactKindPackage is an installer package or disk image.
	ArtifactKindPackage ArtifactKind = "package"

	// ArtifactKindIcon is an icon referenced by rendered pkginfo.
	ArtifactKindIcon ArtifactKind = "icon"
)

var artifactKindValues = []ArtifactKind{
	ArtifactKindPackage,
	ArtifactKindIcon,
}

func (ArtifactKind) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(artifactKindValues...)
}

// AssignmentIntent describes how Munki should enforce or present an assigned release.
type AssignmentIntent string

const (
	// IntentEnsureInstalled puts the release in managed_installs.
	IntentEnsureInstalled AssignmentIntent = "ensure_installed"

	// IntentEnsureAbsent puts the release in managed_uninstalls.
	IntentEnsureAbsent AssignmentIntent = "ensure_absent"

	// IntentUpdateIfPresent puts the release in managed_updates.
	IntentUpdateIfPresent AssignmentIntent = "update_if_present"

	// IntentOptional puts the release in optional_installs.
	IntentOptional AssignmentIntent = "optional"

	// IntentFeatured puts the release in optional_installs and featured_items.
	IntentFeatured AssignmentIntent = "featured"
)

var assignmentIntentValues = []AssignmentIntent{
	IntentEnsureInstalled,
	IntentEnsureAbsent,
	IntentUpdateIfPresent,
	IntentOptional,
	IntentFeatured,
}

func (AssignmentIntent) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(assignmentIntentValues...)
}

// SoftwareTitleMutation is the input shape for creating or updating a Munki software title.
type SoftwareTitleMutation struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Developer   string `json:"developer,omitempty"`
}

// SoftwareTitle is Woodstar-managed metadata for a Munki software item.
type SoftwareTitle struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Developer   string    `json:"developer"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ReleaseMutation is the input shape for a Munki release backed by one pkginfo object.
type ReleaseMutation struct {
	SoftwareID          int64           `json:"software_id"`
	Name                string          `json:"name"`
	Version             string          `json:"version"`
	DisplayName         string          `json:"display_name,omitempty"`
	Pkginfo             json.RawMessage `json:"pkginfo"`
	InstallerArtifactID *int64          `json:"installer_artifact_id,omitempty"`
	Eligible            bool            `json:"eligible"`
}

// Release is one Munki pkginfo version available for assignment.
type Release struct {
	ID                        int64           `json:"id"`
	SoftwareID                int64           `json:"software_id"`
	Name                      string          `json:"name"`
	Version                   string          `json:"version"`
	DisplayName               string          `json:"display_name"`
	Pkginfo                   json.RawMessage `json:"pkginfo"`
	InstallerArtifactID       *int64          `json:"installer_artifact_id,omitempty"`
	InstallerArtifactLocation string          `json:"installer_artifact_location,omitempty"`
	Eligible                  bool            `json:"eligible"`
	CreatedAt                 time.Time       `json:"created_at"`
	UpdatedAt                 time.Time       `json:"updated_at"`
}

// ArtifactMutation is the input shape for registering an existing Munki artifact.
type ArtifactMutation struct {
	Kind        ArtifactKind `json:"kind"`
	DisplayName string       `json:"display_name,omitempty"`
	Location    string       `json:"location"`
	ContentType string       `json:"content_type,omitempty"`
	SizeBytes   int64        `json:"size_bytes"`
	SHA256      string       `json:"sha256"`
	StorageKey  string       `json:"storage_key"`
}

// Artifact references one object stored in Munki's artifact backend.
type Artifact struct {
	ID          int64        `json:"id"`
	Kind        ArtifactKind `json:"kind"`
	DisplayName string       `json:"display_name"`
	Location    string       `json:"location"`
	ContentType string       `json:"content_type"`
	SizeBytes   int64        `json:"size_bytes"`
	SHA256      string       `json:"sha256"`
	StorageKey  string       `json:"storage_key"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// AssignmentMutation is the input shape for assigning a release to a concrete Munki scope.
type AssignmentMutation struct {
	ReleaseID       int64            `json:"release_id"`
	Intent          AssignmentIntent `json:"intent"`
	AllHosts        bool             `json:"all_hosts"`
	IncludeLabelIDs []int64          `json:"include_label_ids,omitempty"`
	ExcludeLabelIDs []int64          `json:"exclude_label_ids,omitempty"`
	IncludeHostIDs  []int64          `json:"include_host_ids,omitempty"`
	ExcludeHostIDs  []int64          `json:"exclude_host_ids,omitempty"`
}

// Assignment links one Munki release, one intent, and concrete include/exclude scope.
type Assignment struct {
	ID              int64            `json:"id"`
	ReleaseID       int64            `json:"release_id"`
	Intent          AssignmentIntent `json:"intent"`
	AllHosts        bool             `json:"all_hosts"`
	IncludeLabelIDs []int64          `json:"include_label_ids"`
	ExcludeLabelIDs []int64          `json:"exclude_label_ids"`
	IncludeHostIDs  []int64          `json:"include_host_ids"`
	ExcludeHostIDs  []int64          `json:"exclude_host_ids"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// EffectiveRelease is a host-resolved Munki release ready for manifest/catalog rendering.
type EffectiveRelease struct {
	AssignmentID int64
	Intent       AssignmentIntent
	Release      Release
	scopeRank    int
}

// HostStatusObservation is Munki state observed for an existing host.
type HostStatusObservation struct {
	HostID          int64
	Version         string
	ManifestName    string
	Success         *bool
	Errors          []string
	Warnings        []string
	ProblemInstalls []string
	RunStartedAt    string
	RunEndedAt      string
}

// HostItem is one Munki-managed item observed on a host.
type HostItem struct {
	HostID           int64     `json:"-"`
	Name             string    `json:"name"`
	Installed        bool      `json:"installed"`
	InstalledVersion string    `json:"installed_version"`
	RunEndedAt       string    `json:"run_ended_at,omitempty"`
	LastSeenAt       time.Time `json:"last_seen_at"`
}

// HostState is the Munki sub-object attached to host detail responses.
type HostState struct {
	Version         string     `json:"version"`
	ManifestName    string     `json:"manifest_name"`
	Success         *bool      `json:"success,omitempty"`
	Errors          []string   `json:"errors"`
	Warnings        []string   `json:"warnings"`
	ProblemInstalls []string   `json:"problem_installs"`
	RunStartedAt    string     `json:"run_started_at,omitempty"`
	RunEndedAt      string     `json:"run_ended_at,omitempty"`
	LastSeenAt      time.Time  `json:"last_seen_at"`
	Items           []HostItem `json:"items"`
}

func (m SoftwareTitleMutation) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	return nil
}

func (m ReleaseMutation) Validate() error {
	if m.SoftwareID <= 0 {
		return fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("%w: version is required", dbutil.ErrInvalidInput)
	}
	if m.InstallerArtifactID != nil && *m.InstallerArtifactID <= 0 {
		return fmt.Errorf("%w: installer_artifact_id must be positive", dbutil.ErrInvalidInput)
	}
	fields, err := parsePkginfoFields(m.Pkginfo)
	if err != nil {
		return err
	}
	if fields.name != strings.TrimSpace(m.Name) {
		return fmt.Errorf("%w: pkginfo name must match release name", dbutil.ErrInvalidInput)
	}
	if fields.version != strings.TrimSpace(m.Version) {
		return fmt.Errorf("%w: pkginfo version must match release version", dbutil.ErrInvalidInput)
	}
	return nil
}

func (m ArtifactMutation) Validate() error {
	if !validArtifactKind(m.Kind) {
		return fmt.Errorf("%w: unsupported artifact kind %q", dbutil.ErrInvalidInput, m.Kind)
	}
	if !validArtifactLocation(m.Location) {
		return fmt.Errorf("%w: location is required and must be a relative Munki path", dbutil.ErrInvalidInput)
	}
	if m.SizeBytes < 0 {
		return fmt.Errorf("%w: size_bytes must not be negative", dbutil.ErrInvalidInput)
	}
	if !validSHA256(m.SHA256) {
		return fmt.Errorf("%w: sha256 must be 64 lowercase hex characters", dbutil.ErrInvalidInput)
	}
	if strings.TrimSpace(m.StorageKey) == "" || strings.HasPrefix(strings.TrimSpace(m.StorageKey), "/") {
		return fmt.Errorf("%w: storage_key is required and must be relative", dbutil.ErrInvalidInput)
	}
	return nil
}

func (m AssignmentMutation) Validate() error {
	if m.ReleaseID <= 0 {
		return fmt.Errorf("%w: release_id is required", dbutil.ErrInvalidInput)
	}
	if !validAssignmentIntent(m.Intent) {
		return fmt.Errorf("%w: unsupported assignment intent %q", dbutil.ErrInvalidInput, m.Intent)
	}
	if !m.AllHosts && len(m.IncludeLabelIDs) == 0 && len(m.IncludeHostIDs) == 0 {
		return fmt.Errorf("%w: assignment scope is required", dbutil.ErrInvalidInput)
	}
	return nil
}

func validAssignmentIntent(intent AssignmentIntent) bool {
	return slices.Contains(assignmentIntentValues, intent)
}

func validArtifactKind(kind ArtifactKind) bool {
	return slices.Contains(artifactKindValues, kind)
}

func validArtifactLocation(location string) bool {
	location = strings.TrimSpace(location)
	if location == "" || strings.HasPrefix(location, "/") || strings.Contains(location, `\`) {
		return false
	}
	for segment := range strings.SplitSeq(location, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

type pkginfoFields struct {
	name    string
	version string
}

func parsePkginfoFields(raw json.RawMessage) (pkginfoFields, error) {
	if !json.Valid(raw) {
		return pkginfoFields{}, fmt.Errorf("%w: pkginfo must be valid JSON", dbutil.ErrInvalidInput)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return pkginfoFields{}, fmt.Errorf("%w: pkginfo must be a JSON object", dbutil.ErrInvalidInput)
	}
	name, ok := object["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return pkginfoFields{}, fmt.Errorf("%w: pkginfo.name is required", dbutil.ErrInvalidInput)
	}
	version, ok := object["version"].(string)
	if !ok || strings.TrimSpace(version) == "" {
		return pkginfoFields{}, fmt.Errorf("%w: pkginfo.version is required", dbutil.ErrInvalidInput)
	}
	return pkginfoFields{name: strings.TrimSpace(name), version: strings.TrimSpace(version)}, nil
}
