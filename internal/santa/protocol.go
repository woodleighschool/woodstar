package santa

import (
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

type PreflightRequest struct {
	SerialNumber      string
	Version           string
	ClientMode        configurations.ReportedClientMode
	RequestCleanSync  bool
	RulesHash         string
	RuleCounts        syncstate.RuleCounts
	PrimaryUser       string
	PrimaryUserGroups []string
	SIPStatus         *int16
	OSBuild           string
	ModelIdentifier   string
}

type PreflightResponse struct {
	SyncType      syncstate.SyncType
	Configuration *configurations.Configuration
}

type EventUploadRequest struct {
	Events           []events.ExecutionEventInput
	FileAccessEvents []events.FileAccessEventInput
}

type EventUploadResponse struct{}

type RuleDownloadRequest struct {
	Cursor string
}

type RuleDownloadResponse = syncstate.PayloadRulePage

type PostflightRequest struct {
	RulesHash      string
	RulesReceived  int
	RulesProcessed int
}

type PostflightResponse struct{}
