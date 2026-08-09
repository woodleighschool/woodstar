package munki

import (
	"context"
	"strings"
	"time"
)

type hostStateStore interface {
	applyCollection(ctx context.Context, update collectionUpdate) error
}

// DetailIngestor stores the Munki query family as one collection attempt.
type DetailIngestor struct {
	store hostStateStore
}

// NewDetailIngestor returns an ingestor that writes observed state to store.
func NewDetailIngestor(store hostStateStore) *DetailIngestor {
	return &DetailIngestor{store: store}
}

// IngestCollection replaces the last successful Munki snapshot.
func (i *DetailIngestor) IngestCollection(ctx context.Context, hostID int64, collection Collection) error {
	if !collectionAuthoritative(collection) {
		return nil
	}

	update := collectionUpdate{HostID: hostID}
	update.Observation, update.HasReport = hostStatusFromInfoRows(collection.Info.Rows)
	if update.HasReport {
		update.Items = itemsFromInstallRows(collection.Installs.Rows)
	}
	return i.store.applyCollection(ctx, update)
}

func collectionAuthoritative(collection Collection) bool {
	if !collection.Info.Present && !collection.Installs.Present {
		return true
	}
	return collection.Info.Present && collection.Info.Successful &&
		collection.Installs.Present && collection.Installs.Successful
}

func hostStatusFromInfoRows(rows []map[string]string) (hostObservation, bool) {
	if len(rows) == 0 {
		return hostObservation{}, false
	}
	row := rows[0]
	return hostObservation{
		Version:         row["version"],
		ManifestName:    row["manifest_name"],
		Errors:          splitMunkiList(row["errors"]),
		Warnings:        splitMunkiList(row["warnings"]),
		ProblemInstalls: splitMunkiList(row["problem_installs"]),
		RunStartedAt:    parseMunkiTime(row["start_time"]),
		RunEndedAt:      parseMunkiTime(row["end_time"]),
	}, true
}

func itemsFromInstallRows(rows []map[string]string) []itemObservation {
	items := make([]itemObservation, 0, len(rows))
	for _, row := range rows {
		items = append(items, itemObservation{
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
