package osquery

import (
	"context"
)

// ScheduleEntry is one osquery scheduled query config item.
type ScheduleEntry struct {
	Query    string `json:"query"`
	Interval int32  `json:"interval"`
	Snapshot bool   `json:"snapshot"`
	Removed  bool   `json:"removed"`
	Version  string `json:"version,omitempty"`
}

// buildScheduleForHost returns the per-host osquery schedule map for reports.
func buildScheduleForHost(
	ctx context.Context,
	store reportStore,
	hostID int64,
) (map[string]ScheduleEntry, error) {
	if store == nil {
		return map[string]ScheduleEntry{}, nil
	}
	items, err := store.ScheduledForHost(ctx, hostID)
	if err != nil {
		return nil, err
	}
	schedule := make(map[string]ScheduleEntry, len(items))
	for _, item := range items {
		entry := ScheduleEntry{
			Query:    item.Query,
			Interval: item.ScheduleInterval,
			Snapshot: true,
		}
		if item.MinOsqueryVersion != nil {
			entry.Version = *item.MinOsqueryVersion
		}
		schedule[queryNameForSQL(kindReport, item.ID, item.Query)] = entry
	}
	return schedule, nil
}
