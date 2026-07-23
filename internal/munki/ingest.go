package munki

import (
	"context"
	"strings"
	"time"
)

// hostStateStore persists observed Munki host state.
type hostStateStore interface {
	UpsertHostObservation(ctx context.Context, observation HostObservation) error
	ClearHostObservation(ctx context.Context, hostID int64) error
	ReplaceHostItems(ctx context.Context, hostID int64, items []ItemObservation) error
}

// DetailIngestor projects osquery munki_info and munki_installs detail rows into
// observed host state. It is registered onto the osquery projector from the
// wiring layer so the projector core stays free of Munki types.
type DetailIngestor struct {
	store hostStateStore
}

// NewDetailIngestor returns an ingestor that writes observed state to store.
func NewDetailIngestor(store hostStateStore) *DetailIngestor {
	return &DetailIngestor{store: store}
}

// IngestInfo records a host's munki_info observation, clearing it when the host
// reports no Munki data.
func (i *DetailIngestor) IngestInfo(ctx context.Context, hostID int64, rows []map[string]string) error {
	status, ok := hostStatusFromInfoRows(hostID, rows)
	if !ok {
		return i.store.ClearHostObservation(ctx, hostID)
	}
	return i.store.UpsertHostObservation(ctx, status)
}

// IngestInstalls records the managed-install items a host reports.
func (i *DetailIngestor) IngestInstalls(ctx context.Context, hostID int64, rows []map[string]string) error {
	return i.store.ReplaceHostItems(ctx, hostID, itemsFromInstallRows(hostID, rows))
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
