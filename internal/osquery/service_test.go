package osquery

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/hosts"
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
	_, err := service.Enroll(t.Context(), EnrollRequest{EnrollSecret: "enroll-secret"})
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
		Logger:      slog.New(slog.DiscardHandler),
	})
	nodeKey, err := service.Enroll(t.Context(), EnrollRequest{
		EnrollSecret:   "enroll-secret",
		HostIdentifier: "host-identifier",
	})
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
		HostStore: &fakeHostStore{host: &hosts.Host{ID: 42}},
		ReportStore: fakeReportStore{
			overwriteErr: wantErr,
		},
		Logger: slog.New(slog.DiscardHandler),
	})

	_, err := service.Log(context.Background(), "node-key", "", LogRequest{
		LogType: "result",
		Data: json.RawMessage(`{
			"name":"woodstar_report_query_7",
			"unixTime":1778848496,
			"action":"snapshot",
			"snapshot":[]
		}`),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Log error = %v, want %v", err, wantErr)
	}
}

type fakeHostStore struct {
	host         *hosts.Host
	update       hosts.InventoryUpdate
	upsertCalled bool
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
	return s.host, nil
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

func (fakeReportStore) ScheduledForHost(context.Context, *hosts.Host) ([]reports.Report, error) {
	return nil, nil
}

func (s fakeReportStore) OverwriteResults(
	context.Context,
	int64,
	int64,
	[]map[string]string,
	time.Time,
) error {
	return s.overwriteErr
}
