package protocol

import (
	"fmt"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

func preflightRequestFromProto(req *syncv1.PreflightRequest) (santa.PreflightRequest, error) {
	var sipStatus *int16
	if req.GetSipStatus() != 0 {
		value := int16(req.GetSipStatus())
		sipStatus = &value
	}
	return santa.PreflightRequest{
		SerialNumber:      req.GetSerialNumber(),
		Version:           req.GetSantaVersion(),
		RulesHash:         req.GetRulesHash(),
		ClientMode:        clientModeFromProto(req.GetClientMode()),
		RequestCleanSync:  req.GetRequestCleanSync(),
		RuleCounts:        ruleCountsFromProto(req),
		PrimaryUser:       req.GetPrimaryUser(),
		PrimaryUserGroups: req.GetPrimaryUserGroups(),
		SIPStatus:         sipStatus,
	}, nil
}

func preflightResponseToProto(resp santa.PreflightResponse) (*syncv1.PreflightResponse, error) {
	syncType, err := protoSyncType(resp.SyncType)
	if err != nil {
		return nil, err
	}
	out := &syncv1.PreflightResponse{SyncType: &syncType}
	if resp.Configuration != nil {
		if err := applyConfigurationToPreflightResponse(out, resp.Configuration); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func ruleCountsFromProto(req *syncv1.PreflightRequest) syncstate.RuleCounts {
	return syncstate.RuleCounts{
		Binary:      int32(req.GetBinaryRuleCount()),
		Certificate: int32(req.GetCertificateRuleCount()),
		TeamID:      int32(req.GetTeamidRuleCount()),
		SigningID:   int32(req.GetSigningidRuleCount()),
		CDHash:      int32(req.GetCdhashRuleCount()),
		Compiler:    int32(req.GetCompilerRuleCount()),
		Transitive:  int32(req.GetTransitiveRuleCount()),
	}
}

func applyConfigurationToPreflightResponse(resp *syncv1.PreflightResponse, config *configurations.Configuration) error {
	clientMode, err := protoClientMode(config.ClientMode)
	if err != nil {
		return err
	}
	resp.ClientMode = clientMode
	resp.EnableBundles = &config.EnableBundles
	resp.EnableTransitiveRules = &config.EnableTransitiveRules
	resp.EnableAllEventUpload = &config.EnableAllEventUpload
	resp.DisableUnknownEventUpload = &config.DisableUnknownEventUpload
	fileAccessAction, err := protoFileAccessAction(config.OverrideFileAccessAction)
	if err != nil {
		return err
	}
	resp.OverrideFileAccessAction = &fileAccessAction
	resp.FullSyncIntervalSeconds = uint32(config.FullSyncIntervalSeconds)
	resp.BatchSize = uint32(config.BatchSize)
	if config.AllowedPathRegex != "" {
		resp.AllowedPathRegex = &config.AllowedPathRegex
	}
	if config.BlockedPathRegex != "" {
		resp.BlockedPathRegex = &config.BlockedPathRegex
	}
	if config.EventDetailURL != "" {
		resp.EventDetailUrl = &config.EventDetailURL
	}
	if config.EventDetailText != "" {
		resp.EventDetailText = &config.EventDetailText
	}
	resp.RemovableMediaPolicy = protoRemovableMediaPolicy(config.RemovableMediaPolicy)
	resp.EncryptedRemovableMediaPolicy = protoRemovableMediaPolicy(config.EncryptedRemovableMediaPolicy)
	return nil
}

func protoRemovableMediaPolicy(policy configurations.RemovableMediaPolicy) *syncv1.RemovableMediaPolicy {
	switch policy.Action {
	case configurations.RemovableMediaActionAllow:
		return &syncv1.RemovableMediaPolicy{Action: &syncv1.RemovableMediaPolicy_Allow{Allow: true}}
	case configurations.RemovableMediaActionBlock:
		return &syncv1.RemovableMediaPolicy{Action: &syncv1.RemovableMediaPolicy_Block{Block: true}}
	case configurations.RemovableMediaActionRemount:
		return &syncv1.RemovableMediaPolicy{
			Action: &syncv1.RemovableMediaPolicy_Remount{
				Remount: &syncv1.RemountPolicy{Flags: policy.RemountFlags},
			},
		}
	default:
		return nil
	}
}

func protoFileAccessAction(action configurations.FileAccessAction) (syncv1.FileAccessAction, error) {
	switch action {
	case configurations.FileAccessActionNone:
		return syncv1.FileAccessAction_NONE, nil
	case configurations.FileAccessActionAuditOnly:
		return syncv1.FileAccessAction_AUDIT_ONLY, nil
	case configurations.FileAccessActionDisable:
		return syncv1.FileAccessAction_DISABLE, nil
	default:
		return syncv1.FileAccessAction_FILE_ACCESS_ACTION_UNSPECIFIED, fmt.Errorf(
			"%w: unsupported override_file_access_action %q",
			dbutil.ErrInvalidInput,
			action,
		)
	}
}

func clientModeFromProto(mode syncv1.ClientMode) configurations.ReportedClientMode {
	switch mode {
	case syncv1.ClientMode_MONITOR:
		return configurations.ReportedClientModeMonitor
	case syncv1.ClientMode_LOCKDOWN:
		return configurations.ReportedClientModeLockdown
	case syncv1.ClientMode_STANDALONE:
		return configurations.ReportedClientModeStandalone
	case syncv1.ClientMode_UNKNOWN_CLIENT_MODE:
		return configurations.ReportedClientModeUnknown
	default:
		return configurations.ReportedClientModeUnknown
	}
}

func protoClientMode(mode configurations.ClientMode) (syncv1.ClientMode, error) {
	switch mode {
	case configurations.ClientModeMonitor:
		return syncv1.ClientMode_MONITOR, nil
	case configurations.ClientModeLockdown:
		return syncv1.ClientMode_LOCKDOWN, nil
	case configurations.ClientModeStandalone:
		return syncv1.ClientMode_STANDALONE, nil
	default:
		return syncv1.ClientMode_UNKNOWN_CLIENT_MODE, fmt.Errorf(
			"%w: unsupported client_mode %q",
			dbutil.ErrInvalidInput,
			mode,
		)
	}
}

func protoSyncType(syncType syncstate.SyncType) (syncv1.SyncType, error) {
	switch syncType {
	case syncstate.SyncTypeNormal:
		return syncv1.SyncType_NORMAL, nil
	case syncstate.SyncTypeClean:
		return syncv1.SyncType_CLEAN, nil
	case syncstate.SyncTypeCleanAll:
		return syncv1.SyncType_CLEAN_ALL, nil
	default:
		return syncv1.SyncType_SYNC_TYPE_UNSPECIFIED, fmt.Errorf(
			"%w: unsupported sync_type %q",
			dbutil.ErrInvalidInput,
			syncType,
		)
	}
}
