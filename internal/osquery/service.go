package osquery

import (
	"context"
	"encoding/json"
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
	hostStore          hostStore
	inventoryProjector inventoryProjector
	labelEvaluator     labelEvaluator
	reportStore        reportStore
	checkStore         checkStore
	liveQueries        liveQueries
	secretStore        agentauth.SecretVerifier
	logger             *slog.Logger
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
	UpsertOnOsqueryEnroll(context.Context, hosts.InventoryUpdate) (*hosts.Host, error)
	GetByOsqueryNodeKey(context.Context, string) (*hosts.Host, error)
	ApplyInventory(context.Context, int64, hosts.InventoryUpdate) error
}

type inventoryProjector interface {
	IngestDetail(context.Context, catalog.DetailQuery, string, int64, []map[string]string) error
	IngestSoftware(context.Context, int64, map[string][]map[string]string) error
	MarkFresh(context.Context, int64) error
}

type labelEvaluator interface {
	ApplicableLabels(context.Context) ([]labels.Label, error)
	Finalize(context.Context, *hosts.Host, []ingest.LabelResult) error
}

type reportStore interface {
	ScheduledForHost(context.Context, *hosts.Host) ([]reports.Report, error)
	OverwriteResults(context.Context, int64, int64, []map[string]string, time.Time) error
}

type checkStore interface {
	ApplicableForHost(context.Context, *hosts.Host) ([]checks.Check, error)
	UpsertMembership(context.Context, int64, int64, *bool) error
}

type liveQueries interface {
	PendingForHost(int64) []livequery.Work
	RecordResult(int64, int64, string, livequery.Status, json.RawMessage, string)
}

func NewAgentService(deps Dependencies) *AgentService {
	return &AgentService{
		hostStore:          deps.HostStore,
		inventoryProjector: deps.InventoryProjector,
		labelEvaluator:     deps.LabelEvaluator,
		reportStore:        deps.ReportStore,
		checkStore:         deps.CheckStore,
		liveQueries:        deps.LiveQueries,
		secretStore:        deps.SecretStore,
		logger:             deps.Logger,
	}
}

// Enroll validates the enroll secret, stores host details, and returns a node key.
func (s *AgentService) Enroll(ctx context.Context, req EnrollRequest) (string, error) {
	nodeKey, err := enrollment.IssueNodeKey(ctx, s.secretStore, req.EnrollSecret)
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

	host, err := s.hostStore.UpsertOnOsqueryEnroll(ctx, update)
	if err != nil {
		return "", fmt.Errorf("upsert host: %w", err)
	}
	s.logger.DebugContext(
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
	host, ok, err := s.hostByNodeKey(ctx, nodeKey)
	if err != nil {
		return ConfigResponse{}, err
	}
	if !ok {
		return ConfigResponse{NodeInvalid: true}, nil
	}
	if err := s.recordHostPublicIP(ctx, host, publicIP); err != nil {
		return ConfigResponse{}, err
	}
	schedule, err := buildScheduleForHost(ctx, s.reportStore, host)
	if err != nil {
		return ConfigResponse{}, err
	}
	return ConfigResponse{
		NodeInvalid: false,
		Schedule:    schedule,
		Options: map[string]string{
			"disable_distributed": "false",
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
	host, ok, err := s.hostByNodeKey(ctx, nodeKey)
	if err != nil {
		return DistributedReadResponse{}, err
	}
	if !ok {
		return DistributedReadResponse{NodeInvalid: true}, nil
	}
	if err := s.recordHostPublicIP(ctx, host, publicIP); err != nil {
		return DistributedReadResponse{}, err
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

	s.logger.DebugContext(
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
		Accelerate:  0,
	}, nil
}

func (s *AgentService) queueLabelQueries(
	ctx context.Context,
	queryMap map[string]string,
) (int, error) {
	labelRows, err := s.labelEvaluator.ApplicableLabels(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, label := range labelRows {
		if label.Query == nil {
			continue
		}
		queryMap[queryNameID(kindLabel, label.ID)] = *label.Query
		count++
	}
	return count, nil
}

func (s *AgentService) queueCheckQueries(
	ctx context.Context,
	host *hosts.Host,
	queryMap map[string]string,
) (int, error) {
	checks, err := s.checkStore.ApplicableForHost(ctx, host)
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
	work := s.liveQueries.PendingForHost(host.ID)
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
	host, ok, err := s.hostByNodeKey(ctx, req.NodeKey)
	if err != nil {
		return DistributedWriteResponse{}, err
	}
	if !ok {
		return DistributedWriteResponse{NodeInvalid: true}, nil
	}
	if err := s.recordHostPublicIP(ctx, host, publicIP); err != nil {
		return DistributedWriteResponse{}, err
	}
	if err := s.dispatchWriteResults(ctx, host, req); err != nil {
		return DistributedWriteResponse{}, err
	}
	return DistributedWriteResponse{NodeInvalid: false}, nil
}

// Log accepts osquery scheduled-query logs and persists snapshot results.
func (s *AgentService) Log(ctx context.Context, nodeKey string, publicIP string, req LogRequest) (LogResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, nodeKey)
	if err != nil {
		return LogResponse{}, err
	}
	if !ok {
		return LogResponse{NodeInvalid: true}, nil
	}
	if err := s.recordHostPublicIP(ctx, host, publicIP); err != nil {
		return LogResponse{}, err
	}
	if req.LogType == "result" {
		if err := s.ingestReportLogs(ctx, host.ID, req.Data); err != nil {
			s.logger.WarnContext(ctx, "report ingest failed", "host_id", host.ID, "err", err)
		}
	}
	return LogResponse{NodeInvalid: false}, nil
}

func (s *AgentService) recordHostPublicIP(ctx context.Context, host *hosts.Host, publicIP string) error {
	if publicIP == "" {
		return nil
	}
	return s.hostStore.ApplyInventory(ctx, host.ID, hosts.InventoryUpdate{
		Network: hosts.InventoryNetwork{LastRemoteIP: publicIP},
	})
}

func (s *AgentService) hostByNodeKey(ctx context.Context, nodeKey string) (*hosts.Host, bool, error) {
	host, err := s.hostStore.GetByOsqueryNodeKey(ctx, nodeKey)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return host, true, nil
}
