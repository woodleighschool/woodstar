package software

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/db"
	"github.com/woodleighschool/woodstar/internal/db/sqlc"
	"github.com/woodleighschool/woodstar/internal/store"
)

// HostSoftwareEntry is one installed software version reported by a host.
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
	ID               int64
	Version          string
	BundleIdentifier string
	HostsCount       int
}

// PathSignatureInformation is code-signing and executable hash data for an installed path.
type PathSignatureInformation struct {
	InstalledPath    string
	TeamIdentifier   string
	CDHashSHA256     string
	ExecutableSHA256 string
	ExecutablePath   string
}

// HostSoftwareInstalledVersion is a host's installed software version and paths.
type HostSoftwareInstalledVersion struct {
	Version              string
	BundleIdentifier     string
	InstalledPaths       []string
	SignatureInformation []PathSignatureInformation
	LastOpenedAt         *time.Time
}

// HostSoftwareRow is software inventory projected for one host.
type HostSoftwareRow struct {
	ID                int64
	Name              string
	DisplayName       string
	Source            string
	ExtensionFor      string
	InstalledVersions []HostSoftwareInstalledVersion
}

// SoftwareTitle is an aggregate software title row.
type SoftwareTitle struct {
	ID               int64
	Name             string
	DisplayName      string
	Source           string
	ExtensionFor     string
	BundleIdentifier string
	Vendor           string
	HostsCount       int
	VersionsCount    int
	CountsUpdatedAt  *time.Time
	Versions         []SoftwareVersion
}

// SoftwareTitleListParams controls software title list filtering and sorting.
type SoftwareTitleListParams struct {
	store.ListParams

	SoftwareSources []string
}

// HostSoftwareListParams controls software installed on one host.
type HostSoftwareListParams struct {
	store.ListParams

	SoftwareSources []string
}

// SoftwareStore persists global software titles and host inventory joins.
type SoftwareStore struct {
	db *db.DB
	q  *sqlc.Queries
}

// NewSoftwareStore returns a software store backed by db.
func NewSoftwareStore(db *db.DB) *SoftwareStore {
	return &SoftwareStore{db: db, q: db.Queries()}
}
