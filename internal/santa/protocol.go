package santa

import (
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

type PreflightRequest struct {
	SerialNumber      string
	Version           string
	RulesHash         string
	ClientMode        configurations.ReportedClientMode
	RequestCleanSync  bool
	RuleCounts        syncstate.RuleCounts
	PrimaryUser       string
	PrimaryUserGroups []string
	SIPStatus         *int16
}

type PreflightResponse struct {
	SyncType      syncstate.SyncType
	Configuration *configurations.Configuration
}

type EventUploadRequest struct {
	Events                       []events.ExecutionEventInput
	FileAccessEvents             []events.FileAccessEventInput
	StandaloneRuleCreationEvents []events.StandaloneRuleCreationEventInput
}

type EventUploadResponse struct {
	BundleBinaryRequests []string
}

type RuleDownloadRequest struct {
	Cursor string
}

type RuleDownloadResponse = syncstate.PayloadRulePage

type PostflightRequest struct {
	RulesReceived  int32
	RulesProcessed int32
	SyncType       syncstate.SyncType
	RulesHash      string
}

type PostflightResponse struct{}
