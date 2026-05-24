package syncstate

import (
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
)

type SyncType string

const (
	SyncTypeNormal SyncType = "normal"
	SyncTypeClean  SyncType = "clean"
)

type RuleCounts struct {
	Binary      int
	Certificate int
	TeamID      int
	SigningID   int
	CDHash      int
}

type PreflightRequest struct {
	SerialNumber      string
	Version           string
	ClientMode        configurations.ClientMode
	RequestCleanSync  bool
	RulesHash         string
	RuleCounts        RuleCounts
	PrimaryUser       string
	PrimaryUserGroups []string
	SIPStatus         *int16
	OSBuild           string
	ModelIdentifier   string
}

type PreflightResponse struct {
	SyncType      SyncType
	Configuration *configurations.Configuration
}

type EventUploadRequest struct {
	Events []events.ExecutionEventInput
}

type EventUploadResponse struct{}

type RuleDownloadRequest struct {
	Cursor string
}

type RuleDownloadResponse struct {
	Rules  []PayloadRule
	Cursor string
}

type PostflightRequest struct {
	RulesHash      string
	RulesReceived  int
	RulesProcessed int
}

type PostflightResponse struct{}
