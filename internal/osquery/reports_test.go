package osquery

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

func TestIngestReportLogsUsesUnixTime(t *testing.T) {
	store := &recordingReportStore{}
	service := NewAgentService(Dependencies{
		ReportStore: store,
		Logger:      slog.New(slog.DiscardHandler),
	})

	err := service.ingestReportLogs(context.Background(), 42, json.RawMessage(`{
		"name":"woodstar_report_query_7",
		"calendarTime":"not a timestamp",
		"unixTime":1778848496,
		"action":"snapshot",
		"snapshot":[{"name":"Alpha"}]
	}`))
	if err != nil {
		t.Fatalf("ingestReportLogs returned error: %v", err)
	}
	if store.reportID != 7 || store.hostID != 42 {
		t.Fatalf("stored report/host = %d/%d, want 7/42", store.reportID, store.hostID)
	}
	if len(store.rows) != 1 || store.rows[0]["name"] != "Alpha" {
		t.Fatalf("stored rows = %#v, want Alpha snapshot", store.rows)
	}
	wantTime := time.Unix(1778848496, 0).UTC()
	if !store.fetchedAt.Equal(wantTime) {
		t.Fatalf("fetchedAt = %s, want %s", store.fetchedAt, wantTime)
	}
}

func TestIngestReportLogsRejectsIncompleteSnapshotMetadata(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{
			name: "missing unix time",
			data: `{"name":"woodstar_report_query_7","action":"snapshot","snapshot":[]}`,
		},
		{
			name: "wrong action",
			data: `{"name":"woodstar_report_query_7","unixTime":1778848496,"action":"added","snapshot":[]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &recordingReportStore{}
			service := NewAgentService(Dependencies{
				ReportStore: store,
				Logger:      slog.New(slog.DiscardHandler),
			})
			if err := service.ingestReportLogs(context.Background(), 42, json.RawMessage(tc.data)); err == nil {
				t.Fatal("ingestReportLogs returned nil error")
			}
			if store.calls != 0 {
				t.Fatalf("OverwriteResults calls = %d, want 0", store.calls)
			}
		})
	}
}

type recordingReportStore struct {
	calls     int
	reportID  int64
	hostID    int64
	rows      []map[string]string
	fetchedAt time.Time
}

func (*recordingReportStore) ScheduledForHost(context.Context, *hosts.Host) ([]reports.Report, error) {
	return nil, nil
}

func (s *recordingReportStore) OverwriteResults(
	_ context.Context,
	reportID int64,
	hostID int64,
	rows []map[string]string,
	fetchedAt time.Time,
) error {
	s.calls++
	s.reportID = reportID
	s.hostID = hostID
	s.rows = rows
	s.fetchedAt = fetchedAt
	return nil
}
