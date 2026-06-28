package inventory

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Source values come from osquery table names.
const (
	SourceChromeExtensions = "chrome_extensions"
	SourceFirefoxAddons    = "firefox_addons"
	SourceSafariExtensions = "safari_extensions"
)

// HostSoftwareEntry is ingest-only installed software.
type HostSoftwareEntry struct {
	Name             string
	Version          string
	Source           string
	BundleIdentifier string
	ExtensionID      string
	ExtensionFor     string
	Vendor           string
	Arch             string
	Release          string
	InstalledPath    string
	TeamIdentifier   string
	CDHashSHA256     string
	ExecutableSHA256 string
	ExecutablePath   string
	LastOpenedAt     *time.Time
}

// SoftwareVersion is one observed version under a software title.
type SoftwareVersion struct {
	ID               int64  `json:"id"`
	Version          string `json:"version"`
	BundleIdentifier string `json:"bundle_identifier,omitempty"`
	HostsCount       int32  `json:"hosts_count"`
}

// PathSignatureInformation is signing/hash data for one path.
type PathSignatureInformation struct {
	InstalledPath    string `json:"installed_path"`
	TeamIdentifier   string `json:"team_identifier"`
	CDHashSHA256     string `json:"hash_sha256"`
	ExecutableSHA256 string `json:"executable_sha256"`
	ExecutablePath   string `json:"executable_path"`
}

// HostSoftwareInstalledVersion is a host's installed software version and paths.
type HostSoftwareInstalledVersion struct {
	Version              string                     `json:"version"`
	BundleIdentifier     string                     `json:"bundle_identifier"`
	InstalledPaths       []string                   `json:"installed_paths"`
	SignatureInformation []PathSignatureInformation `json:"signature_information"`
	LastOpenedAt         *time.Time                 `json:"last_opened_at,omitempty"`
}

// HostSoftware is software inventory projected for one host.
type HostSoftware struct {
	ID                int64                          `json:"id"`
	Name              string                         `json:"name"`
	DisplayName       string                         `json:"display_name"`
	Source            string                         `json:"source"`
	ExtensionFor      string                         `json:"extension_for"`
	InstalledVersions []HostSoftwareInstalledVersion `json:"installed_versions"`
}

// SoftwareTitle is an aggregate software title row.
type SoftwareTitle struct {
	ID               int64             `db:"id"                json:"id"`
	Name             string            `db:"name"              json:"name"`
	DisplayName      string            `db:"display_name"      json:"display_name"`
	Source           string            `db:"source"            json:"source"`
	ExtensionFor     string            `db:"extension_for"     json:"extension_for"`
	Browser          string            `db:"-"                 json:"browser"`
	BundleIdentifier string            `db:"bundle_identifier" json:"bundle_identifier,omitempty"`
	Vendor           string            `db:"vendor"            json:"-"`
	HostsCount       int32             `db:"hosts_count"       json:"hosts_count"`
	VersionsCount    int32             `db:"versions_count"    json:"versions_count"`
	CountsUpdatedAt  *time.Time        `db:"counts_updated_at" json:"counts_updated_at"`
	Versions         []SoftwareVersion `db:"-"                 json:"versions"`
}

// SoftwareTitleListParams controls software title list filtering and sorting.
type SoftwareTitleListParams struct {
	dbutil.ListParams

	SoftwareSources []string
}

// HostSoftwareListParams controls software installed on one host.
type HostSoftwareListParams struct {
	dbutil.ListParams

	SoftwareSources []string
}
