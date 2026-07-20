package santa

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

const ruleDownloadPageSize int32 = 500

// SyncService coordinates Santa sync protocol stages.
type SyncService struct {
	deps Dependencies
}

type Dependencies struct {
	HostStore      hostStore
	Configurations configurationResolver
	Events         eventStore
	Rules          ruleStore
	Sync           syncStore
}

type hostStore interface {
	hostIDByMachineID(ctx context.Context, machineID string) (int64, error)
	UpsertHostObservation(ctx context.Context, observation HostObservation) error
}

type configurationResolver interface {
	ResolveConfigurationForHost(ctx context.Context, hostID int64) (*configurations.ConfigurationMatch, error)
}

type eventStore interface {
	IngestEvents(
		ctx context.Context,
		hostID int64,
		executionEvents []santaevents.ExecutionEventInput,
		fileAccessEvents []santaevents.FileAccessEventInput,
		standaloneEvents []santaevents.StandaloneRuleCreationEventInput,
	) ([]string, error)
}

type ruleStore interface {
	ResolveRulesForHost(ctx context.Context, hostID int64) ([]santarules.HostRule, error)
}

type syncStore interface {
	PreparePending(
		ctx context.Context,
		hostID int64,
		targets []syncstate.Target,
		reported syncstate.RuleCounts,
		requestCleanSync bool,
		clientRulesHash string,
	) (syncstate.SyncType, error)
	LoadPendingPayloadPage(
		ctx context.Context,
		hostID int64,
		cursor string,
		limit int32,
	) (syncstate.PayloadRulePage, error)
	PromotePending(
		ctx context.Context,
		hostID int64,
		rulesReceived, rulesProcessed uint32,
		syncType syncstate.SyncType,
		rulesHash string,
	) error
}

func NewSyncService(deps Dependencies) *SyncService {
	return &SyncService{deps: deps}
}

func (s *SyncService) Preflight(
	ctx context.Context,
	machineID string,
	req PreflightRequest,
) (PreflightResponse, error) {
	hostID, err := s.deps.HostStore.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return PreflightResponse{}, err
	}
	if err := s.deps.HostStore.UpsertHostObservation(
		ctx,
		hostObservationFromPreflight(hostID, machineID, req),
	); err != nil {
		return PreflightResponse{}, err
	}
	rules, err := s.deps.Rules.ResolveRulesForHost(ctx, hostID)
	if err != nil {
		return PreflightResponse{}, err
	}
	targets := santarules.SyncTargetsFromRules(rules)
	syncType, err := s.deps.Sync.PreparePending(
		ctx,
		hostID,
		targets,
		req.RuleCounts,
		req.RequestCleanSync,
		req.RulesHash,
	)
	if err != nil {
		return PreflightResponse{}, err
	}

	resp := PreflightResponse{SyncType: syncType}
	configuration, err := s.deps.Configurations.ResolveConfigurationForHost(ctx, hostID)
	if err != nil {
		return PreflightResponse{}, err
	}
	if configuration != nil {
		resp.Configuration = &configuration.Configuration
	}
	return resp, nil
}

func (s *SyncService) EventUpload(
	ctx context.Context,
	machineID string,
	req EventUploadRequest,
) (EventUploadResponse, error) {
	hostID, err := s.deps.HostStore.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return EventUploadResponse{}, err
	}
	bundleRequests, err := s.deps.Events.IngestEvents(
		ctx,
		hostID,
		req.Events,
		req.FileAccessEvents,
		req.StandaloneRuleCreationEvents,
	)
	if err != nil {
		return EventUploadResponse{}, err
	}
	return EventUploadResponse{BundleBinaryRequests: bundleRequests}, nil
}

func (s *SyncService) RuleDownload(
	ctx context.Context,
	machineID string,
	req RuleDownloadRequest,
) (RuleDownloadResponse, error) {
	hostID, err := s.deps.HostStore.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return RuleDownloadResponse{}, err
	}
	return s.deps.Sync.LoadPendingPayloadPage(ctx, hostID, req.Cursor, ruleDownloadPageSize)
}

func (s *SyncService) Postflight(
	ctx context.Context,
	machineID string,
	req PostflightRequest,
) (PostflightResponse, error) {
	hostID, err := s.deps.HostStore.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return PostflightResponse{}, err
	}
	if err := s.deps.Sync.PromotePending(
		ctx,
		hostID,
		req.RulesReceived,
		req.RulesProcessed,
		req.SyncType,
		req.RulesHash,
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
	}
}
