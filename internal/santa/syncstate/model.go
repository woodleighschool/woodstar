package syncstate

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
)

// SyncToken is a reusable Santa sync bearer token.
type SyncToken struct {
	ID        int64     `json:"id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

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

type RuleDownloadRequest struct {
	Cursor string
}

type RuleDownloadResponse struct {
	Rules  []Target
	Cursor string
}

type PostflightRequest struct {
	RulesHash      string
	RulesReceived  int
	RulesProcessed int
}

type PostflightResponse struct{}
