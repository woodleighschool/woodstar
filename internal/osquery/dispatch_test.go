package osquery

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
)

func TestParseQueryNameRejectsUnknownNames(t *testing.T) {
	for _, name := range []string{
		"system_info",
		"woodstar_label_query_",
		"woodstar_unknown_query_1",
		"fleet_detail_query_system_info",
		// report names belong to /log, not /distributed/write.
		"woodstar_report_query_15",
	} {
		if kind, suffix, ok := parseQueryName(name); ok || kind != "" || suffix != "" {
			t.Fatalf("parseQueryName(%q) = %q, %q, %t; want zero values", name, kind, suffix, ok)
		}
	}
}

func TestSawEveryRequiredDetailQueryRequiresPresenceAndStatus(t *testing.T) {
	registry := map[string]catalog.DetailQuery{
		"required": {},
		"optional": {Optional: true},
	}
	pass := &detailDispatchPass{registry: registry, results: map[string]detailResult{}}
	if sawEveryRequiredDetailQuery(pass) {
		t.Fatal("missing required query was treated as complete")
	}
	pass.results["required"] = detailResult{
		rows:      []map[string]string{},
		status:    json.RawMessage(`1`),
		hasStatus: true,
	}
	if sawEveryRequiredDetailQuery(pass) {
		t.Fatal("failed required query was treated as complete")
	}
	pass.results["required"] = detailResult{rows: []map[string]string{}}
	if sawEveryRequiredDetailQuery(pass) {
		t.Fatal("required query without a status was treated as complete")
	}
	pass.results["required"] = detailResult{
		rows:      []map[string]string{},
		status:    json.RawMessage(`0`),
		hasStatus: true,
	}
	if !sawEveryRequiredDetailQuery(pass) {
		t.Fatal("required query with integer zero status was not treated as complete")
	}
}

func TestRowPresenceResultRequiresIntegerZeroStatus(t *testing.T) {
	rows := []map[string]string{{"present": "1"}}
	tests := []struct {
		name      string
		status    json.RawMessage
		hasStatus bool
		wantOK    bool
		wantMatch bool
	}{
		{name: "missing status", wantOK: false},
		{name: "integer success", status: json.RawMessage(`0`), hasStatus: true, wantOK: true, wantMatch: true},
		{name: "integer failure", status: json.RawMessage(`1`), hasStatus: true, wantOK: false},
		{name: "string zero rejected", status: json.RawMessage(`"0"`), hasStatus: true, wantOK: false},
		{name: "empty string rejected", status: json.RawMessage(`""`), hasStatus: true, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, ok := rowPresenceResult(tt.status, tt.hasStatus, rows)
			if ok != tt.wantOK || matched != tt.wantMatch {
				t.Fatalf("rowPresenceResult() = %v, %v; want %v, %v", matched, ok, tt.wantMatch, tt.wantOK)
			}
		})
	}
}

func TestFinalizeDetailPassPreservesMunkiWithoutSuccessfulObservation(t *testing.T) {
	projector := &recordingInventoryProjector{}
	pass := &detailDispatchPass{
		registry: map[string]catalog.DetailQuery{
			"required":                 {Ingest: catalog.IngestHostDetail},
			catalog.QueryMunkiInfo:     {Optional: true, Ingest: catalog.IngestMunkiInfo},
			catalog.QueryMunkiInstalls: {Optional: true, Ingest: catalog.IngestMunkiInstalls},
		},
		results: map[string]detailResult{
			"required": {
				status:    json.RawMessage(`0`),
				hasStatus: true,
			},
			catalog.QueryMunkiInfo: {status: json.RawMessage(`1`), hasStatus: true},
		},
		allSucceeded: true,
	}

	s := &AgentService{deps: Dependencies{Logger: testLogger(), InventoryProjector: projector}}
	if err := s.finalizeDetailPass(context.Background(), testHost(42), pass); err != nil {
		t.Fatalf("finalize detail pass: %v", err)
	}
	if len(projector.cleared) > 0 {
		t.Fatalf("cleared = %#v, want none", projector.cleared)
	}
	if !projector.markedFresh {
		t.Fatal("optional Munki failure blocked inventory freshness")
	}
}

type recordingInventoryProjector struct {
	cleared     []string
	markedFresh bool
}

func (p *recordingInventoryProjector) IngestDetail(
	_ context.Context,
	_ catalog.DetailQuery,
	name string,
	_ int64,
	rows []map[string]string,
) error {
	if rows == nil {
		p.cleared = append(p.cleared, name)
	}
	return nil
}

func (p *recordingInventoryProjector) IngestSoftware(
	context.Context,
	int64,
	map[string][]map[string]string,
) error {
	return nil
}

func (p *recordingInventoryProjector) MarkFresh(context.Context, int64) error {
	p.markedFresh = true
	return nil
}

func testHost(id int64) *hosts.Host {
	return &hosts.Host{ID: id}
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
