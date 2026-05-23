package santa

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	santasync "github.com/woodleighschool/woodstar/internal/santa/sync"
)

// Service coordinates Santa sync protocol stages.
type Service struct {
	store          *Store
	configurations *configurations.Store
	events         *santaevents.Store
	rules          *santarules.Store
	syncStore      *santasync.Store
}

type ServiceDependencies struct {
	Store          *Store
	Configurations *configurations.Store
	Events         *santaevents.Store
	Rules          *santarules.Store
	SyncStore      *santasync.Store
}

func NewService(deps ServiceDependencies) *Service {
	if deps.Store == nil || deps.Configurations == nil || deps.Events == nil || deps.Rules == nil ||
		deps.SyncStore == nil {
		panic("santa service requires store, configurations, events, rules, and sync store")
	}
	return &Service{
		store:          deps.Store,
		configurations: deps.Configurations,
		events:         deps.Events,
		rules:          deps.Rules,
		syncStore:      deps.SyncStore,
	}
}

func (s *Service) Preflight(
	ctx context.Context,
	machineID string,
	req santasync.PreflightRequest,
) (santasync.PreflightResponse, error) {
	hostID, err := s.store.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return santasync.PreflightResponse{}, err
	}
	if err := s.store.UpsertHostObservation(ctx, hostObservationFromPreflight(hostID, machineID, req)); err != nil {
		return santasync.PreflightResponse{}, err
	}

	effectiveRules, err := s.rules.ResolveRulesForHost(ctx, hostID)
	if err != nil {
		return santasync.PreflightResponse{}, err
	}
	targets := santarules.SyncTargetsFromRules(effectiveRules)
	pending := targets
	syncType := santasync.SyncTypeNormal
	pendingFullSync := false
	if req.RequestCleanSync {
		syncType = santasync.SyncTypeClean
		pendingFullSync = true
	}
	if err := s.syncStore.ReplacePending(
		ctx,
		hostID,
		req.RulesHash,
		targets,
		pending,
		pendingFullSync,
	); err != nil {
		return santasync.PreflightResponse{}, err
	}

	resp := santasync.PreflightResponse{SyncType: syncType}
	configuration, err := s.configurations.ResolveConfigurationForHost(ctx, hostID)
	if err != nil {
		return santasync.PreflightResponse{}, err
	}
	if configuration != nil {
		resp.Configuration = &configuration.Configuration
	}
	return resp, nil
}

func (s *Service) EventUpload(
	ctx context.Context,
	machineID string,
	req santasync.EventUploadRequest,
) (santasync.EventUploadResponse, error) {
	hostID, err := s.store.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return santasync.EventUploadResponse{}, err
	}
	if err := s.events.IngestExecutionEvents(ctx, hostID, req.Events); err != nil {
		return santasync.EventUploadResponse{}, err
	}
	return santasync.EventUploadResponse{}, nil
}

func (s *Service) RuleDownload(
	ctx context.Context,
	machineID string,
	_ santasync.RuleDownloadRequest,
) (santasync.RuleDownloadResponse, error) {
	hostID, err := s.store.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return santasync.RuleDownloadResponse{}, err
	}
	targets, err := s.syncStore.LoadPendingTargets(ctx, hostID)
	if err != nil {
		return santasync.RuleDownloadResponse{}, err
	}
	return santasync.RuleDownloadResponse{Rules: targets}, nil
}

func (s *Service) Postflight(
	ctx context.Context,
	machineID string,
	req santasync.PostflightRequest,
) (santasync.PostflightResponse, error) {
	hostID, err := s.store.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return santasync.PostflightResponse{}, err
	}
	if err := s.syncStore.PromotePending(
		ctx,
		hostID,
		req.RulesHash,
		req.RulesReceived,
		req.RulesProcessed,
	); err != nil {
		return santasync.PostflightResponse{}, err
	}
	return santasync.PostflightResponse{}, nil
}

func (s *Store) hostIDByMachineID(ctx context.Context, machineID string) (int64, error) {
	machineID = strings.TrimSpace(machineID)
	if machineID == "" {
		return 0, dbutil.ErrNotFound
	}
	var hostID int64
	err := s.db.Pool().QueryRow(ctx, `
		SELECT id
		FROM hosts
		WHERE hardware_uuid = $1
			AND deleted_at IS NULL
	`, machineID).Scan(&hostID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, dbutil.ErrNotFound
	}
	return hostID, err
}

func hostObservationFromPreflight(hostID int64, machineID string, req santasync.PreflightRequest) HostObservation {
	return HostObservation{
		HostID:             hostID,
		MachineID:          machineID,
		SerialNumber:       req.SerialNumber,
		Version:            req.Version,
		ClientModeReported: req.ClientMode,
		PrimaryUser:        req.PrimaryUser,
		PrimaryUserGroups:  req.PrimaryUserGroups,
		SIPStatus:          req.SIPStatus,
		OSBuild:            req.OSBuild,
		ModelIdentifier:    req.ModelIdentifier,
	}
}
