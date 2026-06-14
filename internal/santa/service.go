package santa

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

const ruleDownloadPageSize int32 = 500

// SyncService coordinates Santa sync protocol stages.
type SyncService struct {
	hosts          hostStore
	configurations configurationResolver
	userAffinities userAffinityStore
	events         eventStore
	rules          ruleStore
	sync           syncStore
}

type Dependencies struct {
	HostStore      hostStore
	Configurations configurationResolver
	UserAffinities userAffinityStore
	Events         eventStore
	Rules          ruleStore
	Sync           syncStore
}

type hostStore interface {
	hostIDByMachineID(context.Context, string) (int64, error)
	UpsertHostObservation(context.Context, HostObservation) error
}

type userAffinityStore interface {
	Upsert(context.Context, int64, string, hosts.UserAffinitySource) error
}

type configurationResolver interface {
	ResolveConfigurationForHost(context.Context, int64) (*configurations.ConfigurationMatch, error)
}

type eventStore interface {
	IngestEvents(
		context.Context,
		int64,
		[]santaevents.ExecutionEventInput,
		[]santaevents.FileAccessEventInput,
	) ([]string, error)
}

type ruleStore interface {
	ResolveRulesForHost(context.Context, int64) ([]santarules.HostRule, error)
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
	LoadPendingPayloadPage(context.Context, int64, string, int32) (syncstate.PayloadRulePage, error)
	PromotePending(context.Context, int64, string, int32, int32) error
}

func NewSyncService(deps Dependencies) *SyncService {
	return &SyncService{
		hosts:          deps.HostStore,
		configurations: deps.Configurations,
		userAffinities: deps.UserAffinities,
		events:         deps.Events,
		rules:          deps.Rules,
		sync:           deps.Sync,
	}
}

func (s *SyncService) Preflight(
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
	if s.userAffinities != nil {
		if err := s.userAffinities.Upsert(
			ctx,
			hostID,
			req.PrimaryUser,
			hosts.UserAffinitySourceSantaPrimaryUser,
		); err != nil {
			return PreflightResponse{}, err
		}
	}

	rules, err := s.rules.ResolveRulesForHost(ctx, hostID)
	if err != nil {
		return PreflightResponse{}, err
	}
	targets := santarules.SyncTargetsFromRules(rules)
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

func (s *SyncService) EventUpload(
	ctx context.Context,
	machineID string,
	req EventUploadRequest,
) (EventUploadResponse, error) {
	hostID, err := s.hosts.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return EventUploadResponse{}, err
	}
	bundleRequests, err := s.events.IngestEvents(ctx, hostID, req.Events, req.FileAccessEvents)
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
	hostID, err := s.hosts.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return RuleDownloadResponse{}, err
	}
	return s.sync.LoadPendingPayloadPage(ctx, hostID, req.Cursor, ruleDownloadPageSize)
}

func (s *SyncService) Postflight(
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
