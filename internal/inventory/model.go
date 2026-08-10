package inventory

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/listing"
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
	Signature        *SoftwareCodeSignature
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

// SoftwareVersionList is the observed versions of one software title.
type SoftwareVersionList struct {
	Items []SoftwareVersion `json:"items" nullable:"false"`
	Count int32             `json:"count"`
}

// SoftwareSigningIdentity is one code-signing identity observed for a software title.
type SoftwareSigningIdentity struct {
	Identifier        string `json:"identifier"`
	SigningIdentifier string `json:"signing_identifier"`
	TeamIdentifier    string `json:"team_identifier"`
	DeveloperName     string `json:"developer_name"`
	Authority         string `json:"authority"`
	HostsCount        int32  `json:"hosts_count"`
}

// SoftwareSigningIdentityList is the signing identities observed for one software title.
type SoftwareSigningIdentityList struct {
	Items []SoftwareSigningIdentity `json:"items" nullable:"false"`
	Count int32                     `json:"count"`
}

// SoftwareCodeSignature is one code-signing result observed for an installed path.
type SoftwareCodeSignature struct {
	Valid          bool   `json:"valid"`
	Identifier     string `json:"identifier"`
	Authority      string `json:"authority"`
	TeamIdentifier string `json:"team_identifier"`
	CDHash         string `json:"cdhash"`
}

// SoftwareInstalledPath is one observed installation path and its path-owned metadata.
type SoftwareInstalledPath struct {
	Path             string                 `json:"path"`
	ExecutableSHA256 string                 `json:"executable_sha256"`
	ExecutablePath   string                 `json:"executable_path"`
	Signature        *SoftwareCodeSignature `json:"signature,omitempty"`
}

// HostSoftwareInstalledVersion is a host's installed software version and paths.
type HostSoftwareInstalledVersion struct {
	Version          string                  `json:"version"`
	BundleIdentifier string                  `json:"bundle_identifier"`
	Paths            []SoftwareInstalledPath `json:"paths" nullable:"false"`
	LastOpenedAt     *time.Time              `json:"last_opened_at,omitempty"`
}

// HostSoftware is software inventory projected for one host.
type HostSoftware struct {
	ID                int64                          `json:"id"`
	Name              string                         `json:"name"`
	Source            string                         `json:"source"`
	ExtensionFor      string                         `json:"extension_for"`
	InstalledVersions []HostSoftwareInstalledVersion `json:"installed_versions"`
}

// SoftwareTitle is an aggregate software title row.
type SoftwareTitle struct {
	ID                int64                       `db:"id"                json:"id"`
	Name              string                      `db:"name"              json:"name"`
	Source            string                      `db:"source"            json:"source"`
	ExtensionFor      string                      `db:"extension_for"     json:"extension_for"`
	Browser           string                      `db:"-"                 json:"browser"`
	BundleIdentifier  string                      `db:"bundle_identifier" json:"bundle_identifier,omitempty"`
	Vendor            string                      `db:"vendor"            json:"-"`
	HostsCount        int32                       `db:"hosts_count"       json:"hosts_count"`
	VersionsCount     int32                       `db:"versions_count"    json:"-"`
	Versions          SoftwareVersionList         `db:"-"                 json:"versions"`
	SigningIdentities SoftwareSigningIdentityList `db:"-" json:"signing_identities"`
}

// SoftwareTitleListParams controls software title list filtering and sorting.
type SoftwareTitleListParams struct {
	ListParams listing.Params

	SoftwareSources []string
}

// HostSoftwareListParams controls software installed on one host.
type HostSoftwareListParams struct {
	ListParams listing.Params

	SoftwareSources []string
}
