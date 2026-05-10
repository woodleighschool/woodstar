package inventory

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/software"
)

const osqueryFlagConfigRefresh = "config_refresh"

type Projector struct {
	hostStore     *hosts.HostStore
	softwareStore *software.SoftwareStore
	logger        *slog.Logger
}

func NewProjector(hostStore *hosts.HostStore, softwareStore *software.SoftwareStore, logger *slog.Logger) *Projector {
	return &Projector{hostStore: hostStore, softwareStore: softwareStore, logger: logger}
}

func (p *Projector) MarkFresh(ctx context.Context, hostID int64) error {
	return p.hostStore.MarkDetailFresh(ctx, hostID, DetailQueryHash())
}

func (p *Projector) IngestSoftwareMacOSWithEnrichment(
	ctx context.Context,
	hostID int64,
	rows []map[string]string,
	queryRows map[string][]map[string]string,
) error {
	if p.softwareStore == nil {
		return nil
	}
	enrichment := softwareEnrichmentByPath(
		queryRows[querySoftwareMacOSCodesign],
		queryRows[querySoftwareMacOSExecutableHash],
	)
	rows = append(rows, queryRows[querySoftwareVSCodeExtensions]...)
	rows = append(rows, queryRows[querySoftwareJetBrainsPlugins]...)
	rows = append(rows, queryRows[querySoftwareGoBinaries]...)
	rows = append(rows, queryRows[querySoftwarePythonPackages]...)
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
			"codesign_count", len(queryRows[querySoftwareMacOSCodesign]),
			"executable_hash_count", len(queryRows[querySoftwareMacOSExecutableHash]),
		)
	}
	return nil
}

func ingestOSVersion(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return projector.hostStore.ApplyDetail(
		ctx,
		hostID,
		ParseHostDetails(map[string]map[string]string{queryOSVersion: rows[0]}),
	)
}

func ingestSystemInfo(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return projector.hostStore.ApplyDetail(
		ctx,
		hostID,
		ParseHostDetails(map[string]map[string]string{querySystemInfo: rows[0]}),
	)
}

func ingestOsqueryInfo(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return projector.hostStore.ApplyDetail(
		ctx,
		hostID,
		ParseHostDetails(map[string]map[string]string{queryOsqueryInfo: rows[0]}),
	)
}

func ingestOsqueryFlags(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	return projector.hostStore.ApplyDetail(ctx, hostID, parseOsqueryFlags(rows))
}

func ingestOrbitInfo(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return projector.hostStore.ApplyDetail(
		ctx,
		hostID,
		ParseHostDetails(map[string]map[string]string{queryOrbitInfo: rows[0]}),
	)
}

func ingestUptime(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	update := ParseHostDetails(map[string]map[string]string{queryUptime: rows[0]})
	if update.UptimeSeconds != nil {
		restarted := time.Now().Add(-time.Duration(*update.UptimeSeconds) * time.Second)
		update.LastRestartedAt = &restarted
	}
	return projector.hostStore.ApplyDetail(ctx, hostID, update)
}

func ingestRootDisk(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return projector.hostStore.ApplyDetail(
		ctx,
		hostID,
		ParseHostDetails(map[string]map[string]string{queryRootDisk: rows[0]}),
	)
}

func ingestPrimaryInterface(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return projector.hostStore.ApplyDetail(
		ctx,
		hostID,
		ParseHostDetails(map[string]map[string]string{queryPrimaryInterface: rows[0]}),
	)
}

func ingestUsers(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	return projector.hostStore.ReplaceUsers(ctx, hostID, parseHostUsers(rows))
}

func ingestBatteries(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	return projector.hostStore.ReplaceBatteries(ctx, hostID, parseHostBatteries(rows))
}

func ingestSoftwareMacOS(ctx context.Context, projector *Projector, hostID int64, rows []map[string]string) error {
	if projector.softwareStore == nil {
		return nil
	}
	return projector.softwareStore.ReplaceHostSoftware(ctx, hostID, parseSoftwareRows(rows, softwareEnrichment{}))
}

func ingestNoop(context.Context, *Projector, int64, []map[string]string) error {
	return nil
}

func parseHostUsers(rows []map[string]string) []hosts.HostUser {
	users := make([]hosts.HostUser, 0, len(rows))
	for _, row := range rows {
		username := strings.TrimSpace(row["username"])
		uid := strings.TrimSpace(row["uid"])
		if username == "" || uid == "" {
			continue
		}
		users = append(users, hosts.HostUser{
			UID:         uid,
			Username:    username,
			Type:        strings.TrimSpace(row["type"]),
			Description: strings.TrimSpace(row["description"]),
			Directory:   strings.TrimSpace(row["directory"]),
			Shell:       strings.TrimSpace(row["shell"]),
		})
	}
	return users
}

func parseHostBatteries(rows []map[string]string) []hosts.HostBattery {
	batteries := make([]hosts.HostBattery, 0, len(rows))
	for _, row := range rows {
		serialNumber := strings.TrimSpace(row["serial_number"])
		if serialNumber == "" {
			continue
		}
		batteries = append(batteries, hosts.HostBattery{
			SerialNumber:     serialNumber,
			Manufacturer:     strings.TrimSpace(row["manufacturer"]),
			Model:            strings.TrimSpace(row["model"]),
			Chemistry:        strings.TrimSpace(row["chemistry"]),
			CycleCount:       parseInt32Ptr(row["cycle_count"]),
			Health:           strings.TrimSpace(row["health"]),
			DesignedCapacity: parseInt32Ptr(row["designed_capacity"]),
			MaxCapacity:      parseInt32Ptr(row["max_capacity"]),
			CurrentCapacity:  parseInt32Ptr(row["current_capacity"]),
			PercentRemaining: parseFloat64Ptr(row["percent_remaining"]),
		})
	}
	return batteries
}

func parseOsqueryFlags(rows []map[string]string) hosts.HostDetailUpdate {
	var update hosts.HostDetailUpdate
	var configRefresh *int32
	for _, row := range rows {
		switch strings.TrimSpace(row["name"]) {
		case "distributed_interval":
			update.DistributedInterval = parseInt32Ptr(row["value"])
		case "config_tls_refresh":
			update.ConfigTLSRefresh = parseInt32Ptr(row["value"])
		case osqueryFlagConfigRefresh:
			configRefresh = parseInt32Ptr(row["value"])
		}
	}
	if update.ConfigTLSRefresh == nil {
		update.ConfigTLSRefresh = configRefresh
	}
	return update
}

func parseInt32Ptr(value string) *int32 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return nil
	}
	out := int32(parsed)
	return &out
}

func parseFloat64Ptr(value string) *float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return nil
	}
	return &parsed
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
		path := strings.TrimSpace(row["path"])
		if path == "" {
			continue
		}
		info := enrichment[path]
		info.TeamIdentifier = strings.TrimSpace(row["team_identifier"])
		info.CDHashSHA256 = strings.TrimSpace(row["cdhash_sha256"])
		enrichment[path] = info
	}
	for _, row := range executableRows {
		path := strings.TrimSpace(row["path"])
		if path == "" {
			continue
		}
		info := enrichment[path]
		info.ExecutableSHA256 = strings.TrimSpace(row["executable_sha256"])
		info.ExecutablePath = strings.TrimSpace(row["executable_path"])
		enrichment[path] = info
	}
	return enrichment
}

func parseSoftwareRows(rows []map[string]string, enrichment softwareEnrichment) []software.HostSoftwareEntry {
	entries := make([]software.HostSoftwareEntry, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row["name"])
		if name == "" {
			continue
		}
		installedPath := installedPathForSoftware(row)
		pathEnrichment := enrichment[installedPath]
		entries = append(entries, software.HostSoftwareEntry{
			Name:             name,
			Version:          versionForSoftware(row),
			Source:           strings.TrimSpace(row["source"]),
			BundleIdentifier: strings.TrimSpace(row["bundle_identifier"]),
			ExtensionID:      strings.TrimSpace(row["extension_id"]),
			ExtensionFor:     strings.TrimSpace(row["extension_for"]),
			Vendor:           strings.TrimSpace(row["vendor"]),
			Arch:             strings.TrimSpace(row["arch"]),
			Release:          strings.TrimSpace(row["release"]),
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

func versionForSoftware(row map[string]string) string {
	if version := strings.TrimSpace(row["version"]); version != "" {
		return version
	}
	return strings.TrimSpace(row["bundle_short_version"])
}

func installedPathForSoftware(row map[string]string) string {
	if path := strings.TrimSpace(row["installed_path"]); path != "" {
		return path
	}
	return strings.TrimSpace(row["path"])
}

func parseSoftwareLastOpenedAt(row map[string]string) *time.Time {
	if value := strings.TrimSpace(row["last_opened_at"]); value != "" {
		return parseUnixTime(value)
	}
	return parseUnixTime(row["last_opened_time"])
}

func parseUnixTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" || strings.EqualFold(value, "null") {
		return nil
	}
	seconds, nanos, ok := parseUnixTimeParts(value)
	if !ok {
		return nil
	}
	opened := time.Unix(seconds, nanos).UTC()
	return &opened
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
	for len(fractionRaw) < 9 {
		fractionRaw += "0"
	}
	nanos, err := strconv.ParseInt(fractionRaw, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return seconds, nanos, true
}
