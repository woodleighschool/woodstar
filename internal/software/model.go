package software

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
	HostsCount       int    `json:"hosts_count"`
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

// HostSoftwareRow is software inventory projected for one host.
type HostSoftwareRow struct {
	ID                int64                          `json:"id"`
	Name              string                         `json:"name"`
	DisplayName       string                         `json:"display_name"`
	Source            string                         `json:"source"`
	ExtensionFor      string                         `json:"extension_for"`
	InstalledVersions []HostSoftwareInstalledVersion `json:"installed_versions"`
}

// SoftwareTitle is an aggregate software title row.
type SoftwareTitle struct {
	ID               int64             `json:"id"`
	Name             string            `json:"name"`
	DisplayName      string            `json:"display_name"`
	Source           string            `json:"source"`
	ExtensionFor     string            `json:"extension_for"`
	Browser          string            `json:"browser"`
	BundleIdentifier string            `json:"bundle_identifier,omitempty"`
	Vendor           string            `json:"-"`
	HostsCount       int               `json:"hosts_count"`
	VersionsCount    int               `json:"versions_count"`
	CountsUpdatedAt  *time.Time        `json:"counts_updated_at"`
	Versions         []SoftwareVersion `json:"versions"`
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
