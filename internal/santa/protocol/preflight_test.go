package protocol

import (
	"testing"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"

	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

func TestPreflightResponseOmitsSettingsWithoutConfiguration(t *testing.T) {
	t.Parallel()

	response, err := preflightResponseToProto(santa.PreflightResponse{
		SyncType: syncstate.SyncTypeClean,
	})
	if err != nil {
		t.Fatalf("encode preflight response: %v", err)
	}
	if response.GetClientMode() != 0 || response.EnableBundles != nil ||
		response.EnableTransitiveRules != nil || response.EnableAllEventUpload != nil ||
		response.DisableUnknownEventUpload != nil || response.OverrideFileAccessAction != nil ||
		response.GetFullSyncIntervalSeconds() != 0 || response.GetBatchSize() != 0 ||
		response.AllowedPathRegex != nil || response.BlockedPathRegex != nil ||
		response.GetRemovableMediaPolicy() != nil || response.GetEncryptedRemovableMediaPolicy() != nil ||
		response.EventDetailUrl != nil || response.EventDetailText != nil {
		t.Fatalf("undefined configuration settings were encoded: %+v", response)
	}
}

func TestPreflightResponseEmitsConcreteConfigurationSettings(t *testing.T) {
	t.Parallel()

	response, err := preflightResponseToProto(santa.PreflightResponse{
		SyncType: syncstate.SyncTypeClean,
		Configuration: &configurations.Configuration{
			ClientMode:               configurations.ClientModeMonitor,
			OverrideFileAccessAction: configurations.FileAccessActionNone,
			FullSyncIntervalSeconds:  600,
			BatchSize:                50,
		},
	})
	if err != nil {
		t.Fatalf("encode preflight response: %v", err)
	}
	if response.EnableBundles == nil || response.GetEnableBundles() ||
		response.EnableTransitiveRules == nil || response.GetEnableTransitiveRules() ||
		response.EnableAllEventUpload == nil || response.GetEnableAllEventUpload() ||
		response.DisableUnknownEventUpload == nil || response.GetDisableUnknownEventUpload() {
		t.Fatalf("concrete false settings lost presence: %+v", response)
	}
	if response.GetClientMode() != syncv1.ClientMode_MONITOR ||
		response.OverrideFileAccessAction == nil ||
		response.GetOverrideFileAccessAction() != syncv1.FileAccessAction_NONE ||
		response.GetFullSyncIntervalSeconds() != 600 || response.GetBatchSize() != 50 {
		t.Fatalf("concrete scalar settings were not encoded: %+v", response)
	}
}
