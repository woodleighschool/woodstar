package osquery

import (
	"context"
	"encoding/json"
	"fmt"
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

	reportSQL := "select 1;"
	err := service.ingestReportLogs(context.Background(), 42, json.RawMessage(fmt.Sprintf(`{
		"name":%q,
		"calendarTime":"not a timestamp",
		"unixTime":1778848496,
		"action":"snapshot",
		"snapshot":[{"name":"Alpha"}]
	}`, queryNameForSQL(kindReport, 7, reportSQL))))
	if err != nil {
		t.Fatalf("ingestReportLogs returned error: %v", err)
	}
	if store.reportID != 7 || store.queryHash != queryHash(reportSQL) || store.hostID != 42 {
		t.Fatalf(
			"stored report/hash/host = %d/%q/%d, want 7/%q/42",
			store.reportID,
			store.queryHash,
			store.hostID,
			queryHash(reportSQL),
		)
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
	reportName := queryNameForSQL(kindReport, 7, "select 1;")
	for _, tc := range []struct {
		name string
		data string
	}{
		{
			name: "missing unix time",
			data: fmt.Sprintf(`{"name":%q,"action":"snapshot","snapshot":[]}`, reportName),
		},
		{
			name: "wrong action",
			data: fmt.Sprintf(
				`{"name":%q,"unixTime":1778848496,"action":"added","snapshot":[]}`,
				reportName,
			),
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
				t.Fatalf("OverwriteSnapshot calls = %d, want 0", store.calls)
			}
		})
	}
}

func TestIngestReportStatusLogsStoresScheduledQueryError(t *testing.T) {
	store := &recordingReportStore{}
	service := NewAgentService(Dependencies{
		ReportStore: store,
		Logger:      slog.New(slog.DiscardHandler),
	})

	reportSQL := "select upn from app_sso_platform_info;"
	reportName := queryNameForSQL(kindReport, 7, reportSQL)
	err := service.ingestReportStatusLogs(context.Background(), 42, json.RawMessage(fmt.Sprintf(`{
		"unixTime":1778848496,
		"severity":2,
		"filename":"scheduler.cpp",
		"message":%q
	}`, scheduledQueryErrorPrefix+reportName+`: error generating table: missing "upn" key`)))
	if err != nil {
		t.Fatalf("ingestReportStatusLogs returned error: %v", err)
	}
	if store.reportID != 7 || store.queryHash != queryHash(reportSQL) || store.hostID != 42 {
		t.Fatalf(
			"stored report/hash/host = %d/%q/%d, want 7/%q/42",
			store.reportID,
			store.queryHash,
			store.hostID,
			queryHash(reportSQL),
		)
	}
	if store.reportError != `error generating table: missing "upn" key` {
		t.Fatalf("stored error = %q", store.reportError)
	}
	wantTime := time.Unix(1778848496, 0).UTC()
	if !store.fetchedAt.Equal(wantTime) {
		t.Fatalf("fetchedAt = %s, want %s", store.fetchedAt, wantTime)
	}
}

func TestIngestReportStatusLogsIgnoresUnrelatedStatus(t *testing.T) {
	store := &recordingReportStore{}
	service := NewAgentService(Dependencies{
		ReportStore: store,
		Logger:      slog.New(slog.DiscardHandler),
	})

	err := service.ingestReportStatusLogs(context.Background(), 42, json.RawMessage(`[
		{"unixTime":1778848496,"message":"Scheduled query completed"},
		{"unixTime":1778848496,"message":"Error executing scheduled query pack_other: failed"}
	]`))
	if err != nil {
		t.Fatalf("ingestReportStatusLogs returned error: %v", err)
	}
	if store.calls != 0 {
		t.Fatalf("report store calls = %d, want 0", store.calls)
	}
}

func TestIngestReportStatusLogsRejectsMissingUnixTime(t *testing.T) {
	store := &recordingReportStore{}
	service := NewAgentService(Dependencies{
		ReportStore: store,
		Logger:      slog.New(slog.DiscardHandler),
	})
	reportName := queryNameForSQL(kindReport, 7, "select 1;")
	data := json.RawMessage(fmt.Sprintf(
		`{"message":%q}`,
		scheduledQueryErrorPrefix+reportName+": failed",
	))
	if err := service.ingestReportStatusLogs(context.Background(), 42, data); err == nil {
		t.Fatal("ingestReportStatusLogs returned nil error")
	}
	if store.calls != 0 {
		t.Fatalf("report store calls = %d, want 0", store.calls)
	}
}

type recordingReportStore struct {
	calls       int
	reportID    int64
	queryHash   string
	hostID      int64
	rows        []map[string]string
	reportError string
	fetchedAt   time.Time
}

func (s *recordingReportStore) OverwriteError(
	_ context.Context,
	reportID int64,
	queryHash string,
	hostID int64,
	reportError string,
	fetchedAt time.Time,
) error {
	s.calls++
	s.reportID = reportID
	s.queryHash = queryHash
	s.hostID = hostID
	s.reportError = reportError
	s.fetchedAt = fetchedAt
	return nil
}

func (*recordingReportStore) ScheduledForHost(context.Context, *hosts.Host) ([]reports.Report, error) {
	return nil, nil
}

func (s *recordingReportStore) OverwriteSnapshot(
	_ context.Context,
	reportID int64,
	queryHash string,
	hostID int64,
	rows []map[string]string,
	fetchedAt time.Time,
) error {
	s.calls++
	s.reportID = reportID
	s.queryHash = queryHash
	s.hostID = hostID
	s.rows = rows
	s.fetchedAt = fetchedAt
	return nil
}
