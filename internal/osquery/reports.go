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

type statusLogRow struct {
	UnixTime int64  `json:"unixTime"`
	Message  string `json:"message"`
}

const scheduledQueryErrorPrefix = "Error executing scheduled query "

// ingestReportLogs writes the latest per-host snapshots emitted by osquery's
// scheduled query log.
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
		reportID, queryHash, ok := parseReportQueryName(item.Name)
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
		if err := s.deps.ReportStore.OverwriteSnapshot(
			ctx,
			reportID,
			queryHash,
			hostID,
			item.Snapshot,
			fetchedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

// ingestReportStatusLogs writes scheduled-query errors for reports.
func (s *AgentService) ingestReportStatusLogs(
	ctx context.Context,
	hostID int64,
	data json.RawMessage,
) error {
	var logs []statusLogRow
	if err := json.Unmarshal(data, &logs); err != nil {
		var single statusLogRow
		if err := json.Unmarshal(data, &single); err != nil {
			return err
		}
		logs = []statusLogRow{single}
	}

	for _, item := range logs {
		reportID, queryHash, reportError, ok := parseReportErrorMessage(item.Message)
		if !ok {
			continue
		}
		if item.UnixTime <= 0 {
			return fmt.Errorf("report %d: unixTime must be positive", reportID)
		}
		if err := s.deps.ReportStore.OverwriteError(
			ctx,
			reportID,
			queryHash,
			hostID,
			reportError,
			time.Unix(item.UnixTime, 0).UTC(),
		); err != nil {
			return err
		}
	}
	return nil
}

func parseReportErrorMessage(message string) (int64, string, string, bool) {
	remainder, ok := strings.CutPrefix(message, scheduledQueryErrorPrefix)
	if !ok {
		return 0, "", "", false
	}
	name, reportError, ok := strings.Cut(remainder, ": ")
	if !ok || strings.TrimSpace(reportError) == "" {
		return 0, "", "", false
	}
	reportID, queryHash, ok := parseReportQueryName(name)
	if !ok {
		return 0, "", "", false
	}
	return reportID, queryHash, reportError, true
}

func parseReportQueryName(name string) (int64, string, bool) {
	suffix, ok := strings.CutPrefix(name, queryName(kindReport, ""))
	if !ok {
		return 0, "", false
	}
	return parseQueryIdentity(suffix)
}
