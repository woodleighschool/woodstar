package santa

import (
	"context"

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
	req PreflightRequest,
) (PreflightResponse, error) {
	hostID, err := s.hosts.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return PreflightResponse{}, err
	}
	if err := s.hosts.UpsertHostObservation(ctx, hostObservationFromPreflight(hostID, machineID, req)); err != nil {
		return PreflightResponse{}, err
	}

	effectiveRules, err := s.rules.ResolveRulesForHost(ctx, hostID)
	if err != nil {
		return PreflightResponse{}, err
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
		return PreflightResponse{}, err
	}

	resp := PreflightResponse{SyncType: syncType}
	configuration, err := s.configurations.ResolveConfigurationForHost(ctx, hostID)
	if err != nil {
		return PreflightResponse{}, err
	}
	if configuration != nil {
		resp.Configuration = &configuration.Configuration
	}
	return resp, nil
}

func (s *Service) EventUpload(
	ctx context.Context,
	machineID string,
	req EventUploadRequest,
) (EventUploadResponse, error) {
	hostID, err := s.hosts.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return EventUploadResponse{}, err
	}
	if err := s.events.IngestExecutionEvents(ctx, hostID, req.Events); err != nil {
		return EventUploadResponse{}, err
	}
	return EventUploadResponse{}, nil
}

func (s *Service) RuleDownload(
	ctx context.Context,
	machineID string,
	req RuleDownloadRequest,
) (RuleDownloadResponse, error) {
	hostID, err := s.hosts.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return RuleDownloadResponse{}, err
	}
	return s.sync.LoadPendingPayloadPage(ctx, hostID, req.Cursor, ruleDownloadPageSize)
}

func (s *Service) Postflight(
	ctx context.Context,
	machineID string,
	req PostflightRequest,
) (PostflightResponse, error) {
	hostID, err := s.hosts.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return PostflightResponse{}, err
	}
	if err := s.sync.PromotePending(
		ctx,
		hostID,
		req.RulesHash,
		req.RulesReceived,
		req.RulesProcessed,
	); err != nil {
		return PostflightResponse{}, err
	}
	return PostflightResponse{}, nil
}

func hostObservationFromPreflight(hostID int64, machineID string, req PreflightRequest) HostObservation {
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
