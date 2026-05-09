package osquery

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type resultLogRow struct {
	Name         string              `json:"name"`
	CalendarTime string              `json:"calendarTime"`
	Snapshot     []map[string]string `json:"snapshot"`
	Action       string              `json:"action"`
}

// ingestReportLogs writes per-host snapshot results emitted by osquery's
// scheduled query log to query_results, replacing the previous snapshot.
func (s *Service) ingestReportLogs(ctx context.Context, hostID int64, data json.RawMessage) error {
	var logs []resultLogRow
	if err := json.Unmarshal(data, &logs); err != nil {
		var single resultLogRow
		if err := json.Unmarshal(data, &single); err != nil {
			return err
		}
		logs = []resultLogRow{single}
	}

	for _, item := range logs {
		kind, suffix, ok := parseQueryName(item.Name)
		if !ok || kind != kindReport {
			continue
		}
		queryID, ok := parsePositiveSuffix(suffix)
		if !ok {
			continue
		}
		fetchedAt := parseCalendarTime(item.CalendarTime)
		if err := s.queries.OverwriteResults(ctx, queryID, hostID, item.Snapshot, fetchedAt); err != nil {
			return err
		}
	}
	return nil
}

func parseCalendarTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().UTC()
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "Mon Jan 2 15:04:05 2006 UTC"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC()
	}
	return time.Now().UTC()
}
