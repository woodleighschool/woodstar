package osquery

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/agents/queries"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// ScheduleEntry is one osquery scheduled query config item.
type ScheduleEntry struct {
	Query    string `json:"query"`
	Interval int    `json:"interval"`
	Snapshot bool   `json:"snapshot"`
	Removed  bool   `json:"removed"`
	Platform string `json:"platform,omitempty"`
}

// buildScheduleForHost returns the per-host osquery schedule map for reports.
func buildScheduleForHost(
	ctx context.Context,
	store *queries.Store,
	host *hosts.Host,
) (map[string]ScheduleEntry, error) {
	if store == nil {
		return map[string]ScheduleEntry{}, nil
	}
	items, err := store.ScheduledForHost(ctx, host)
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
		if item.Platform != nil {
			entry.Platform = *item.Platform
		}
		schedule[queryNameID(kindReport, item.ID)] = entry
	}
	return schedule, nil
}
