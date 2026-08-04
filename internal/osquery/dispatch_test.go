package osquery

import (
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
)

func TestParseQueryNameRejectsUnknownNames(t *testing.T) {
	for _, name := range []string{
		"system_info",
		"woodstar_label_query_",
		"woodstar_unknown_query_1",
		"fleet_detail_query_system_info",
		// report names belong to /log, not /distributed/write.
		"woodstar_report_query_15_2",
	} {
		if kind, suffix, ok := parseQueryName(name); ok || kind != "" || suffix != "" {
			t.Fatalf("parseQueryName(%q) = %q, %q, %t; want zero values", name, kind, suffix, ok)
		}
	}
}

func TestHashedQueryNameRoundTrip(t *testing.T) {
	sql := "select 1;"
	name := queryNameForSQL(kindCheck, 15, sql)
	kind, suffix, ok := parseQueryName(name)
	if !ok || kind != kindCheck {
		t.Fatalf("parseQueryName(%q) = %q, %q, %t", name, kind, suffix, ok)
	}
	id, hash, ok := parseQueryIdentity(suffix)
	if !ok || id != 15 || hash != queryHash(sql) {
		t.Fatalf(
			"parseQueryIdentity(%q) = %d, %q, %t; want 15, %q, true",
			suffix,
			id,
			hash,
			ok,
			queryHash(sql),
		)
	}

	for _, invalid := range []string{
		"15",
		"15_short",
		"0_" + queryHash(sql),
		"15_" + strings.Repeat("g", 64),
		"15_" + queryHash(sql) + "_extra",
	} {
		if id, hash, ok := parseQueryIdentity(invalid); ok || id != 0 || hash != "" {
			t.Fatalf(
				"parseQueryIdentity(%q) = %d, %q, %t; want zero values",
				invalid,
				id,
				hash,
				ok,
			)
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

func TestHandleLiveResultReplacesHostSnapshot(t *testing.T) {
	tests := []struct {
		name       string
		status     json.RawMessage
		hasStatus  bool
		message    string
		wantStatus livequery.Status
	}{
		{
			name:       "collected",
			status:     json.RawMessage(`0`),
			hasStatus:  true,
			wantStatus: livequery.StatusCollected,
		},
		{
			name:       "error",
			status:     json.RawMessage(`1`),
			hasStatus:  true,
			message:    "query failed",
			wantStatus: livequery.StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := livequery.NewManager()
			handle := manager.Start("select 1", []livequery.Target{
				{HostID: 42, HostName: "original-name"},
			})
			service := &AgentService{deps: Dependencies{LiveQueries: manager}}
			rows := []map[string]string{{"answer": "42"}}

			service.handleLiveResult(
				&hosts.Host{ID: 42, DisplayName: "current-name"},
				strconv.FormatInt(handle.ID, 10),
				rows,
				tt.status,
				tt.hasStatus,
				tt.message,
			)

			snapshots, release, err := manager.Subscribe(handle.ID)
			if err != nil {
				t.Fatalf("Subscribe: %v", err)
			}
			defer release()
			snapshot := <-snapshots
			if snapshot.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", snapshot.Status, tt.wantStatus)
			}
			if snapshot.HostName != "current-name" {
				t.Fatalf("host name = %q, want current-name", snapshot.HostName)
			}
			if !reflect.DeepEqual(snapshot.Rows, rows) {
				t.Fatalf("rows = %#v, want %#v", snapshot.Rows, rows)
			}
			if snapshot.Error != tt.message {
				t.Fatalf("error = %q, want %q", snapshot.Error, tt.message)
			}
		})
	}
}

func TestFinalizeDetailPassDeliversFailedMunkiEnvelopeWithoutBlockingInventory(t *testing.T) {
	projector := &recordingInventoryProjector{}
	munkiIngestor := &recordingMunkiEnvelopeIngestor{}
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

	s := &AgentService{deps: Dependencies{Logger: testLogger(), InventoryProjector: projector, MunkiEnvelopeIngestor: munkiIngestor}}
	if err := s.finalizeDetailPass(context.Background(), testHost(42), pass); err != nil {
		t.Fatalf("finalize detail pass: %v", err)
	}
	if munkiIngestor.calls != 1 || munkiIngestor.envelope.Info.Status != 1 || !munkiIngestor.envelope.Info.Present || munkiIngestor.envelope.Installs.Present {
		t.Fatalf("Munki envelope = %+v calls=%d, want one failed/missing family envelope", munkiIngestor.envelope, munkiIngestor.calls)
	}
	if !projector.markedFresh {
		t.Fatal("optional Munki failure blocked inventory freshness")
	}
}

func TestDetailPassAccumulatesMunkiEnvelopeUntilFinalization(t *testing.T) {
	projector := &recordingInventoryProjector{}
	munkiIngestor := &recordingMunkiEnvelopeIngestor{}
	pass := &detailDispatchPass{
		registry: map[string]catalog.DetailQuery{
			catalog.QueryMunkiInfo:     {Optional: true, Ingest: catalog.IngestMunkiInfo},
			catalog.QueryMunkiInstalls: {Optional: true, Ingest: catalog.IngestMunkiInstalls},
		},
		results:      map[string]detailResult{},
		allSucceeded: true,
	}
	s := &AgentService{deps: Dependencies{Logger: testLogger(), InventoryProjector: projector, MunkiEnvelopeIngestor: munkiIngestor}}
	infoRows := []map[string]string{{"version": "7.1"}}
	if err := s.handleDetailResult(context.Background(), 42, catalog.QueryMunkiInfo, infoRows, json.RawMessage(`0`), true, "info ok", pass); err != nil {
		t.Fatalf("handle info: %v", err)
	}
	if munkiIngestor.calls != 0 {
		t.Fatalf("Munki calls after first family result = %d, want 0", munkiIngestor.calls)
	}
	installsRows := []map[string]string{{"name": "Firefox"}}
	if err := s.handleDetailResult(context.Background(), 42, catalog.QueryMunkiInstalls, installsRows, json.RawMessage(`0`), true, "installs ok", pass); err != nil {
		t.Fatalf("handle installs: %v", err)
	}
	if munkiIngestor.calls != 0 {
		t.Fatalf("Munki calls before finalization = %d, want 0", munkiIngestor.calls)
	}
	if err := s.finalizeDetailPass(context.Background(), testHost(42), pass); err != nil {
		t.Fatalf("finalize detail pass: %v", err)
	}
	if munkiIngestor.calls != 1 || !munkiIngestor.envelope.Info.Present || !munkiIngestor.envelope.Installs.Present ||
		munkiIngestor.envelope.Info.Message != "info ok" || munkiIngestor.envelope.Installs.Message != "installs ok" ||
		!reflect.DeepEqual(munkiIngestor.envelope.Info.Rows, infoRows) || !reflect.DeepEqual(munkiIngestor.envelope.Installs.Rows, installsRows) {
		t.Fatalf("Munki envelope = %+v calls=%d, want one complete retained envelope", munkiIngestor.envelope, munkiIngestor.calls)
	}
}

func TestFinalizeDetailPassSkipsAbsentMunkiFamily(t *testing.T) {
	projector := &recordingInventoryProjector{}
	munkiIngestor := &recordingMunkiEnvelopeIngestor{}
	pass := &detailDispatchPass{
		registry:     map[string]catalog.DetailQuery{"required": {Ingest: catalog.IngestHostDetail}},
		results:      map[string]detailResult{"required": {status: json.RawMessage(`0`), hasStatus: true}},
		allSucceeded: true,
	}
	s := &AgentService{deps: Dependencies{Logger: testLogger(), InventoryProjector: projector, MunkiEnvelopeIngestor: munkiIngestor}}
	if err := s.finalizeDetailPass(context.Background(), testHost(42), pass); err != nil {
		t.Fatalf("finalize detail pass: %v", err)
	}
	if munkiIngestor.calls != 0 {
		t.Fatalf("Munki calls = %d, want none", munkiIngestor.calls)
	}
}

func TestDetailQueryCadenceRemainsHourly(t *testing.T) {
	lastUpdated := time.Now().Add(-59 * time.Minute)
	if due := catalog.DetailQueriesDue(&lastUpdated, catalog.DetailQueryHash()); len(due.Queries) != 0 {
		t.Fatalf("detail queries due after 59 minutes = %#v, want none", due.Queries)
	}
	lastUpdated = time.Now().Add(-61 * time.Minute)
	if due := catalog.DetailQueriesDue(&lastUpdated, catalog.DetailQueryHash()); len(due.Queries) == 0 {
		t.Fatal("detail queries were not due after 61 minutes")
	}
}

type recordingInventoryProjector struct {
	markedFresh bool
}

func (p *recordingInventoryProjector) IngestDetail(
	_ context.Context,
	_ catalog.DetailQuery,
	_ string,
	_ int64,
	_ []map[string]string,
) error {
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

type recordingMunkiEnvelopeIngestor struct {
	calls    int
	envelope munki.EnvelopeInput
}

func (i *recordingMunkiEnvelopeIngestor) IngestEnvelope(_ context.Context, _ int64, envelope munki.EnvelopeInput) error {
	i.calls++
	i.envelope = envelope
	return nil
}

func testHost(id int64) *hosts.Host {
	return &hosts.Host{ID: id}
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
