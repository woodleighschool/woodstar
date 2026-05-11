package software

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
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
	dbutil.ListParams

	SoftwareSources []string
}

// HostSoftwareListParams controls software installed on one host.
type HostSoftwareListParams struct {
	dbutil.ListParams

	SoftwareSources []string
}

// Store persists global software titles and host inventory joins.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

// NewStore returns a software store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}
