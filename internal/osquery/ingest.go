package osquery

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

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

func ingestSoftwareMacOS(ctx context.Context, svc *Service, hostID int64, rows []map[string]string) error {
	if svc.software == nil {
		return nil
	}
	return svc.software.ReplaceHostSoftware(ctx, hostID, parseSoftwareRows(rows))
}

func parseSoftwareRows(rows []map[string]string) []models.HostSoftwareEntry {
	entries := make([]models.HostSoftwareEntry, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row["name"])
		if name == "" {
			continue
		}
		entries = append(entries, models.HostSoftwareEntry{
			Name:             name,
			Version:          versionForSoftware(row),
			Source:           strings.TrimSpace(row["source"]),
			BundleIdentifier: strings.TrimSpace(row["bundle_identifier"]),
			InstalledPath:    strings.TrimSpace(row["path"]),
			LastOpenedAt:     parseUnixTime(row["last_opened_time"]),
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

func parseUnixTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" || strings.EqualFold(value, "null") {
		return nil
	}
	seconds := parseInt64(value)
	if seconds <= 0 {
		log.Debug().Str("value", value).Msg("ignoring invalid osquery last_opened_time")
		return nil
	}
	opened := time.Unix(seconds, 0).UTC()
	return &opened
}
