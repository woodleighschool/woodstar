package sync

import (
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
)

type SyncType string

const (
	SyncTypeNormal SyncType = "normal"
	SyncTypeClean  SyncType = "clean"
)

type PreflightRequest struct {
	SerialNumber      string
	Version           string
	ClientMode        configurations.ClientMode
	RequestCleanSync  bool
	RulesHash         string
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

type RuleDownloadRequest struct{}

type RuleDownloadResponse struct {
	Rules []Target
}

type PostflightRequest struct {
	RulesHash      string
	RulesReceived  int
	RulesProcessed int
}

type PostflightResponse struct{}
