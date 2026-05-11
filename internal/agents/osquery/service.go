package osquery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/woodleighschool/woodstar/internal/agents"
	"github.com/woodleighschool/woodstar/internal/agents/catalog"
	"github.com/woodleighschool/woodstar/internal/agents/checks"
	"github.com/woodleighschool/woodstar/internal/agents/ingest"
	"github.com/woodleighschool/woodstar/internal/agents/livequery"
	"github.com/woodleighschool/woodstar/internal/agents/queries"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/secrets"
)

// Service performs osquery TLS-plugin operations.
type Service struct {
	hostStore          *hosts.HostStore
	inventoryProjector *ingest.Projector
	labelEvaluator     *ingest.LabelEvaluator
	queryStore         *queries.QueryStore
	checkStore         *checks.CheckStore
	liveQueries        *livequery.LiveQueryManager
	secretStore        *secrets.Store
	logger             *slog.Logger
}

// NewService returns an osquery service.
func NewService(
	hostStore *hosts.HostStore,
	inventoryProjector *ingest.Projector,
	labelEvaluator *ingest.LabelEvaluator,
	queryStore *queries.QueryStore,
	checkStore *checks.CheckStore,
	liveQueries *livequery.LiveQueryManager,
	secrets *secrets.Store,
	logger *slog.Logger,
) *Service {
	return &Service{
		hostStore:          hostStore,
		inventoryProjector: inventoryProjector,
		labelEvaluator:     labelEvaluator,
		queryStore:         queryStore,
		checkStore:         checkStore,
		liveQueries:        liveQueries,
		secretStore:        secrets,
		logger:             logger,
	}
}

// Enroll validates the enroll secret, stores host details, and returns a node key.
func (s *Service) Enroll(ctx context.Context, req EnrollRequest) (string, error) {
	ok, err := s.secretStore.HasActiveOrbitEnrollSecret(ctx, req.EnrollSecret)
	if err != nil {
		return "", fmt.Errorf("validate enroll secret: %w", err)
	}
	if !ok {
		return "", agents.ErrInvalidEnrollSecret
	}

	update := hosts.ParseHostDetails(req.HostDetails)
	if update.HardwareUUID == "" {
		update.HardwareUUID = strings.TrimSpace(req.HostIdentifier)
	}
	if update.HardwareUUID == "" {
		return "", agents.ErrMissingHardwareUUID
	}

	nodeKey, err := agents.GenerateNodeKey()
	if err != nil {
		return "", fmt.Errorf("generate node key: %w", err)
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
		"hardware_uuid", host.HardwareUUID,
		"display_name", host.DisplayName,
	)
	return nodeKey, nil
}

// Config returns the current osquery config including the host's report schedule.
func (s *Service) Config(ctx context.Context, nodeKey string, publicIP string) (ConfigResponse, error) {
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
	schedule, err := buildScheduleForHost(ctx, s.queryStore, *host)
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

// DistributedRead returns due detail, label, check, and campaign queries for a host.
func (s *Service) DistributedRead(
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

	due := catalog.DetailQueriesDue(host.DetailUpdatedAt, host.DetailQueryHash)
	detailQueries := make(map[string]string, len(due.Queries))
	for suffix, sql := range due.Queries {
		detailQueries[detailQueryName(suffix)] = sql
	}
	detailDiscovery := make(map[string]string, len(due.Discovery))
	for suffix, sql := range due.Discovery {
		detailDiscovery[detailQueryName(suffix)] = sql
	}

	labelCount, err := s.queueLabelQueries(ctx, host, detailQueries)
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

func (s *Service) queueCheckQueries(ctx context.Context, host *hosts.Host, queryMap map[string]string) (int, error) {
	if s.checkStore == nil {
		return 0, nil
	}
	checks, err := s.checkStore.ApplicableForHost(ctx, *host)
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
func (s *Service) queueLiveQueries(host *hosts.Host, queryMap map[string]string) int {
	if s.liveQueries == nil {
		return 0
	}
	work := s.liveQueries.PendingForHost(host.ID)
	for _, item := range work {
		queryMap[queryNameID(kindLive, item.QueryID)] = item.SQL
	}
	return len(work)
}

// DistributedWrite ingests results for every kind of distributed query.
func (s *Service) DistributedWrite(
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
func (s *Service) Log(ctx context.Context, nodeKey string, publicIP string, req LogRequest) (LogResponse, error) {
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
	if req.LogType == "result" && s.queryStore != nil {
		if err := s.ingestReportLogs(ctx, host.ID, req.Data); err != nil {
			s.logger.WarnContext(ctx, "report ingest failed", "host_id", host.ID, "err", err)
		}
	}
	return LogResponse{NodeInvalid: false}, nil
}

func (s *Service) recordHostPublicIP(ctx context.Context, host *hosts.Host, publicIP string) error {
	publicIP = strings.TrimSpace(publicIP)
	if publicIP == "" {
		return nil
	}
	return s.hostStore.ApplyDetail(ctx, host.ID, hosts.HostDetailUpdate{PublicIP: publicIP})
}

func (s *Service) hostByNodeKey(ctx context.Context, nodeKey string) (*hosts.Host, bool, error) {
	host, err := s.hostStore.GetByOsqueryNodeKey(ctx, nodeKey)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return host, true, nil
}
