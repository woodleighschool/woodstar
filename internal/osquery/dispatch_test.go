package osquery

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
)

func TestQueryNameRoundTrips(t *testing.T) {
	tests := []struct {
		name   string
		kind   queryKind
		suffix string
	}{
		{name: "detail", kind: kindDetail, suffix: "system_info"},
		{name: "label", kind: kindLabel, suffix: "42"},
		{name: "check", kind: kindCheck, suffix: "7"},
		{name: "live", kind: kindLive, suffix: "3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name := queryName(tt.kind, tt.suffix)
			gotKind, gotSuffix, ok := parseQueryName(name)
			if !ok {
				t.Fatalf("parseQueryName(%q) ok = false, want true", name)
			}
			if gotKind != tt.kind || gotSuffix != tt.suffix {
				t.Fatalf("parseQueryName(%q) = %q, %q; want %q, %q", name, gotKind, gotSuffix, tt.kind, tt.suffix)
			}
		})
	}
}

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
	pass.results["required"] = detailResult{rows: []map[string]string{}, status: json.RawMessage(`1`)}
	if sawEveryRequiredDetailQuery(pass) {
		t.Fatal("failed required query was treated as complete")
	}
	pass.results["required"] = detailResult{rows: []map[string]string{}}
	if !sawEveryRequiredDetailQuery(pass) {
		t.Fatal("empty successful required query was not treated as complete")
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

func TestFinalizeDetailPassClearsMissingOrFailedMunkiDetails(t *testing.T) {
	projector := &recordingInventoryProjector{}
	pass := &detailDispatchPass{
		registry: map[string]catalog.DetailQuery{
			"required":                 {Ingest: catalog.IngestHostDetail},
			catalog.QueryMunkiInfo:     {Optional: true, Ingest: catalog.IngestMunkiInfo},
			catalog.QueryMunkiInstalls: {Optional: true, Ingest: catalog.IngestMunkiInstalls},
		},
		results: map[string]detailResult{
			"required":             {},
			catalog.QueryMunkiInfo: {status: json.RawMessage(`1`)},
		},
		allSucceeded: true,
	}

	s := &AgentService{deps: Dependencies{Logger: testLogger(), InventoryProjector: projector}}
	if err := s.finalizeDetailPass(context.Background(), testHost(42), pass); err != nil {
		t.Fatalf("finalize detail pass: %v", err)
	}
	if len(projector.cleared) != 2 ||
		projector.cleared[0] != catalog.QueryMunkiInfo ||
		projector.cleared[1] != catalog.QueryMunkiInstalls {
		t.Fatalf("cleared = %#v, want munki_info and munki_installs", projector.cleared)
	}
	if !projector.markedFresh {
		t.Fatal("optional munki clear blocked freshness")
	}
}

func TestFinalizeDetailPassDoesNotClearMunkiOnNonDetailWrite(t *testing.T) {
	projector := &recordingInventoryProjector{}
	pass := &detailDispatchPass{
		registry: map[string]catalog.DetailQuery{
			"required":             {Ingest: catalog.IngestHostDetail},
			catalog.QueryMunkiInfo: {Optional: true, Ingest: catalog.IngestMunkiInfo},
		},
		results:      map[string]detailResult{},
		allSucceeded: true,
	}

	s := &AgentService{deps: Dependencies{Logger: testLogger(), InventoryProjector: projector}}
	if err := s.finalizeDetailPass(context.Background(), testHost(42), pass); err != nil {
		t.Fatalf("finalize detail pass: %v", err)
	}
	if len(projector.cleared) > 0 {
		t.Fatalf("cleared = %#v, want none", projector.cleared)
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
