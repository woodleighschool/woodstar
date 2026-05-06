package osquery

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/models"
)

func ingestOSVersion(ctx context.Context, svc *Service, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return svc.hosts.ApplyDetail(ctx, hostID, ParseHostDetails(map[string]map[string]string{queryOSVersion: rows[0]}))
}

func ingestSystemInfo(ctx context.Context, svc *Service, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return svc.hosts.ApplyDetail(ctx, hostID, ParseHostDetails(map[string]map[string]string{querySystemInfo: rows[0]}))
}

func ingestOsqueryInfo(ctx context.Context, svc *Service, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return svc.hosts.ApplyDetail(ctx, hostID, ParseHostDetails(map[string]map[string]string{queryOsqueryInfo: rows[0]}))
}

func ingestOrbitInfo(ctx context.Context, svc *Service, hostID int64, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	return svc.hosts.ApplyDetail(ctx, hostID, ParseHostDetails(map[string]map[string]string{queryOrbitInfo: rows[0]}))
}

func ingestSoftwareMacOS(ctx context.Context, svc *Service, hostID int64, rows []map[string]string) error {
	if svc.software == nil {
		return nil
	}
	return svc.software.ReplaceHostSoftware(ctx, hostID, parseSoftwareRows(rows, softwareEnrichment{}))
}

func ingestNoop(context.Context, *Service, int64, []map[string]string) error {
	return nil
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

func parseSoftwareRows(rows []map[string]string, enrichment softwareEnrichment) []models.HostSoftwareEntry {
	entries := make([]models.HostSoftwareEntry, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row["name"])
		if name == "" {
			continue
		}
		installedPath := installedPathForSoftware(row)
		pathEnrichment := enrichment[installedPath]
		entries = append(entries, models.HostSoftwareEntry{
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
