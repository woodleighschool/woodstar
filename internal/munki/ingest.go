package munki

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// hostStateStore persists observed Munki host state.
type hostStateStore interface {
	ApplyEnvelope(ctx context.Context, result EnvelopeResult) error
}

// DetailIngestor projects osquery munki_info and munki_installs detail rows into
// observed host state.
type DetailIngestor struct {
	store hostStateStore
}

// NewDetailIngestor returns an ingestor that writes observed state to store.
func NewDetailIngestor(store hostStateStore) *DetailIngestor {
	return &DetailIngestor{store: store}
}

// IngestEnvelope projects the whole Munki osquery query family into one
// authoritative collection attempt.
func (i *DetailIngestor) IngestEnvelope(ctx context.Context, hostID int64, envelope EnvelopeInput) error {
	if !envelope.Info.Present && !envelope.Installs.Present {
		return nil
	}
	result := EnvelopeResult{HostID: hostID, AttemptedAt: time.Now().UTC()}
	if diagnostics := envelopeDiagnostics(envelope); diagnostics != "" {
		result.CollectionError = diagnostics
		return i.store.ApplyEnvelope(ctx, result)
	}

	result.Complete = true
	result.Observation, result.HasReport = hostStatusFromInfoRows(hostID, envelope.Info.Rows)
	if result.HasReport {
		result.Items = itemsFromInstallRows(hostID, envelope.Installs.Rows)
	}
	return i.store.ApplyEnvelope(ctx, result)
}

func envelopeDiagnostics(envelope EnvelopeInput) string {
	diagnostics := []string{
		queryResultDiagnostic("munki_info", envelope.Info),
		queryResultDiagnostic("munki_installs", envelope.Installs),
	}
	return strings.Join(compactStrings(diagnostics), "; ")
}

func queryResultDiagnostic(name string, result QueryResult) string {
	if !result.Present {
		return name + ": missing result"
	}
	if result.Status == 0 {
		return ""
	}
	if result.Message != "" {
		return fmt.Sprintf("%s: %s", name, result.Message)
	}
	return fmt.Sprintf("%s: status %d", name, result.Status)
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func hostStatusFromInfoRows(hostID int64, rows []map[string]string) (HostObservation, bool) {
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

func itemsFromInstallRows(hostID int64, rows []map[string]string) []ItemObservation {
	items := make([]ItemObservation, 0, len(rows))
	for _, row := range rows {
		items = append(items, ItemObservation{
			HostID:           hostID,
			Name:             row["name"],
			DisplayName:      row["display_name"],
			Installed:        row["installed"] == "true",
			InstalledVersion: row["installed_version"],
			TargetVersion:    row["version_to_install"],
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
