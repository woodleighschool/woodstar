package osquery

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/netip"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

func TestConfigRecordsPublicIPOnlyWhenChanged(t *testing.T) {
	ctx := context.Background()
	hostStore := &fakeHostStore{
		host: &hosts.Host{
			ID: 42,
			Network: hosts.HostNetwork{
				LastRemoteIP: addrPtr("192.0.2.10"),
			},
		},
	}
	service := NewAgentService(Dependencies{
		HostStore:   hostStore,
		ReportStore: fakeReportStore{},
		Logger:      slog.New(slog.DiscardHandler),
	})

	if _, err := service.Config(ctx, "node-key", "192.0.2.10"); err != nil {
		t.Fatalf("Config() same IP error = %v", err)
	}
	if hostStore.applyCount != 0 {
		t.Fatalf("ApplyInventory calls after same IP = %d, want 0", hostStore.applyCount)
	}

	if _, err := service.Config(ctx, "node-key", "192.0.2.11"); err != nil {
		t.Fatalf("Config() changed IP error = %v", err)
	}
	if hostStore.applyCount != 1 {
		t.Fatalf("ApplyInventory calls after changed IP = %d, want 1", hostStore.applyCount)
	}
	if hostStore.lastUpdate.Network.LastRemoteIP != "192.0.2.11" {
		t.Fatalf("last remote IP update = %q, want 192.0.2.11", hostStore.lastUpdate.Network.LastRemoteIP)
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
	host       *hosts.Host
	applyCount int
	lastUpdate hosts.InventoryUpdate
}

func (s *fakeHostStore) UpsertOnOsqueryEnroll(
	context.Context,
	hosts.InventoryUpdate,
) (*hosts.Host, error) {
	return s.host, nil
}

func (s *fakeHostStore) GetByOsqueryNodeKey(context.Context, string) (*hosts.Host, error) {
	return s.host, nil
}

func (s *fakeHostStore) ApplyInventory(
	_ context.Context,
	_ int64,
	update hosts.InventoryUpdate,
) error {
	s.applyCount++
	s.lastUpdate = update
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

func addrPtr(value string) *netip.Addr {
	addr := netip.MustParseAddr(value)
	return &addr
}
