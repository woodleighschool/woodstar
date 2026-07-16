package munki

import (
	"strings"
	"time"
)

func HostStatusFromInfoRows(hostID int64, rows []map[string]string) (HostObservation, bool) {
	if len(rows) == 0 {
		return HostObservation{}, false
	}
	row := rows[0]
	return HostObservation{
		HostID:          hostID,
		Version:         row["version"],
		ManifestName:    row["manifest_name"],
		Errors:          splitMunkiList(row["errors"]),
		Warnings:        splitMunkiList(row["warnings"]),
		ProblemInstalls: splitMunkiList(row["problem_installs"]),
		RunStartedAt:    parseMunkiTime(row["start_time"]),
		RunEndedAt:      parseMunkiTime(row["end_time"]),
	}, true
}

func ItemsFromInstallRows(hostID int64, rows []map[string]string) []Item {
	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, Item{
			HostID:           hostID,
			Name:             row["name"],
			Installed:        row["installed"] == "true",
			InstalledVersion: row["installed_version"],
		})
	}
	return items
}

func splitMunkiList(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ";")
}

func parseMunkiTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02 15:04:05 -0700", value)
	if err != nil {
		return nil
	}
	return &parsed
}
