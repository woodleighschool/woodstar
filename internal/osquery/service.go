package osquery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

// AgentService performs osquery TLS-plugin operations.
type AgentService struct {
	deps Dependencies
}

type Dependencies struct {
	HostStore          hostStore
	InventoryProjector inventoryProjector
	MunkiCollector     munkiCollector
	LabelEvaluator     labelEvaluator
	ReportStore        reportStore
	PolicyStore        policyStore
	LiveQueries        liveQueries
	SecretStore        agentauth.SecretVerifier
	Heartbeats         heartbeatRecorder
	Activity           activity.Recorder
	Logger             *slog.Logger
}

type munkiCollector interface {
	IngestCollection(ctx context.Context, hostID int64, collection munki.Collection) error
}

type hostStore interface {
	UpsertOnOsqueryEnroll(ctx context.Context, update hosts.InventoryUpdate) (*hosts.Host, error)
	GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*hosts.Host, error)
}

type heartbeatRecorder interface {
	Record(context.Context, int64, heartbeats.Source, heartbeats.Contact) error
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
	OverwriteSnapshot(
		ctx context.Context,
		reportID int64,
		queryHash string,
		hostID int64,
		rows []map[string]string,
		reportedAt time.Time,
	) error
	OverwriteError(
		ctx context.Context,
		reportID int64,
		queryHash string,
		hostID int64,
		reportError string,
		reportedAt time.Time,
	) error
}

type policyStore interface {
	IssueEvaluationsForHost(ctx context.Context, host *hosts.Host) ([]policies.Evaluation, error)
	RecordEvaluation(
		ctx context.Context,
		policyID int64,
		queryHash string,
		revision int64,
		sequence int64,
		hostID int64,
		result policies.EvaluationResult,
	) error
}

type liveQueries interface {
	PendingForHost(context.Context, int64) ([]livequery.Work, error)
	RecordResult(context.Context, livequery.Result) error
}

func NewAgentService(deps Dependencies) *AgentService {
	return &AgentService{deps: deps}
}

// Enroll validates the enroll secret, upserts the host, and returns a node key.
// Re-enrollment reuses the existing host identity and replaces its osquery node key.
func (s *AgentService) Enroll(ctx context.Context, req EnrollRequest, contact heartbeats.Contact) (string, error) {
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
	if err := s.deps.Heartbeats.Record(ctx, host.ID, heartbeats.SourceOsquery, contact); err != nil {
		return "", fmt.Errorf("record heartbeat: %w", err)
	}
	activity.RecordSystem(
		ctx,
		s.deps.Activity,
		s.deps.Logger,
		activity.AreaOsquery,
		activity.ActionOsqueryHostEnrolled,
		activity.Resource("host", host.ID, host.DisplayName),
	)
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
func (s *AgentService) Config(ctx context.Context, nodeKey string, contact heartbeats.Contact) (ConfigResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, nodeKey, contact)
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
			"logger_min_status":       "2",
		},
		Decorators: map[string][]string{},
	}, nil
}

// DistributedRead returns due detail, label, policy, and live queries for a host.
func (s *AgentService) DistributedRead(
	ctx context.Context,
	nodeKey string,
	contact heartbeats.Contact,
) (DistributedReadResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, nodeKey, contact)
	if err != nil {
		return DistributedReadResponse{}, err
	}
	if !ok {
		return DistributedReadResponse{NodeInvalid: true}, nil
	}

	inventoryUpdatedAt := host.InventoryUpdatedAt
	inventoryQueryHash := host.InventoryQueryHash
	if host.InventoryRefreshRequested {
		inventoryUpdatedAt = nil
		inventoryQueryHash = ""
	}
	due := catalog.DetailQueriesDue(inventoryUpdatedAt, inventoryQueryHash)
	detailQueries := make(map[string]string, len(due.Queries))
	for suffix, sql := range due.Queries {
		detailQueries[queryName(kindDetail, suffix)] = sql
	}
	detailDiscovery := make(map[string]string, len(due.Discovery))
	for suffix, sql := range due.Discovery {
		detailDiscovery[queryName(kindDetail, suffix)] = sql
	}

	labelCount, err := s.queueLabelQueries(ctx, detailQueries)
	if err != nil {
		return DistributedReadResponse{}, err
	}
	policyCount, err := s.queuePolicyQueries(ctx, host, detailQueries)
	if err != nil {
		return DistributedReadResponse{}, err
	}
	liveCount, err := s.queueLiveQueries(ctx, host, detailQueries)
	if err != nil {
		return DistributedReadResponse{}, err
	}

	s.deps.Logger.DebugContext(
		ctx,
		"osquery distributed queries prepared", "operation", "distributed_read",
		"host_id", host.ID,
		"query_count", len(detailQueries),
		"discovery_count", len(detailDiscovery),
		"label_count", labelCount,
		"policy_count", policyCount,
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

func (s *AgentService) queuePolicyQueries(
	ctx context.Context,
	host *hosts.Host,
	queryMap map[string]string,
) (int, error) {
	evaluations, err := s.deps.PolicyStore.IssueEvaluationsForHost(ctx, host)
	if err != nil {
		return 0, err
	}
	for _, evaluation := range evaluations {
		queryMap[queryNameForEvaluation(kindPolicy, evaluation)] = evaluation.Query
	}
	return len(evaluations), nil
}

// queueLiveQueries injects unfinished live queries targeting host.
func (s *AgentService) queueLiveQueries(
	ctx context.Context,
	host *hosts.Host,
	queryMap map[string]string,
) (int, error) {
	work, err := s.deps.LiveQueries.PendingForHost(ctx, host.ID)
	if err != nil {
		return 0, err
	}
	for _, item := range work {
		queryMap[queryNameID(kindLive, item.QueryID)] = item.SQL
	}
	return len(work), nil
}

// DistributedWrite ingests results for every kind of distributed query.
func (s *AgentService) DistributedWrite(
	ctx context.Context,
	req DistributedWriteRequest,
	contact heartbeats.Contact,
) (DistributedWriteResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, req.NodeKey, contact)
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

// Log accepts osquery scheduled-query logs and persists report observations.
func (s *AgentService) Log(ctx context.Context, nodeKey string, contact heartbeats.Contact, req LogRequest) (LogResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, nodeKey, contact)
	if err != nil {
		return LogResponse{}, err
	}
	if !ok {
		return LogResponse{NodeInvalid: true}, nil
	}
	switch req.LogType {
	case "result":
		if err := s.ingestReportLogs(ctx, host.ID, req.Data); err != nil {
			return LogResponse{}, fmt.Errorf("ingest report logs: %w", err)
		}
	case "status":
		if err := s.ingestReportStatusLogs(ctx, host.ID, req.Data); err != nil {
			return LogResponse{}, fmt.Errorf("ingest report status logs: %w", err)
		}
	}
	return LogResponse{NodeInvalid: false}, nil
}

func (s *AgentService) hostByNodeKey(ctx context.Context, nodeKey string, contact heartbeats.Contact) (*hosts.Host, bool, error) {
	host, err := s.deps.HostStore.GetByOsqueryNodeKey(ctx, nodeKey)
	if errors.Is(err, fault.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if err := s.deps.Heartbeats.Record(ctx, host.ID, heartbeats.SourceOsquery, contact); err != nil {
		return nil, false, fmt.Errorf("record heartbeat: %w", err)
	}
	return host, true, nil
}
