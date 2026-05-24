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
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

const ruleDownloadPageSize = 500

// Service coordinates Santa sync protocol stages.
type Service struct {
	hosts          hostStore
	configurations configurationResolver
	events         eventStore
	rules          ruleStore
	sync           syncStore
}

type Dependencies struct {
	HostStore      hostStore
	Configurations configurationResolver
	Events         eventStore
	Rules          ruleStore
	Sync           syncStore
}

type hostStore interface {
	hostIDByMachineID(context.Context, string) (int64, error)
	UpsertHostObservation(context.Context, HostObservation) error
}

type configurationResolver interface {
	ResolveConfigurationForHost(context.Context, int64) (*configurations.ResolvedConfiguration, error)
}

type eventStore interface {
	IngestExecutionEvents(context.Context, int64, []santaevents.ExecutionEventInput) error
}

type ruleStore interface {
	ResolveRulesForHost(context.Context, int64) ([]santarules.EffectiveRule, error)
}

type syncStore interface {
	PreparePending(
		context.Context,
		int64,
		string,
		[]syncstate.Target,
		syncstate.RuleCounts,
		bool,
	) (syncstate.SyncType, error)
	LoadPendingPayloadPage(context.Context, int64, string, int) (syncstate.PayloadRulePage, error)
	PromotePending(context.Context, int64, string, int, int) error
}

func NewService(deps Dependencies) *Service {
	return &Service{
		hosts:          deps.HostStore,
		configurations: deps.Configurations,
		events:         deps.Events,
		rules:          deps.Rules,
		sync:           deps.Sync,
	}
}

func (s *Service) Preflight(
	ctx context.Context,
	machineID string,
	req syncstate.PreflightRequest,
) (syncstate.PreflightResponse, error) {
	hostID, err := s.hosts.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return syncstate.PreflightResponse{}, err
	}
	if err := s.hosts.UpsertHostObservation(ctx, hostObservationFromPreflight(hostID, machineID, req)); err != nil {
		return syncstate.PreflightResponse{}, err
	}

	effectiveRules, err := s.rules.ResolveRulesForHost(ctx, hostID)
	if err != nil {
		return syncstate.PreflightResponse{}, err
	}
	targets := santarules.SyncTargetsFromRules(effectiveRules)
	syncType, err := s.sync.PreparePending(
		ctx,
		hostID,
		req.RulesHash,
		targets,
		req.RuleCounts,
		req.RequestCleanSync,
	)
	if err != nil {
		return syncstate.PreflightResponse{}, err
	}

	resp := syncstate.PreflightResponse{SyncType: syncType}
	configuration, err := s.configurations.ResolveConfigurationForHost(ctx, hostID)
	if err != nil {
		return syncstate.PreflightResponse{}, err
	}
	if configuration != nil {
		resp.Configuration = &configuration.Configuration
	}
	return resp, nil
}

func (s *Service) EventUpload(
	ctx context.Context,
	machineID string,
	req syncstate.EventUploadRequest,
) (syncstate.EventUploadResponse, error) {
	hostID, err := s.hosts.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return syncstate.EventUploadResponse{}, err
	}
	if err := s.events.IngestExecutionEvents(ctx, hostID, req.Events); err != nil {
		return syncstate.EventUploadResponse{}, err
	}
	return syncstate.EventUploadResponse{}, nil
}

func (s *Service) RuleDownload(
	ctx context.Context,
	machineID string,
	req syncstate.RuleDownloadRequest,
) (syncstate.RuleDownloadResponse, error) {
	hostID, err := s.hosts.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return syncstate.RuleDownloadResponse{}, err
	}
	page, err := s.sync.LoadPendingPayloadPage(ctx, hostID, req.Cursor, ruleDownloadPageSize)
	if err != nil {
		return syncstate.RuleDownloadResponse{}, err
	}
	return syncstate.RuleDownloadResponse(page), nil
}

func (s *Service) Postflight(
	ctx context.Context,
	machineID string,
	req syncstate.PostflightRequest,
) (syncstate.PostflightResponse, error) {
	hostID, err := s.hosts.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return syncstate.PostflightResponse{}, err
	}
	if err := s.sync.PromotePending(
		ctx,
		hostID,
		req.RulesHash,
		req.RulesReceived,
		req.RulesProcessed,
	); err != nil {
		return syncstate.PostflightResponse{}, err
	}
	return syncstate.PostflightResponse{}, nil
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

func hostObservationFromPreflight(hostID int64, machineID string, req syncstate.PreflightRequest) HostObservation {
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
