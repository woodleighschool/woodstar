package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
)

func TestSantaEventsListFiltersAndPaginates(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	santaStore := santa.NewStore(db)
	eventsStore := events.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-events-wire-host"},
		OrbitNodeKey: "santa-events-wire-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := santaStore.UpsertHostObservation(ctx, santa.HostObservation{
		HostID:             host.ID,
		MachineID:          "santa-events-wire-host",
		SerialNumber:       "WIRE",
		ClientModeReported: configurations.ReportedClientModeMonitor,
	}); err != nil {
		t.Fatalf("upsert observation: %v", err)
	}
	occurredAt := time.Date(2026, 5, 23, 14, 0, 0, 0, time.UTC)
	if _, err := eventsStore.IngestEvents(ctx, host.ID, []events.ExecutionEventInput{
		{
			FileSHA256:    "wire-blocked-1",
			FileName:      "Blocked One",
			ExecutingUser: "alice",
			OccurredAt:    occurredAt,
			Decision:      events.ExecutionDecisionBlockBinary,
		},
		{
			FileSHA256:    "wire-blocked-2",
			FileName:      "Blocked Two",
			ExecutingUser: "root",
			OccurredAt:    occurredAt.Add(time.Second),
			Decision:      events.ExecutionDecisionBlockCertificate,
		},
		{
			FileSHA256:    "wire-allowed",
			FileName:      "Allowed",
			ExecutingUser: "alice",
			OccurredAt:    occurredAt.Add(2 * time.Second),
			Decision:      events.ExecutionDecisionAllowBinary,
		},
	}, []events.FileAccessEventInput{
		{
			RuleVersion: "wire-v1",
			RuleName:    "Protect Wire Payroll",
			Target:      "/Users/alice/WirePayroll.csv",
			Decision:    events.FileAccessDecisionDenied,
			OccurredAt:  occurredAt.Add(3 * time.Second),
			ProcessChain: []events.ProcessInput{{
				PID:        42,
				FilePath:   "/Applications/Wire.app/Contents/MacOS/Wire",
				FileSHA256: "wire-process",
			}},
		},
		{
			RuleVersion: "wire-v1",
			RuleName:    "Audit Downloads",
			Target:      "/Users/alice/Downloads/audit.txt",
			Decision:    events.FileAccessDecisionAuditOnly,
			OccurredAt:  occurredAt.Add(4 * time.Second),
		},
	}, nil); err != nil {
		t.Fatalf("ingest events: %v", err)
	}

	router := chi.NewRouter()
	api := humachi.New(router, huma.DefaultConfig("test", "test"))
	registerSantaEvents(api, eventsStore, discardLogger())

	rec := santaEventsRequest(t, router, "/api/santa/events?decisions=blocked&per_page=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Blocked") {
		t.Fatalf("body = %q, want a blocked event", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "Allowed") {
		t.Fatalf("body = %q, decisions=blocked filter did not exclude allowed events", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"count":2`) {
		t.Fatalf("body = %q, want count=2 for normal pagination", rec.Body.String())
	}
	var blockedList Page[events.ExecutionEvent]
	if err := json.Unmarshal(rec.Body.Bytes(), &blockedList); err != nil {
		t.Fatalf("decode blocked execution list: %v", err)
	}
	if blockedList.Count != 2 || len(blockedList.Items) != 1 {
		t.Fatalf("blocked execution list = %+v count=%d, want one of two", blockedList.Items, blockedList.Count)
	}

	rec = santaEventsRequest(t, router, "/api/santa/events?q=Allowed&decisions=allowed")
	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Allowed") || strings.Contains(rec.Body.String(), "Blocked") {
		t.Fatalf("search response = %q, want only allowed event", rec.Body.String())
	}

	rec = santaEventsRequest(t, router, "/api/santa/events?user=alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("user filter status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var executionList Page[events.ExecutionEvent]
	if err := json.Unmarshal(rec.Body.Bytes(), &executionList); err != nil {
		t.Fatalf("decode execution list: %v", err)
	}
	if executionList.Count != 2 || len(executionList.Items) != 2 {
		t.Fatalf("execution list = %+v count=%d, want two alice events", executionList.Items, executionList.Count)
	}
	for _, event := range executionList.Items {
		if event.ExecutingUser != "alice" {
			t.Fatalf("execution event user = %q, want alice", event.ExecutingUser)
		}
	}

	rec = santaEventsRequest(t, router, "/api/santa/file-access-events?decisions=denied")
	if rec.Code != http.StatusOK {
		t.Fatalf("file access status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var fileAccessList Page[events.FileAccessEvent]
	if err := json.Unmarshal(rec.Body.Bytes(), &fileAccessList); err != nil {
		t.Fatalf("decode file access list: %v", err)
	}
	if fileAccessList.Count != 1 ||
		len(fileAccessList.Items) != 1 ||
		fileAccessList.Items[0].Decision != events.FileAccessDecisionDenied ||
		fileAccessList.Items[0].Target != "/Users/alice/WirePayroll.csv" {
		t.Fatalf("file access list = %+v, want one denied payroll event", fileAccessList)
	}

	rec = santaEventsRequest(
		t,
		router,
		fmt.Sprintf("/api/santa/file-access-events/%d", fileAccessList.Items[0].ID),
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("file access detail status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var fileAccessDetail events.FileAccessEvent
	if err := json.Unmarshal(rec.Body.Bytes(), &fileAccessDetail); err != nil {
		t.Fatalf("decode file access detail: %v", err)
	}
	if len(fileAccessDetail.ProcessChain) != 1 || fileAccessDetail.ProcessChain[0].FileName != "Wire" {
		t.Fatalf("file access detail = %+v, want persisted process chain", fileAccessDetail)
	}
}

func santaEventsRequest(t *testing.T, router *chi.Mux, path string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	router.ServeHTTP(rec, req)
	return rec
}
