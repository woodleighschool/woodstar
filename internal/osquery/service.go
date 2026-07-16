package osquery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

// AgentService performs osquery TLS-plugin operations.
type AgentService struct {
	deps Dependencies
}

type Dependencies struct {
	HostStore          hostStore
	InventoryProjector inventoryProjector
	LabelEvaluator     labelEvaluator
	ReportStore        reportStore
	CheckStore         checkStore
	LiveQueries        liveQueries
	SecretStore        agentauth.SecretVerifier
	Logger             *slog.Logger
}

type hostStore interface {
	UpsertOnOsqueryEnroll(ctx context.Context, update hosts.InventoryUpdate) (*hosts.Host, error)
	GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*hosts.Host, error)
	ApplyInventory(ctx context.Context, hostID int64, update hosts.InventoryUpdate) error
}

type inventoryProjector interface {
	IngestDetail(
		ctx context.Context,
		query catalog.DetailQuery,
		status string,
		hostID int64,
		rows []map[string]string,
	) error
	IngestSoftware(ctx context.Context, hostID int64, rows map[string][]map[string]string) error
	MarkFresh(ctx context.Context, hostID int64) error
}

type labelEvaluator interface {
	ApplicableLabels(ctx context.Context) ([]labels.DynamicLabel, error)
	Finalize(ctx context.Context, host *hosts.Host, results []ingest.LabelResult) error
}

type reportStore interface {
	ScheduledForHost(ctx context.Context, host *hosts.Host) ([]reports.Report, error)
	OverwriteResults(ctx context.Context, reportID, hostID int64, rows []map[string]string, updatedAt time.Time) error
}

type checkStore interface {
	ApplicableForHost(ctx context.Context, host *hosts.Host) ([]checks.Check, error)
	UpsertMembership(ctx context.Context, checkID, hostID int64, result *bool) error
}

type liveQueries interface {
	PendingForHost(hostID int64) []livequery.Work
	RecordResult(result livequery.Result)
}

func NewAgentService(deps Dependencies) *AgentService {
	return &AgentService{deps: deps}
}

// Enroll validates the enroll secret, stores host details, and returns a node key.
func (s *AgentService) Enroll(ctx context.Context, req EnrollRequest) (string, error) {
	nodeKey, err := enrollment.IssueNodeKey(ctx, s.deps.SecretStore, req.EnrollSecret)
	if err != nil {
		return "", err
	}

	update := ingest.ParseHostDetails(req.HostDetails)
	if update.Hardware.UUID == "" {
		update.Hardware.UUID = req.HostIdentifier
	}
	if update.Hardware.UUID == "" {
		return "", enrollment.ErrMissingHardwareUUID
	}
	update.OsqueryNodeKey = nodeKey

	host, err := s.deps.HostStore.UpsertOnOsqueryEnroll(ctx, update)
	if err != nil {
		return "", fmt.Errorf("upsert host: %w", err)
	}
	s.deps.Logger.DebugContext(
		ctx,
		"osquery host enrolled", "operation", "enroll",
		"host_id", host.ID,
		"hardware_uuid", host.Hardware.UUID,
		"display_name", host.DisplayName,
	)
	return nodeKey, nil
}

// Config returns the current osquery config including the host's report schedule.
func (s *AgentService) Config(ctx context.Context, nodeKey string, publicIP string) (ConfigResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, nodeKey, publicIP)
	if err != nil {
		return ConfigResponse{}, err
	}
	if !ok {
		return ConfigResponse{NodeInvalid: true}, nil
	}
	schedule, err := buildScheduleForHost(ctx, s.deps.ReportStore, host)
	if err != nil {
		return ConfigResponse{}, err
	}
	return ConfigResponse{
		NodeInvalid: false,
		Schedule:    schedule,
		Options: map[string]string{
			"disable_distributed":     "false",
			"disable_carver":          "true",
			"carver_disable_function": "true",
			"logger_min_status":       "4",
		},
		Decorators: map[string][]string{},
	}, nil
}

// DistributedRead returns due detail, label, check, and live queries for a host.
func (s *AgentService) DistributedRead(
	ctx context.Context,
	nodeKey string,
	publicIP string,
) (DistributedReadResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, nodeKey, publicIP)
	if err != nil {
		return DistributedReadResponse{}, err
	}
	if !ok {
		return DistributedReadResponse{NodeInvalid: true}, nil
	}

	due := catalog.DetailQueriesDue(host.Timestamps.InventoryUpdatedAt, host.InventoryQueryHash)
	detailQueries := make(map[string]string, len(due.Queries))
	for suffix, sql := range due.Queries {
		detailQueries[detailQueryName(suffix)] = sql
	}
	detailDiscovery := make(map[string]string, len(due.Discovery))
	for suffix, sql := range due.Discovery {
		detailDiscovery[detailQueryName(suffix)] = sql
	}

	labelCount, err := s.queueLabelQueries(ctx, detailQueries)
	if err != nil {
		return DistributedReadResponse{}, err
	}
	checkCount, err := s.queueCheckQueries(ctx, host, detailQueries)
	if err != nil {
		return DistributedReadResponse{}, err
	}
	liveCount := s.queueLiveQueries(host, detailQueries)

	s.deps.Logger.DebugContext(
		ctx,
		"osquery distributed queries prepared", "operation", "distributed_read",
		"host_id", host.ID,
		"query_count", len(detailQueries),
		"discovery_count", len(detailDiscovery),
		"label_count", labelCount,
		"check_count", checkCount,
		"live_count", liveCount,
	)
	return DistributedReadResponse{
		NodeInvalid: false,
		Queries:     detailQueries,
		Discovery:   detailDiscovery,
	}, nil
}

func (s *AgentService) queueLabelQueries(
	ctx context.Context,
	queryMap map[string]string,
) (int, error) {
	labelRows, err := s.deps.LabelEvaluator.ApplicableLabels(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, label := range labelRows {
		queryMap[queryNameID(kindLabel, label.ID)] = label.Query
		count++
	}
	return count, nil
}

func (s *AgentService) queueCheckQueries(
	ctx context.Context,
	host *hosts.Host,
	queryMap map[string]string,
) (int, error) {
	checks, err := s.deps.CheckStore.ApplicableForHost(ctx, host)
	if err != nil {
		return 0, err
	}
	for _, check := range checks {
		queryMap[queryNameID(kindCheck, check.ID)] = check.Query
	}
	return len(checks), nil
}

// queueLiveQueries injects ephemeral live queries pending for host. The
// in-memory manager owns lifecycle; results route back through dispatch.
func (s *AgentService) queueLiveQueries(host *hosts.Host, queryMap map[string]string) int {
	work := s.deps.LiveQueries.PendingForHost(host.ID)
	for _, item := range work {
		queryMap[queryNameID(kindLive, item.QueryID)] = item.SQL
	}
	return len(work)
}

// DistributedWrite ingests results for every kind of distributed query.
func (s *AgentService) DistributedWrite(
	ctx context.Context,
	req DistributedWriteRequest,
	publicIP string,
) (DistributedWriteResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, req.NodeKey, publicIP)
	if err != nil {
		return DistributedWriteResponse{}, err
	}
	if !ok {
		return DistributedWriteResponse{NodeInvalid: true}, nil
	}
	if err := s.dispatchWriteResults(ctx, host, req); err != nil {
		return DistributedWriteResponse{}, err
	}
	return DistributedWriteResponse{NodeInvalid: false}, nil
}

// Log accepts osquery scheduled-query logs and persists snapshot results.
func (s *AgentService) Log(ctx context.Context, nodeKey string, publicIP string, req LogRequest) (LogResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, nodeKey, publicIP)
	if err != nil {
		return LogResponse{}, err
	}
	if !ok {
		return LogResponse{NodeInvalid: true}, nil
	}
	if req.LogType == "result" {
		if err := s.ingestReportLogs(ctx, host.ID, req.Data); err != nil {
			return LogResponse{}, fmt.Errorf("ingest report logs: %w", err)
		}
	}
	return LogResponse{NodeInvalid: false}, nil
}

func (s *AgentService) hostByNodeKey(ctx context.Context, nodeKey string, publicIP string) (*hosts.Host, bool, error) {
	host, err := s.deps.HostStore.GetByOsqueryNodeKey(ctx, nodeKey)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if publicIP != "" && !hostPublicIPMatches(host, publicIP) {
		if err := s.deps.HostStore.ApplyInventory(ctx, host.ID, hosts.InventoryUpdate{
			Network: hosts.InventoryNetwork{LastRemoteIP: publicIP},
		}); err != nil {
			return nil, false, err
		}
	}
	return host, true, nil
}

func hostPublicIPMatches(host *hosts.Host, publicIP string) bool {
	return host.Network.LastRemoteIP != nil && host.Network.LastRemoteIP.String() == publicIP
}
