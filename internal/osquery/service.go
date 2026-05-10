package osquery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/models"
	queryinfra "github.com/woodleighschool/woodstar/internal/queries"
	softwarepkg "github.com/woodleighschool/woodstar/internal/software"
	"github.com/woodleighschool/woodstar/internal/store"
)

var (
	ErrInvalidEnrollSecret = errors.New("invalid enroll secret")
	ErrMissingHardwareUUID = errors.New("hardware_uuid is required")
)

// Service performs osquery TLS-plugin operations.
type Service struct {
	hosts    *hosts.HostStore
	software *softwarepkg.SoftwareStore
	labels   labelStore
	queries  *queryinfra.QueryStore
	checks   *queryinfra.CheckStore
	live     *queryinfra.LiveQueryManager
	secrets  *models.SecretStore
	logger   *slog.Logger
}

// NewService returns an osquery service.
func NewService(
	hosts *hosts.HostStore,
	software *softwarepkg.SoftwareStore,
	labels *labels.LabelStore,
	queries *queryinfra.QueryStore,
	checks *queryinfra.CheckStore,
	live *queryinfra.LiveQueryManager,
	secrets *models.SecretStore,
	logger *slog.Logger,
) *Service {
	return &Service{
		hosts:    hosts,
		software: software,
		labels:   labels,
		queries:  queries,
		checks:   checks,
		live:     live,
		secrets:  secrets,
		logger:   logger,
	}
}

// Enroll validates the enroll secret, stores host details, and returns a node key.
func (s *Service) Enroll(ctx context.Context, req EnrollRequest) (string, error) {
	if s.hosts == nil || s.secrets == nil {
		return "", errors.New("osquery service is not configured")
	}

	ok, err := s.secrets.ValidateActive(ctx, models.SecretOrbit, req.EnrollSecret)
	if err != nil {
		return "", fmt.Errorf("validate enroll secret: %w", err)
	}
	if !ok {
		return "", ErrInvalidEnrollSecret
	}

	update := ParseHostDetails(req.HostDetails)
	if update.HardwareUUID == "" {
		update.HardwareUUID = strings.TrimSpace(req.HostIdentifier)
	}
	if update.HardwareUUID == "" {
		return "", ErrMissingHardwareUUID
	}

	nodeKey, err := generateNodeKey()
	if err != nil {
		return "", fmt.Errorf("generate node key: %w", err)
	}
	update.OsqueryNodeKey = nodeKey

	host, err := s.hosts.UpsertOnOsqueryEnroll(ctx, update)
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
	schedule, err := buildScheduleForHost(ctx, s.queries, *host)
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

	due := detailQueriesDue(host.DetailUpdatedAt, host.DetailQueryHash)

	labelCount, err := s.queueLabelQueries(ctx, host, due.Queries)
	if err != nil {
		return DistributedReadResponse{}, err
	}
	checkCount, err := s.queueCheckQueries(ctx, host, due.Queries)
	if err != nil {
		return DistributedReadResponse{}, err
	}
	liveCount := s.queueLiveQueries(host, due.Queries)

	s.logger.DebugContext(
		ctx,
		"osquery distributed queries prepared", "operation", "distributed_read",
		"host_id", host.ID,
		"query_count", len(due.Queries),
		"discovery_count", len(due.Discovery),
		"label_count", labelCount,
		"check_count", checkCount,
		"live_count", liveCount,
	)
	return DistributedReadResponse{
		NodeInvalid: false,
		Queries:     due.Queries,
		Discovery:   due.Discovery,
		Accelerate:  0,
	}, nil
}

func (s *Service) queueCheckQueries(ctx context.Context, host *hosts.Host, queries map[string]string) (int, error) {
	if s.checks == nil {
		return 0, nil
	}
	checks, err := s.checks.ApplicableForHost(ctx, *host)
	if err != nil {
		return 0, err
	}
	for _, check := range checks {
		queries[queryNameID(kindCheck, check.ID)] = check.Query
	}
	return len(checks), nil
}

// queueLiveQueries injects ephemeral live queries pending for host. The
// in-memory manager owns lifecycle; results route back through dispatch.
func (s *Service) queueLiveQueries(host *hosts.Host, queries map[string]string) int {
	if s.live == nil {
		return 0
	}
	work := s.live.PendingForHost(host.ID)
	for _, item := range work {
		queries[queryNameID(kindLive, item.QueryID)] = item.SQL
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
	if req.LogType == "result" && s.queries != nil {
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
	return s.hosts.ApplyDetail(ctx, host.ID, hosts.HostDetailUpdate{PublicIP: publicIP})
}

func (s *Service) hostByNodeKey(ctx context.Context, nodeKey string) (*hosts.Host, bool, error) {
	host, err := s.hosts.GetByOsqueryNodeKey(ctx, nodeKey)
	if errors.Is(err, store.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return host, true, nil
}

func ingestSoftwareMacOSWithEnrichment(
	ctx context.Context,
	svc *Service,
	hostID int64,
	rows []map[string]string,
	queryRows map[string][]map[string]string,
) error {
	if svc.software == nil {
		return nil
	}
	enrichment := softwareEnrichmentByPath(
		queryRows[querySoftwareMacOSCodesign],
		queryRows[querySoftwareMacOSExecutableHash],
	)
	rows = append(rows, queryRows[querySoftwareVSCodeExtensions]...)
	rows = append(rows, queryRows[querySoftwareJetBrainsPlugins]...)
	rows = append(rows, queryRows[querySoftwareGoBinaries]...)
	rows = append(rows, queryRows[querySoftwarePythonPackages]...)
	entries := parseSoftwareRows(rows, enrichment)
	if err := svc.software.ReplaceHostSoftware(ctx, hostID, entries); err != nil {
		return err
	}
	svc.logger.DebugContext(
		ctx,
		"software inventory ingested", "operation", "software_ingest",
		"host_id", hostID,
		"row_count", len(rows),
		"entry_count", len(entries),
		"codesign_count", len(queryRows[querySoftwareMacOSCodesign]),
		"executable_hash_count", len(queryRows[querySoftwareMacOSExecutableHash]),
	)
	return nil
}
