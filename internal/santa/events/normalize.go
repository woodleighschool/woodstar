package events

import (
	"slices"
	"strings"
)

func normalizeStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(normalizeString(value))
		if value == "" || slices.Contains(out, value) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeString(value string) string {
	return strings.ReplaceAll(value, "\x00", "")
}

func normalizeTeamID(value string) string {
	if value == "<unknown team id>" {
		return ""
	}
	return value
}

func normalizeSigningStatus(status SigningStatus) SigningStatus {
	if slices.Contains(SigningStatusValues, status) {
		return status
	}
	return SigningStatusUnspecified
}
