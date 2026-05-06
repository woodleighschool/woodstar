package osquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/woodleighschool/woodstar/internal/models"
)

var (
	ErrInvalidEnrollSecret = errors.New("invalid enroll secret")
	ErrMissingHardwareUUID = errors.New("hardware_uuid is required")
)

// Service performs osquery TLS-plugin operations.
type Service struct {
	hosts    *models.HostStore
	software *models.SoftwareStore
	secrets  *models.SecretStore
	logger   *slog.Logger
}

// NewService returns an osquery service.
func NewService(
	hosts *models.HostStore,
	software *models.SoftwareStore,
	secrets *models.SecretStore,
	logger *slog.Logger,
) *Service {
	return &Service{hosts: hosts, software: software, secrets: secrets, logger: logger}
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

// Config returns the current osquery config.
func (s *Service) Config(ctx context.Context, nodeKey string) (ConfigResponse, error) {
	if ok, err := s.validNodeKey(ctx, nodeKey); err != nil {
		return ConfigResponse{}, err
	} else if !ok {
		return ConfigResponse{NodeInvalid: true}, nil
	}
	return ConfigResponse{
		NodeInvalid: false,
		Schedule:    map[string]string{},
		Options: map[string]string{
			"disable_distributed": "false",
		},
		Decorators: map[string]any{},
	}, nil
}

// DistributedRead returns built-in detail queries when the host is stale.
func (s *Service) DistributedRead(ctx context.Context, nodeKey string) (DistributedReadResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, nodeKey)
	if err != nil {
		return DistributedReadResponse{}, err
	}
	if !ok {
		return DistributedReadResponse{NodeInvalid: true}, nil
	}
	due := detailQueriesDue(host.DetailUpdatedAt)
	s.logger.DebugContext(
		ctx,
		"osquery distributed queries prepared", "operation", "distributed_read",
		"host_id", host.ID,
		"query_count", len(due.Queries),
		"discovery_count", len(due.Discovery),
	)
	return DistributedReadResponse{
		NodeInvalid: false,
		Queries:     due.Queries,
		Discovery:   due.Discovery,
		Accelerate:  0,
	}, nil
}

// DistributedWrite ingests successful built-in detail query results.
func (s *Service) DistributedWrite(ctx context.Context, req DistributedWriteRequest) (DistributedWriteResponse, error) {
	host, ok, err := s.hostByNodeKey(ctx, req.NodeKey)
	if err != nil {
		return DistributedWriteResponse{}, err
	}
	if !ok {
		return DistributedWriteResponse{NodeInvalid: true}, nil
	}

	registry := DetailQueries()
	allDetailSucceeded := true
	for name, rows := range req.Queries {
		if name == querySoftwareMacOS {
			continue
		}
		query, ok := registry[name]
		if !ok {
			continue
		}
		if !statusOK(req.Statuses[name]) {
			if !query.Optional {
				allDetailSucceeded = false
			}
			s.logger.WarnContext(
				ctx,
				"osquery detail query failed", "operation", "distributed_write",
				"host_id", host.ID,
				"query", name,
				"optional", query.Optional,
				"message", req.Messages[name],
			)
			continue
		}
		if err := query.Ingest(ctx, s, host.ID, rows); err != nil {
			return DistributedWriteResponse{}, fmt.Errorf("ingest %s: %w", name, err)
		}
	}
	if rows, ok := req.Queries[querySoftwareMacOS]; ok {
		if !statusOK(req.Statuses[querySoftwareMacOS]) {
			allDetailSucceeded = false
			s.logger.WarnContext(
				ctx,
				"osquery detail query failed", "operation", "distributed_write",
				"host_id", host.ID,
				"query", querySoftwareMacOS,
				"optional", false,
				"message", req.Messages[querySoftwareMacOS],
			)
		} else if err := ingestSoftwareMacOSWithEnrichment(ctx, s, host.ID, rows, req.Queries); err != nil {
			return DistributedWriteResponse{}, fmt.Errorf("ingest %s: %w", querySoftwareMacOS, err)
		}
	}
	if allDetailSucceeded && sawEveryRequiredDetailQuery(req.Queries, registry) {
		if err := s.hosts.MarkDetailFresh(ctx, host.ID); err != nil {
			return DistributedWriteResponse{}, err
		}
		s.logger.DebugContext(
			ctx,
			"osquery detail inventory refreshed", "operation", "inventory_refresh",
			"host_id", host.ID,
			"query_count", len(req.Queries),
		)
	}
	return DistributedWriteResponse{NodeInvalid: false}, nil
}

// Log accepts osquery logs. Storage and retention are a later slice.
func (s *Service) Log(ctx context.Context, nodeKey string) (LogResponse, error) {
	if ok, err := s.validNodeKey(ctx, nodeKey); err != nil {
		return LogResponse{}, err
	} else if !ok {
		return LogResponse{NodeInvalid: true}, nil
	}
	return LogResponse{NodeInvalid: false}, nil
}

func (s *Service) validNodeKey(ctx context.Context, nodeKey string) (bool, error) {
	_, ok, err := s.hostByNodeKey(ctx, nodeKey)
	return ok, err
}

func (s *Service) hostByNodeKey(ctx context.Context, nodeKey string) (*models.Host, bool, error) {
	host, err := s.hosts.GetByOsqueryNodeKey(ctx, nodeKey)
	if errors.Is(err, models.ErrNotFound) {
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

func sawEveryRequiredDetailQuery(results map[string][]map[string]string, registry map[string]DetailQuery) bool {
	for name, query := range registry {
		if query.Optional {
			continue
		}
		if _, ok := results[name]; !ok {
			return false
		}
	}
	return true
}

func statusOK(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number == 0
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text == "" || text == "0"
	}
	return false
}
