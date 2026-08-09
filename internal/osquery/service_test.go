package osquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

func TestEnrollRequiresHardwareIdentity(t *testing.T) {
	t.Parallel()

	hostStore := &fakeHostStore{host: &hosts.Host{ID: 42}}
	service := NewAgentService(Dependencies{
		HostStore:   hostStore,
		SecretStore: fakeSecretVerifier{ok: true},
		Logger:      slog.New(slog.DiscardHandler),
	})
	_, err := service.Enroll(t.Context(), EnrollRequest{EnrollSecret: "enroll-secret"}, heartbeats.Contact{})
	if !errors.Is(err, enrollment.ErrMissingHardwareUUID) {
		t.Fatalf("Enroll error = %v, want ErrMissingHardwareUUID", err)
	}
	if hostStore.upsertCalled {
		t.Fatal("Enroll attempted to persist a host without hardware identity")
	}
}

func TestEnrollUsesHostIdentifierAsHardwareUUID(t *testing.T) {
	t.Parallel()

	hostStore := &fakeHostStore{host: &hosts.Host{ID: 42}}
	service := NewAgentService(Dependencies{
		HostStore:   hostStore,
		SecretStore: fakeSecretVerifier{ok: true},
		Heartbeats:  &fakeOsqueryHeartbeatRecorder{},
		Logger:      slog.New(slog.DiscardHandler),
	})
	nodeKey, err := service.Enroll(t.Context(), EnrollRequest{
		EnrollSecret:   "enroll-secret",
		HostIdentifier: "host-identifier",
	}, heartbeats.Contact{})
	if err != nil {
		t.Fatalf("Enroll: %v", err)
	}
	if nodeKey == "" || hostStore.update.OsqueryNodeKey != nodeKey {
		t.Fatalf("node key = %q, persisted node key = %q; want same non-empty value", nodeKey, hostStore.update.OsqueryNodeKey)
	}
	if hostStore.update.Hardware.UUID != "host-identifier" {
		t.Fatalf("hardware UUID = %q, want host identifier fallback", hostStore.update.Hardware.UUID)
	}
}

func TestLogPropagatesReportPersistenceFailure(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service := NewAgentService(Dependencies{
		HostStore:  &fakeHostStore{host: &hosts.Host{ID: 42}},
		Heartbeats: &fakeOsqueryHeartbeatRecorder{},
		ReportStore: fakeReportStore{
			overwriteErr: wantErr,
		},
		Logger: slog.New(slog.DiscardHandler),
	})

	_, err := service.Log(context.Background(), "node-key", heartbeats.Contact{}, LogRequest{
		LogType: "result",
		Data: json.RawMessage(fmt.Sprintf(`{
			"name":%q,
			"unixTime":1778848496,
			"action":"snapshot",
			"snapshot":[]
		}`, queryNameForSQL(kindReport, 7, "select 1;"))),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Log error = %v, want %v", err, wantErr)
	}
}

func TestAgentServiceRecordsAuthenticatedHostContact(t *testing.T) {
	t.Parallel()

	contact := heartbeats.Contact{RemoteIP: "203.0.113.42", UserAgent: "osquery/5.12.1"}
	tests := []struct {
		name string
		call func(t *testing.T, service *AgentService)
	}{
		{
			name: "enroll",
			call: func(t *testing.T, service *AgentService) {
				t.Helper()
				if _, err := service.Enroll(t.Context(), EnrollRequest{EnrollSecret: "secret", HostIdentifier: "hardware-uuid"}, contact); err != nil {
					t.Fatalf("Enroll: %v", err)
				}
			},
		},
		{
			name: "config",
			call: func(t *testing.T, service *AgentService) {
				t.Helper()
				if _, err := service.Config(t.Context(), "node-key", contact); err != nil {
					t.Fatalf("Config: %v", err)
				}
			},
		},
		{
			name: "distributed read",
			call: func(t *testing.T, service *AgentService) {
				t.Helper()
				if _, err := service.DistributedRead(t.Context(), "node-key", contact); err != nil {
					t.Fatalf("DistributedRead: %v", err)
				}
			},
		},
		{
			name: "distributed write",
			call: func(t *testing.T, service *AgentService) {
				t.Helper()
				if _, err := service.DistributedWrite(t.Context(), DistributedWriteRequest{NodeKey: "node-key"}, contact); err != nil {
					t.Fatalf("DistributedWrite: %v", err)
				}
			},
		},
		{
			name: "log",
			call: func(t *testing.T, service *AgentService) {
				t.Helper()
				if _, err := service.Log(t.Context(), "node-key", contact, LogRequest{}); err != nil {
					t.Fatalf("Log: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			recorder := &fakeOsqueryHeartbeatRecorder{}
			hostStore := &fakeHostStore{host: &hosts.Host{ID: 42}}
			service := newTestAgentService(recorder, hostStore)

			tt.call(t, service)

			if len(recorder.calls) != 1 {
				t.Fatalf("Record calls = %d, want 1", len(recorder.calls))
			}
			got := recorder.calls[0]
			if got.hostID != 42 || got.source != heartbeats.SourceOsquery || got.contact != contact {
				t.Fatalf("Record call = %#v, want host 42, osquery, %#v", got, contact)
			}
		})
	}
}

func TestAgentServiceDoesNotRecordUnauthenticatedContact(t *testing.T) {
	t.Parallel()

	recorder := &fakeOsqueryHeartbeatRecorder{}
	hostStore := &fakeHostStore{host: &hosts.Host{ID: 42}}
	service := newTestAgentService(recorder, hostStore)
	service.deps.SecretStore = fakeSecretVerifier{ok: false}
	_, err := service.Enroll(t.Context(), EnrollRequest{EnrollSecret: "bad", HostIdentifier: "hardware-uuid"}, heartbeats.Contact{})
	if !errors.Is(err, agentauth.ErrInvalidSecret) {
		t.Fatalf("Enroll error = %v, want invalid secret", err)
	}
	if len(recorder.calls) != 0 {
		t.Fatalf("Record calls = %d, want 0", len(recorder.calls))
	}

	hostStore = &fakeHostStore{host: &hosts.Host{ID: 42}}
	service = newTestAgentService(recorder, hostStore)
	hostStore.getErr = fault.ErrNotFound
	resp, err := service.Config(t.Context(), "bad", heartbeats.Contact{})
	if err != nil || !resp.NodeInvalid {
		t.Fatalf("Config = %#v, %v; want invalid node without error", resp, err)
	}
	if len(recorder.calls) != 0 {
		t.Fatalf("Record calls = %d, want 0", len(recorder.calls))
	}
}

func TestAgentServicePropagatesHeartbeatError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("heartbeat unavailable")
	hostStore := &fakeHostStore{host: &hosts.Host{ID: 42}}
	service := newTestAgentService(&fakeOsqueryHeartbeatRecorder{err: wantErr}, hostStore)
	_, err := service.Config(t.Context(), "node-key", heartbeats.Contact{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Config error = %v, want %v", err, wantErr)
	}
}

func TestDistributedReadQueuesFreshInventoryWhenRefreshRequested(t *testing.T) {
	t.Parallel()

	now := time.Now()
	hostStore := &fakeHostStore{host: &hosts.Host{
		ID:                        42,
		InventoryUpdatedAt:        &now,
		InventoryQueryHash:        catalog.DetailQueryHash(),
		InventoryRefreshRequested: true,
	}}
	service := newTestAgentService(&fakeOsqueryHeartbeatRecorder{}, hostStore)

	response, err := service.DistributedRead(t.Context(), "node-key", heartbeats.Contact{})
	if err != nil {
		t.Fatalf("DistributedRead: %v", err)
	}
	want := catalog.DetailQueriesDue(nil, "")
	if len(response.Queries) != len(want.Queries) {
		t.Fatalf("query count = %d, want %d requested detail queries", len(response.Queries), len(want.Queries))
	}
	if len(response.Discovery) != len(want.Discovery) {
		t.Fatalf("discovery count = %d, want %d", len(response.Discovery), len(want.Discovery))
	}
}

type fakeHostStore struct {
	host         *hosts.Host
	update       hosts.InventoryUpdate
	upsertCalled bool
	getErr       error
}

func (s *fakeHostStore) UpsertOnOsqueryEnroll(
	_ context.Context,
	update hosts.InventoryUpdate,
) (*hosts.Host, error) {
	s.update = update
	s.upsertCalled = true
	return s.host, nil
}

type fakeSecretVerifier struct {
	ok bool
}

func (v fakeSecretVerifier) Verify(context.Context, agentauth.Agent, string) (bool, error) {
	return v.ok, nil
}

func (s *fakeHostStore) GetByOsqueryNodeKey(context.Context, string) (*hosts.Host, error) {
	return s.host, s.getErr
}

type fakeOsqueryHeartbeatCall struct {
	hostID  int64
	source  heartbeats.Source
	contact heartbeats.Contact
}

type fakeOsqueryHeartbeatRecorder struct {
	calls []fakeOsqueryHeartbeatCall
	err   error
}

func (r *fakeOsqueryHeartbeatRecorder) Record(_ context.Context, hostID int64, source heartbeats.Source, contact heartbeats.Contact) error {
	r.calls = append(r.calls, fakeOsqueryHeartbeatCall{hostID: hostID, source: source, contact: contact})
	return r.err
}

func newTestAgentService(recorder heartbeatRecorder, hostStore *fakeHostStore) *AgentService {
	return NewAgentService(Dependencies{
		HostStore:      hostStore,
		ReportStore:    fakeReportStore{},
		CheckStore:     fakeCheckStore{},
		LabelEvaluator: fakeLabelEvaluator{},
		LiveQueries:    fakeLiveQueries{},
		SecretStore:    fakeSecretVerifier{ok: true},
		Heartbeats:     recorder,
		Logger:         slog.New(slog.DiscardHandler),
	})
}

func (s *fakeHostStore) ApplyInventory(
	context.Context,
	int64,
	hosts.InventoryUpdate,
) error {
	return nil
}

type fakeReportStore struct {
	overwriteErr error
}

type fakeLabelEvaluator struct{}

func (fakeLabelEvaluator) ApplicableLabels(context.Context) ([]labels.DynamicLabel, error) {
	return nil, nil
}

func (fakeLabelEvaluator) Finalize(context.Context, *hosts.Host, []ingest.LabelResult) error {
	return nil
}

type fakeCheckStore struct{}

func (fakeCheckStore) ApplicableForHost(context.Context, *hosts.Host) ([]checks.Check, error) {
	return nil, nil
}

func (fakeCheckStore) UpsertMembership(context.Context, int64, string, int64, *bool) error {
	return nil
}

type fakeLiveQueries struct{}

func (fakeLiveQueries) PendingForHost(int64) []livequery.Work { return nil }

func (fakeLiveQueries) RecordResult(livequery.Result) {}

func (fakeReportStore) ScheduledForHost(context.Context, *hosts.Host) ([]reports.Report, error) {
	return nil, nil
}

func (s fakeReportStore) OverwriteSnapshot(
	context.Context,
	int64,
	string,
	int64,
	[]map[string]string,
	time.Time,
) error {
	return s.overwriteErr
}
