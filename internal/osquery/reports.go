package osquery

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

type resultLogRow struct {
	Name         string              `json:"name"`
	CalendarTime string              `json:"calendarTime"`
	Snapshot     []map[string]string `json:"snapshot"`
	Action       string              `json:"action"`
}

// ingestReportLogs writes per-host snapshot results emitted by osquery's
// scheduled query log to report_results, replacing the previous snapshot.
func (s *AgentService) ingestReportLogs(ctx context.Context, hostID int64, data json.RawMessage) error {
	var logs []resultLogRow
	if err := json.Unmarshal(data, &logs); err != nil {
		var single resultLogRow
		if err := json.Unmarshal(data, &single); err != nil {
			return err
		}
		logs = []resultLogRow{single}
	}

	for _, item := range logs {
		reportID, ok := parseReportQueryName(item.Name)
		if !ok {
			continue
		}
		fetchedAt := parseCalendarTime(item.CalendarTime)
		if err := s.deps.ReportStore.OverwriteResults(ctx, reportID, hostID, item.Snapshot, fetchedAt); err != nil {
			if errors.Is(err, reports.ErrSnapshotTooLarge) {
				s.deps.Logger.WarnContext(ctx, "snapshot dropped", "report_id", reportID, "host_id", hostID, "err", err)
				continue
			}
			return err
		}
	}
	return nil
}

func parseReportQueryName(name string) (int64, bool) {
	suffix, ok := strings.CutPrefix(name, queryName(kindReport, ""))
	if !ok {
		return 0, false
	}
	return parsePositiveSuffix(suffix)
}

func parseCalendarTime(value string) time.Time {
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
