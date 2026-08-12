package osquery

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
)

func TestQueryFailureLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		invoke    func(t *testing.T, service *AgentService)
		wantLevel string
	}{
		{
			name: "label",
			invoke: func(t *testing.T, service *AgentService) {
				t.Helper()
				service.handleLabelResult(
					t.Context(), 42, "5", nil, json.RawMessage(`1`), true,
					"distributed query is denylisted", &labelDispatchPass{},
				)
			},
			wantLevel: "DEBUG",
		},
		{
			name: "policy",
			invoke: func(t *testing.T, service *AgentService) {
				t.Helper()
				suffix := "10_" + queryHash("SELECT 1") + "_1_1"
				if err := service.handlePolicyResult(
					t.Context(), 42, suffix, nil, json.RawMessage(`1`), true,
					"distributed query is denylisted",
				); err != nil {
					t.Fatalf("handle policy result: %v", err)
				}
			},
			wantLevel: "DEBUG",
		},
		{
			name: "optional detail",
			invoke: func(t *testing.T, service *AgentService) {
				t.Helper()
				pass := &detailDispatchPass{
					registry: map[string]catalog.DetailQuery{"optional": {Optional: true}},
					results:  map[string]detailResult{},
				}
				if err := service.handleDetailResult(
					t.Context(), 42, "optional", nil, json.RawMessage(`1`), true,
					"table unavailable", pass,
				); err != nil {
					t.Fatalf("handle detail result: %v", err)
				}
			},
			wantLevel: "DEBUG",
		},
		{
			name: "required detail",
			invoke: func(t *testing.T, service *AgentService) {
				t.Helper()
				pass := &detailDispatchPass{
					registry: map[string]catalog.DetailQuery{"required": {}},
					results:  map[string]detailResult{},
				}
				if err := service.handleDetailResult(
					t.Context(), 42, "required", nil, json.RawMessage(`1`), true,
					"table unavailable", pass,
				); err != nil {
					t.Fatalf("handle detail result: %v", err)
				}
			},
			wantLevel: "WARN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(
				&output,
				&slog.HandlerOptions{Level: slog.LevelDebug},
			))
			service := &AgentService{deps: Dependencies{
				Logger:      logger,
				PolicyStore: fakePolicyStore{},
			}}

			tt.invoke(t, service)

			var record struct {
				Level string `json:"level"`
			}
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatalf("decode log record: %v; output: %s", err, output.String())
			}
			if record.Level != tt.wantLevel {
				t.Fatalf("log level = %q, want %q; output: %s", record.Level, tt.wantLevel, output.String())
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
		"woodstar_report_query_15_2",
	} {
		if kind, suffix, ok := parseQueryName(name); ok || kind != "" || suffix != "" {
			t.Fatalf("parseQueryName(%q) = %q, %q, %t; want zero values", name, kind, suffix, ok)
		}
	}
}

func TestHashedQueryNameRoundTrip(t *testing.T) {
	sql := "select 1;"
	name := queryNameForSQL(kindPolicy, 15, sql)
	kind, suffix, ok := parseQueryName(name)
	if !ok || kind != kindPolicy {
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

func TestPolicyEvaluationNameRoundTrip(t *testing.T) {
	evaluation := policies.Evaluation{
		PolicyID: 15,
		Query:    "select 1;",
		Revision: 3,
		Sequence: 8,
	}
	name := queryNameForEvaluation(kindPolicy, evaluation)
	kind, suffix, ok := parseQueryName(name)
	if !ok || kind != kindPolicy {
		t.Fatalf("parseQueryName(%q) = %q, %q, %t", name, kind, suffix, ok)
	}
	policyID, hash, revision, sequence, ok := parsePolicyEvaluationIdentity(suffix)
	if !ok || policyID != 15 || hash != queryHash(evaluation.Query) || revision != 3 || sequence != 8 {
		t.Fatalf(
			"parsePolicyEvaluationIdentity(%q) = %d, %q, %d, %d, %t",
			suffix,
			policyID,
			hash,
			revision,
			sequence,
			ok,
		)
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
			recorder := &recordingLiveQueries{}
			service := &AgentService{deps: Dependencies{LiveQueries: recorder}}
			rows := []map[string]string{{"answer": "42"}}

			if err := service.handleLiveResult(
				context.Background(),
				&hosts.Host{ID: 42, DisplayName: "current-name"},
				strconv.FormatInt(7, 10),
				rows,
				tt.status,
				tt.hasStatus,
				tt.message,
			); err != nil {
				t.Fatalf("handleLiveResult: %v", err)
			}
			result := recorder.result
			if result.QueryID != 7 || result.HostID != 42 || result.Status != tt.wantStatus {
				t.Fatalf("result = %+v, want query 7 host 42 status %q", result, tt.wantStatus)
			}
			if result.HostName != "current-name" {
				t.Fatalf("host name = %q, want current-name", result.HostName)
			}
			if !reflect.DeepEqual(result.Rows, rows) {
				t.Fatalf("rows = %#v, want %#v", result.Rows, rows)
			}
			if result.Error != tt.message {
				t.Fatalf("error = %q, want %q", result.Error, tt.message)
			}
		})
	}
}

type recordingLiveQueries struct {
	result livequery.Result
}

func (r *recordingLiveQueries) PendingForHost(context.Context, int64) ([]livequery.Work, error) {
	return nil, nil
}

func (r *recordingLiveQueries) RecordResult(_ context.Context, result livequery.Result) error {
	r.result = result
	return nil
}

func TestFinalizeDetailPassSendsMunkiQueryFamilyTogether(t *testing.T) {
	registry := map[string]catalog.DetailQuery{
		"required":                 {Ingest: catalog.IngestHostDetail},
		catalog.QueryMunkiInfo:     {Optional: true, Ingest: catalog.IngestMunkiInfo},
		catalog.QueryMunkiInstalls: {Optional: true, Ingest: catalog.IngestMunkiInstalls},
	}

	t.Run("partial family", func(t *testing.T) {
		projector := &recordingInventoryProjector{}
		collector := &recordingMunkiCollector{}
		pass := &detailDispatchPass{
			registry: registry,
			results: map[string]detailResult{
				"required": {
					status:    json.RawMessage(`0`),
					hasStatus: true,
				},
				catalog.QueryMunkiInfo: {
					status:    json.RawMessage(`1`),
					hasStatus: true,
				},
			},
			allSucceeded: true,
		}

		s := &AgentService{deps: Dependencies{
			Logger:             testLogger(),
			InventoryProjector: projector,
			MunkiCollector:     collector,
		}}
		if err := s.finalizeDetailPass(context.Background(), testHost(42), pass); err != nil {
			t.Fatalf("finalize detail pass: %v", err)
		}
		if !projector.markedFresh {
			t.Fatal("optional Munki failure blocked inventory freshness")
		}
		if len(collector.collections) != 1 {
			t.Fatalf("collections = %d, want one", len(collector.collections))
		}
		collection := collector.collections[0]
		if !collection.Info.Present || collection.Info.Successful {
			t.Fatalf("info result = %+v, want failed result", collection.Info)
		}
		if collection.Installs.Present {
			t.Fatalf("installs result = %+v, want missing result", collection.Installs)
		}
	})

	t.Run("queries unavailable", func(t *testing.T) {
		collector := &recordingMunkiCollector{}
		pass := &detailDispatchPass{
			registry: registry,
			results: map[string]detailResult{
				"required": {status: json.RawMessage(`0`), hasStatus: true},
			},
			allSucceeded: true,
		}
		s := &AgentService{deps: Dependencies{
			Logger:             testLogger(),
			InventoryProjector: &recordingInventoryProjector{},
			MunkiCollector:     collector,
		}}
		if err := s.finalizeDetailPass(context.Background(), testHost(42), pass); err != nil {
			t.Fatalf("finalize detail pass: %v", err)
		}
		if len(collector.collections) != 1 {
			t.Fatalf("collections = %d, want one", len(collector.collections))
		}
		collection := collector.collections[0]
		if collection.Info.Present || collection.Installs.Present {
			t.Fatalf("collection = %+v, want authoritative no-report input", collection)
		}
	})

	t.Run("no detail results", func(t *testing.T) {
		collector := &recordingMunkiCollector{}
		pass := &detailDispatchPass{registry: registry, results: map[string]detailResult{}, allSucceeded: true}
		s := &AgentService{deps: Dependencies{
			Logger:             testLogger(),
			InventoryProjector: &recordingInventoryProjector{},
			MunkiCollector:     collector,
		}}
		if err := s.finalizeDetailPass(context.Background(), testHost(42), pass); err != nil {
			t.Fatalf("finalize detail pass: %v", err)
		}
		if len(collector.collections) != 0 {
			t.Fatalf("collections = %d, want none", len(collector.collections))
		}
	})
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

type recordingMunkiCollector struct {
	collections []munki.Collection
}

func (c *recordingMunkiCollector) IngestCollection(
	_ context.Context,
	_ int64,
	collection munki.Collection,
) error {
	c.collections = append(c.collections, collection)
	return nil
}

func testHost(id int64) *hosts.Host {
	return &hosts.Host{ID: id}
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
