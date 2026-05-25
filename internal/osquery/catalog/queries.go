package catalog

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/scope"
)

const detailQueryCadence = time.Hour

const (
	QueryOSVersion                    = "os_version"
	QuerySystemInfo                   = "system_info"
	QueryOsqueryInfo                  = "osquery_info"
	QueryOsqueryFlags                 = "osquery_flags"
	QueryOrbitInfo                    = "orbit_info"
	QueryUptime                       = "uptime"
	QueryRootDiskDarwin               = "root_disk_darwin"
	QueryPrimaryInterfaceUnix         = "primary_interface_unix"
	QueryPrimaryInterfaceWindows      = "primary_interface_windows"
	QueryUsers                        = "users"
	QueryBatteries                    = "batteries"
	QueryCertificatesDarwin           = "certificates_darwin"
	QueryCertificatesWindows          = "certificates_windows"
	QuerySoftwareMacOS                = "software_macos"
	QuerySoftwareLinux                = "software_linux"
	QuerySoftwareWindows              = "software_windows"
	QuerySoftwareVSCodeExtensions     = "software_vscode_extensions"
	QuerySoftwareJetBrainsPlugins     = "software_jetbrains_plugins"
	QuerySoftwareGoBinaries           = "software_go_binaries"
	QuerySoftwarePythonPackages       = "software_python_packages"
	QuerySoftwarePythonPackagesLegacy = "software_python_packages_legacy"
	QuerySoftwareMacOSCodesign        = "software_macos_codesign"
	QuerySoftwareMacOSExecutableHash  = "software_macos_executable_sha256"
)

//go:embed queries/*.sql
var queryFS embed.FS

var tableDiscoverySQL = map[string]string{
	"battery":           tableDiscovery("battery"),
	"certificates":      tableDiscovery("certificates"),
	"codesign":          tableDiscovery("codesign"),
	"disk_space":        tableDiscovery("disk_space"),
	"executable_hashes": tableDiscovery("executable_hashes"),
	"go_binaries":       tableDiscovery("go_binaries"),
	"jetbrains_plugins": tableDiscovery("jetbrains_plugins"),
	"orbit_info":        tableDiscovery("orbit_info"),
	"vscode_extensions": tableDiscovery("vscode_extensions"),
}

var (
	detailRegistry     = buildDetailQueries()
	detailRegistryHash = hashDetailQueries(detailRegistry)
)

// DetailQuery is one built-in detail query.
type DetailQuery struct {
	SQL       string
	Discovery string
	Optional  bool
	Platforms []scope.Platform
	Ingest    DetailIngest
}

type DetailIngest string

const (
	IngestHostDetail         DetailIngest = "host_detail"
	IngestOsqueryFlags       DetailIngest = "osquery_flags"
	IngestUptime             DetailIngest = "uptime"
	IngestUsers              DetailIngest = "users"
	IngestBatteries          DetailIngest = "batteries"
	IngestCertificates       DetailIngest = "certificates"
	IngestSoftwareBase       DetailIngest = "software_base"
	IngestSoftwareEnrichment DetailIngest = "software_enrichment"
)

func (q DetailQuery) RunsForPlatform(platform scope.Platform) bool {
	if len(q.Platforms) == 0 {
		return true
	}
	if platform == "" || platform == scope.PlatformUnknown {
		return false
	}
	return slices.Contains(q.Platforms, platform)
}

func (q DetailQuery) Deferred() bool {
	return q.Ingest == IngestSoftwareBase || q.Ingest == IngestSoftwareEnrichment
}

// DueDetailQueries is due osquery work.
type DueDetailQueries struct {
	Queries   map[string]string
	Discovery map[string]string
}

// DetailQueries returns the built-in detail queries.
func DetailQueries() map[string]DetailQuery {
	return detailRegistry
}

func buildDetailQueries() map[string]DetailQuery {
	registry := map[string]DetailQuery{
		QueryOSVersion: {
			SQL:    mustQuery("queries/os_version.sql"),
			Ingest: IngestHostDetail,
		},
		QuerySystemInfo: {
			SQL:    mustQuery("queries/system_info.sql"),
			Ingest: IngestHostDetail,
		},
		QueryOsqueryInfo: {
			SQL:    mustQuery("queries/osquery_info.sql"),
			Ingest: IngestHostDetail,
		},
		QueryOsqueryFlags: {
			SQL:       mustQuery("queries/osquery_flags.sql"),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformLinux, scope.PlatformWindows},
			Ingest:    IngestOsqueryFlags,
		},
		QueryOrbitInfo: {
			SQL:       mustQuery("queries/orbit_info.sql"),
			Discovery: tableExistsSQL("orbit_info"),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformLinux, scope.PlatformWindows},
			Ingest:    IngestHostDetail,
		},
		QueryUptime: {
			SQL:       mustQuery("queries/uptime.sql"),
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformLinux, scope.PlatformWindows},
			Ingest:    IngestUptime,
		},
		QueryRootDiskDarwin: {
			SQL:       mustQuery("queries/root_disk_darwin.sql"),
			Discovery: tableExistsSQL("disk_space"),
			Platforms: []scope.Platform{scope.PlatformDarwin},
			Ingest:    IngestHostDetail,
		},
		QueryPrimaryInterfaceUnix: {
			SQL:       mustQuery("queries/primary_interface_unix.sql"),
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformLinux},
			Ingest:    IngestHostDetail,
		},
		QueryPrimaryInterfaceWindows: {
			SQL:       mustQuery("queries/primary_interface_windows.sql"),
			Platforms: []scope.Platform{scope.PlatformWindows},
			Ingest:    IngestHostDetail,
		},
		QueryUsers: {
			SQL:       mustQuery("queries/users.sql"),
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformLinux, scope.PlatformWindows},
			Ingest:    IngestUsers,
		},
		QueryBatteries: {
			SQL:       mustQuery("queries/batteries.sql"),
			Discovery: tableExistsSQL("battery"),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformWindows},
			Ingest:    IngestBatteries,
		},
		QueryCertificatesDarwin: {
			SQL:       mustQuery("queries/certificates_darwin.sql"),
			Discovery: tableExistsSQL("certificates"),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin},
			Ingest:    IngestCertificates,
		},
		QueryCertificatesWindows: {
			SQL:       mustQuery("queries/certificates_windows.sql"),
			Discovery: tableExistsSQL("certificates"),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformWindows},
			Ingest:    IngestCertificates,
		},
	}
	maps.Copy(registry, softwareDetailQueries())
	return registry
}

func softwareDetailQueries() map[string]DetailQuery {
	return map[string]DetailQuery{
		QuerySoftwareMacOS: {
			SQL:       mustQuery("queries/software_macos.sql"),
			Platforms: []scope.Platform{scope.PlatformDarwin},
			Ingest:    IngestSoftwareBase,
		},
		QuerySoftwareLinux: {
			SQL:       mustQuery("queries/software_linux.sql"),
			Platforms: []scope.Platform{scope.PlatformLinux},
			Ingest:    IngestSoftwareBase,
		},
		QuerySoftwareWindows: {
			SQL:       mustQuery("queries/software_windows.sql"),
			Platforms: []scope.Platform{scope.PlatformWindows},
			Ingest:    IngestSoftwareBase,
		},
		QuerySoftwareVSCodeExtensions: {
			SQL:       mustQuery("queries/software_vscode_extensions.sql"),
			Discovery: tableExistsSQL("vscode_extensions"),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformLinux, scope.PlatformWindows},
			Ingest:    IngestSoftwareEnrichment,
		},
		QuerySoftwareJetBrainsPlugins: {
			SQL:       mustQuery("queries/software_jetbrains_plugins.sql"),
			Discovery: tableExistsSQL("jetbrains_plugins"),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformLinux, scope.PlatformWindows},
			Ingest:    IngestSoftwareEnrichment,
		},
		QuerySoftwareGoBinaries: {
			SQL:       mustQuery("queries/software_go_binaries.sql"),
			Discovery: tableExistsSQL("go_binaries"),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformLinux, scope.PlatformWindows},
			Ingest:    IngestSoftwareEnrichment,
		},
		QuerySoftwarePythonPackages: {
			SQL:       mustQuery("queries/software_python_packages.sql"),
			Discovery: osqueryVersionAtLeastSQL(5, 16),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformLinux, scope.PlatformWindows},
			Ingest:    IngestSoftwareEnrichment,
		},
		QuerySoftwarePythonPackagesLegacy: {
			SQL:       mustQuery("queries/software_python_packages_legacy.sql"),
			Discovery: osqueryVersionBeforeSQL(5, 16),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin, scope.PlatformLinux, scope.PlatformWindows},
			Ingest:    IngestSoftwareEnrichment,
		},
		QuerySoftwareMacOSCodesign: {
			SQL:       mustQuery("queries/software_macos_codesign.sql"),
			Discovery: tableExistsSQL("codesign"),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin},
			Ingest:    IngestSoftwareEnrichment,
		},
		QuerySoftwareMacOSExecutableHash: {
			SQL:       mustQuery("queries/software_macos_executable_sha256.sql"),
			Discovery: tableExistsSQL("executable_hashes"),
			Optional:  true,
			Platforms: []scope.Platform{scope.PlatformDarwin},
			Ingest:    IngestSoftwareEnrichment,
		},
	}
}

func DetailQueriesDue(lastUpdated *time.Time, lastHash string, platform scope.Platform) DueDetailQueries {
	if lastUpdated != nil && lastHash == detailRegistryHash && time.Since(*lastUpdated) < detailQueryCadence {
		return DueDetailQueries{}
	}

	querySQL := make(map[string]string, len(detailRegistry))
	discovery := make(map[string]string)
	for name, query := range detailRegistry {
		if !query.RunsForPlatform(platform) {
			continue
		}
		querySQL[name] = query.SQL
		if query.Discovery != "" {
			discovery[name] = query.Discovery
		}
	}
	return DueDetailQueries{Queries: querySQL, Discovery: discovery}
}

func DetailQueryHash() string {
	return detailRegistryHash
}

func hashDetailQueries(registry map[string]DetailQuery) string {
	names := make([]string, 0, len(registry))
	for name, query := range registry {
		if !query.Optional {
			names = append(names, name+"\x00"+query.SQL)
		}
	}
	sort.Strings(names)
	sum := sha256.Sum256([]byte(strings.Join(names, "\x00")))
	return hex.EncodeToString(sum[:])
}

func tableExistsSQL(name string) string {
	sql, ok := tableDiscoverySQL[name]
	if !ok {
		panic("unknown osquery discovery table: " + name)
	}
	return sql
}

func tableDiscovery(name string) string {
	return fmt.Sprintf(
		"SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = '%s'",
		name,
	)
}

func osqueryVersionAtLeastSQL(major int, minor int) string {
	return osqueryVersionCompareSQL(fmt.Sprintf("> %d OR (major = %d AND minor >= %d)", major, major, minor))
}

func osqueryVersionBeforeSQL(major int, minor int) string {
	return osqueryVersionCompareSQL(fmt.Sprintf("< %d OR (major = %d AND minor < %d)", major, major, minor))
}

func osqueryVersionCompareSQL(condition string) string {
	return `SELECT 1 FROM (
  SELECT
    CAST(substr(version, 1, instr(version, '.') - 1) AS INTEGER) AS major,
    CAST(substr(version, instr(version, '.') + 1, instr(substr(version, instr(version, '.') + 1), '.') - 1) AS INTEGER) AS minor
  FROM osquery_info
) AS version_parts
WHERE major ` + condition
}

func mustQuery(path string) string {
	content, err := queryFS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(content)
}
