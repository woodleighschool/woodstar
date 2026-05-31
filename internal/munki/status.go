package munki

import "strings"

func HostStatusFromInfoRows(hostID int64, rows []map[string]string) (HostStatusObservation, bool) {
	if len(rows) == 0 {
		return HostStatusObservation{}, false
	}
	row := rows[0]
	return HostStatusObservation{
		HostID:          hostID,
		Version:         row["version"],
		ManifestName:    row["manifest_name"],
		Success:         parseBoolPtr(row["success"]),
		Errors:          splitMunkiList(row["errors"]),
		Warnings:        splitMunkiList(row["warnings"]),
		ProblemInstalls: splitMunkiList(row["problem_installs"]),
		RunStartedAt:    row["start_time"],
		RunEndedAt:      row["end_time"],
	}, true
}

func HostItemsFromInstallRows(hostID int64, rows []map[string]string) []HostItem {
	items := make([]HostItem, 0, len(rows))
	for _, row := range rows {
		name := row["name"]
		if name == "" {
			continue
		}
		items = append(items, HostItem{
			HostID:           hostID,
			Name:             name,
			Installed:        parseBool(row["installed"]),
			InstalledVersion: row["installed_version"],
			RunEndedAt:       row["end_time"],
		})
	}
	return items
}

func splitMunkiList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == '\n'
	})
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

func parseBoolPtr(value string) *bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed := parseBool(value)
	return &parsed
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "t", "yes":
		return true
	default:
		return false
	}
}
