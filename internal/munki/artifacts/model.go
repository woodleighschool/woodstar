package artifacts

import (
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

	// ArtifactKindIcon is an icon referenced by Munki catalogs.
	ArtifactKindIcon ArtifactKind = "icon"
)

var kindValues = []ArtifactKind{
	ArtifactKindPackage,
	ArtifactKindIcon,
}

func (ArtifactKind) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(kindValues...)
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

// ArtifactUploadURL is a temporary object-storage upload target.
type ArtifactUploadURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ArtifactObject is the storage-side view of an uploaded artifact.
type ArtifactObject struct {
	ContentType string
	SizeBytes   int64
	SHA256      string
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

func (m ArtifactMutation) Validate() error {
	if !ValidArtifactKind(m.Kind) {
		return fmt.Errorf("%w: unsupported artifact kind %q", dbutil.ErrInvalidInput, m.Kind)
	}
	if !ValidArtifactLocation(m.Location) {
		return fmt.Errorf("%w: location is required and must be a relative Munki path", dbutil.ErrInvalidInput)
	}
	if !validSHA256(m.SHA256) {
		return fmt.Errorf("%w: sha256 must be 64 lowercase hex characters", dbutil.ErrInvalidInput)
	}
	if strings.TrimSpace(m.StorageKey) == "" || strings.HasPrefix(strings.TrimSpace(m.StorageKey), "/") {
		return fmt.Errorf("%w: storage_key is required and must be relative", dbutil.ErrInvalidInput)
	}
	return nil
}

func ValidArtifactKind(kind ArtifactKind) bool {
	return slices.Contains(kindValues, kind)
}

func ValidArtifactLocation(location string) bool {
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
