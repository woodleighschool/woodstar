package osquery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type resultLogRow struct {
	Name     string              `json:"name"`
	UnixTime int64               `json:"unixTime"`
	Snapshot []map[string]string `json:"snapshot"`
	Action   string              `json:"action"`
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
		if item.Action != "snapshot" {
			return fmt.Errorf("report %d: action must be snapshot", reportID)
		}
		if item.UnixTime <= 0 {
			return fmt.Errorf("report %d: unixTime must be positive", reportID)
		}
		fetchedAt := time.Unix(item.UnixTime, 0).UTC()
		if err := s.deps.ReportStore.OverwriteResults(ctx, reportID, hostID, item.Snapshot, fetchedAt); err != nil {
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
