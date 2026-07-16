package osquery

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

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
	host *hosts.Host
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
