package ingest

import (
	"context"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/software"
)

type hostStore interface {
	MarkDetailFresh(context.Context, int64, string) error
	ApplyDetail(context.Context, int64, hosts.DetailUpdate) error
	ReplaceUsers(context.Context, int64, []hosts.HostUser) error
	ReplaceBatteries(context.Context, int64, []hosts.HostBattery) error
	ReplaceCertificates(context.Context, int64, []hosts.HostCertificate) error
}

type softwareStore interface {
	ReplaceHostSoftware(context.Context, int64, []software.HostSoftwareEntry) error
}

type Projector struct {
	hostStore     hostStore
	softwareStore softwareStore
	logger        *slog.Logger
}

func NewProjector(hostStore hostStore, softwareStore softwareStore, logger *slog.Logger) *Projector {
	return &Projector{hostStore: hostStore, softwareStore: softwareStore, logger: logger}
}

func (p *Projector) MarkFresh(ctx context.Context, hostID int64) error {
	return p.hostStore.MarkDetailFresh(ctx, hostID, catalog.DetailQueryHash())
}

// IngestDetail dispatches a single detail query result to the appropriate ingester.
func (p *Projector) IngestDetail(
	ctx context.Context,
	query catalog.DetailQuery,
	name string,
	hostID int64,
	rows []map[string]string,
) error {
	switch query.Ingest {
	case catalog.IngestHostDetail:
		return ingestHostDetail(ctx, p, hostID, name, rows)
	case catalog.IngestOsqueryFlags:
		return ingestOsqueryFlags(ctx, p, hostID, rows)
	case catalog.IngestUptime:
		return ingestUptime(ctx, p, hostID, rows)
	case catalog.IngestUsers:
		return ingestUsers(ctx, p, hostID, rows)
	case catalog.IngestBatteries:
		return ingestBatteries(ctx, p, hostID, rows)
	case catalog.IngestCertificates:
		return ingestCertificates(ctx, p, hostID, name, rows)
	default:
		return nil
	}
}

func (p *Projector) IngestSoftware(
	ctx context.Context,
	hostID int64,
	queryRows map[string][]map[string]string,
) error {
	if p.softwareStore == nil {
		return nil
	}
	enrichment := softwareEnrichmentByPath(
		queryRows[catalog.QuerySoftwareMacOSCodesign],
		queryRows[catalog.QuerySoftwareMacOSExecutableHash],
	)
	rows := softwareRows(queryRows)
	entries := parseSoftwareRows(rows, enrichment)
	if err := p.softwareStore.ReplaceHostSoftware(ctx, hostID, entries); err != nil {
		return err
	}
	if p.logger != nil {
		p.logger.DebugContext(
			ctx,
			"software inventory ingested", "operation", "software_ingest",
			"host_id", hostID,
			"row_count", len(rows),
			"entry_count", len(entries),
			"codesign_count", len(queryRows[catalog.QuerySoftwareMacOSCodesign]),
			"executable_hash_count", len(queryRows[catalog.QuerySoftwareMacOSExecutableHash]),
		)
	}
	return nil
}

func softwareRows(queryRows map[string][]map[string]string) []map[string]string {
	keys := []string{
		catalog.QuerySoftwareMacOS,
		catalog.QuerySoftwareVSCodeExtensions,
		catalog.QuerySoftwareJetBrainsPlugins,
		catalog.QuerySoftwareGoBinaries,
		catalog.QuerySoftwarePythonPackages,
		catalog.QuerySoftwarePythonPackagesLegacy,
	}
	var rows []map[string]string
	for _, key := range keys {
		rows = append(rows, queryRows[key]...)
	}
	return rows
}

func ingestHostDetail(
	ctx context.Context,
	projector *Projector,
	hostID int64,
	name string,
	rows []map[string]string,
) error {
	if len(rows) == 0 {
		return nil
	}
	name = canonicalHostDetailName(name)
	return projector.hostStore.ApplyDetail(
		ctx,
		hostID,
		ParseHostDetails(map[string]map[string]string{name: rows[0]}),
	)
}

func canonicalHostDetailName(name string) string {
	switch name {
	case catalog.QueryRootDiskDarwin:
		return "root_disk"
	case catalog.QueryPrimaryInterfaceUnix:
		return "primary_interface"
	default:
		return name
	}
}

func ingestOsqueryFlags(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	return projector.hostStore.ApplyDetail(ctx, hostID, parseOsqueryFlags(rows))
}

func ingestUptime(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return projector.hostStore.ApplyDetail(
		ctx,
		hostID,
		ParseHostDetails(map[string]map[string]string{catalog.QueryUptime: rows[0]}),
	)
}

func ingestUsers(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	return projector.hostStore.ReplaceUsers(ctx, hostID, parseHostUsers(rows))
}

func ingestBatteries(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	return projector.hostStore.ReplaceBatteries(ctx, hostID, parseHostBatteries(rows))
}

func ingestCertificates(
	ctx context.Context,
	projector *Projector,
	hostID int64,
	name string,
	rows []map[string]string,
) error {
	return projector.hostStore.ReplaceCertificates(ctx, hostID, parseHostCertificates(name, rows))
}

func parseHostUsers(rows []map[string]string) []hosts.HostUser {
	users := make([]hosts.HostUser, 0, len(rows))
	for _, row := range rows {
		username := row["username"]
		uid := row["uid"]
		if username == "" || uid == "" {
			continue
		}
		users = append(users, hosts.HostUser{
			UID:         uid,
			Username:    username,
			Type:        row["type"],
			Description: row["description"],
			Directory:   row["directory"],
			Shell:       row["shell"],
		})
	}
	return users
}

func parseHostBatteries(rows []map[string]string) []hosts.HostBattery {
	batteries := make([]hosts.HostBattery, 0, len(rows))
	for _, row := range rows {
		serialNumber := row["serial_number"]
		if serialNumber == "" {
			continue
		}
		batteries = append(batteries, hosts.HostBattery{
			SerialNumber:     serialNumber,
			Manufacturer:     row["manufacturer"],
			Model:            row["model"],
			Chemistry:        row["chemistry"],
			CycleCount:       parseInt32Ptr(row["cycle_count"]),
			Health:           row["health"],
			DesignedCapacity: parseInt32Ptr(row["designed_capacity"]),
			MaxCapacity:      parseInt32Ptr(row["max_capacity"]),
			CurrentCapacity:  parseInt32Ptr(row["current_capacity"]),
			PercentRemaining: parseFloat64Ptr(row["percent_remaining"]),
		})
	}
	return batteries
}

func parseHostCertificates(queryName string, rows []map[string]string) []hosts.HostCertificate {
	certificates := make([]hosts.HostCertificate, 0, len(rows))
	for _, row := range rows {
		sha1 := row["sha1"]
		if sha1 == "" {
			continue
		}
		subject := parseCertificateName(queryName, row["subject"])
		issuer := parseCertificateName(queryName, row["issuer"])
		commonName := row["common_name"]
		if commonName == "" {
			commonName = subject.CommonName
		}
		certificates = append(certificates, hosts.HostCertificate{
			SHA1:                 sha1,
			CommonName:           commonName,
			Subject:              subject,
			Issuer:               issuer,
			KeyAlgorithm:         row["key_algorithm"],
			KeyStrength:          parseInt32Ptr(row["key_strength"]),
			KeyUsage:             row["key_usage"],
			SigningAlgorithm:     row["signing_algorithm"],
			NotValidAfter:        parseUnixTime(row["not_valid_after"]),
			NotValidBefore:       parseUnixTime(row["not_valid_before"]),
			Serial:               row["serial"],
			CertificateAuthority: parseBool(row["ca"]),
			Source:               certificateSource(row),
			Username:             certificateUsername(row),
			Path:                 row["path"],
		})
	}
	return certificates
}

func parseCertificateName(queryName string, value string) hosts.CertificateName {
	if value == "" {
		return hosts.CertificateName{}
	}
	return parseDarwinCertificateName(value)
}

// parseDarwinCertificateName splits an osquery-formatted DN string like "/CN=foo/O=bar/OU=baz" into its component fields.
func parseDarwinCertificateName(value string) hosts.CertificateName {
	dn := strings.Trim(value, "/")
	escapedSlash := "\x00SLASH\x00"
	dn = strings.ReplaceAll(dn, `\/`, escapedSlash)
	parts := strings.Split(dn, "/")
	if len(parts) == 1 {
		parts = strings.Split(dn, "+")
	}

	var name hosts.CertificateName
	var organizationalUnits []string
	for _, part := range parts {
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		val = strings.ReplaceAll(val, escapedSlash, "/")
		switch strings.ToUpper(key) {
		case "C":
			name.Country = val
		case "O":
			name.Organization = val
		case "OU":
			organizationalUnits = append(organizationalUnits, val)
		case "CN":
			name.CommonName = val
		}
	}
	name.OrganizationalUnit = strings.Join(organizationalUnits, "+OU=")
	if name == (hosts.CertificateName{}) {
		name.CommonName = value
	}
	return name
}

func certificateSource(row map[string]string) string {
	if source := row["source"]; source != "" {
		return source
	}
	if strings.EqualFold(row["username"], "SYSTEM") {
		return "system"
	}
	return "user"
}

func certificateUsername(row map[string]string) string {
	if username := row["username"]; username != "" && !strings.EqualFold(username, "SYSTEM") {
		return username
	}
	path := filepath.Clean(row["path"])
	const prefix = "/Users/"
	const suffix = "/Library/Keychains/login.keychain-db"
	if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix) {
		return strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	}
	return ""
}

func parseBool(value string) bool {
	switch strings.ToLower(value) {
	case "1", "true", "t", "yes":
		return true
	default:
		return false
	}
}

func parseOsqueryFlags(rows []map[string]string) hosts.DetailUpdate {
	var update hosts.DetailUpdate
	for _, row := range rows {
		switch row["name"] {
		case "distributed_interval":
			update.DistributedInterval = parseInt32Ptr(row["value"])
		case "config_tls_refresh":
			update.ConfigTLSRefresh = parseInt32Ptr(row["value"])
		}
	}
	return update
}

func parseInt32Ptr(value string) *int32 {
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil
	}
	return new(int32(parsed))
}

func parseFloat64Ptr(value string) *float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return new(parsed)
}

type softwareEnrichment map[string]softwarePathEnrichment

type softwarePathEnrichment struct {
	TeamIdentifier   string
	CDHashSHA256     string
	ExecutableSHA256 string
	ExecutablePath   string
}

func softwareEnrichmentByPath(codesignRows []map[string]string, executableRows []map[string]string) softwareEnrichment {
	enrichment := make(softwareEnrichment)
	for _, row := range codesignRows {
		path := row["path"]
		if path == "" {
			continue
		}
		// Read-modify-write preserves fields set by the other row source for the same path.
		info := enrichment[path]
		info.TeamIdentifier = row["team_identifier"]
		info.CDHashSHA256 = row["cdhash_sha256"]
		enrichment[path] = info
	}
	for _, row := range executableRows {
		path := row["path"]
		if path == "" {
			continue
		}
		// Read-modify-write preserves fields set by the other row source for the same path.
		info := enrichment[path]
		info.ExecutableSHA256 = row["executable_sha256"]
		info.ExecutablePath = row["executable_path"]
		enrichment[path] = info
	}
	return enrichment
}

func parseSoftwareRows(rows []map[string]string, enrichment softwareEnrichment) []software.HostSoftwareEntry {
	entries := make([]software.HostSoftwareEntry, 0, len(rows))
	for _, row := range rows {
		name := row["name"]
		if name == "" {
			continue
		}
		installedPath := row["installed_path"]
		pathEnrichment := enrichment[installedPath]
		entries = append(entries, software.HostSoftwareEntry{
			Name:             name,
			Version:          row["version"],
			Source:           row["source"],
			BundleIdentifier: row["bundle_identifier"],
			ExtensionID:      row["extension_id"],
			ExtensionFor:     row["extension_for"],
			Vendor:           row["vendor"],
			Arch:             row["arch"],
			Release:          row["release"],
			InstalledPath:    installedPath,
			TeamIdentifier:   pathEnrichment.TeamIdentifier,
			CDHashSHA256:     pathEnrichment.CDHashSHA256,
			ExecutableSHA256: pathEnrichment.ExecutableSHA256,
			ExecutablePath:   pathEnrichment.ExecutablePath,
			LastOpenedAt:     parseSoftwareLastOpenedAt(row),
		})
	}
	return entries
}

func parseSoftwareLastOpenedAt(row map[string]string) *time.Time {
	return parseUnixTime(row["last_opened_at"])
}

func parseUnixTime(value string) *time.Time {
	if value == "" || value == "0" || strings.EqualFold(value, "null") {
		return nil
	}
	seconds, nanos, ok := parseUnixTimeParts(value)
	if !ok {
		return nil
	}
	return new(time.Unix(seconds, nanos).UTC())
}

func parseUnixTimeParts(value string) (int64, int64, bool) {
	secondsRaw, fractionRaw, _ := strings.Cut(value, ".")
	seconds, err := strconv.ParseInt(secondsRaw, 10, 64)
	if err != nil || seconds <= 0 {
		return 0, 0, false
	}
	if fractionRaw == "" {
		return seconds, 0, true
	}
	if len(fractionRaw) > 9 {
		fractionRaw = fractionRaw[:9]
	}
	fractionRaw += strings.Repeat("0", 9-len(fractionRaw))
	nanos, err := strconv.ParseInt(fractionRaw, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return seconds, nanos, true
}
