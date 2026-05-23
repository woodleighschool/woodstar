package santa

import "time"

type ClientMode string

const (
	ClientModeUnknown    ClientMode = "unknown"
	ClientModeMonitor    ClientMode = "monitor"
	ClientModeLockdown   ClientMode = "lockdown"
	ClientModeStandalone ClientMode = "standalone"
)

// SyncToken is a Santa sync bearer token metadata record.
type SyncToken struct {
	ID         int64      `json:"id"`
	ValueHash  string     `json:"value_hash"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// CreatedSyncToken includes the one-time plaintext token value.
type CreatedSyncToken struct {
	SyncToken
	Value string `json:"value"`
}

// HostObservation is Santa-reported state for an existing Woodstar host.
type HostObservation struct {
	HostID             int64
	MachineID          string
	SerialNumber       string
	Version            string
	ClientModeReported ClientMode
	PrimaryUser        string
	PrimaryUserGroups  []string
	SIPStatus          *int16
	OSBuild            string
	ModelIdentifier    string
	LastSeenAt         *time.Time
}

// HostState is the Santa sub-object attached to host detail responses.
type HostState struct {
	Enrolled               bool                    `json:"enrolled"`
	Version                string                  `json:"version"`
	ClientModeReported     ClientMode              `json:"client_mode_reported"`
	LastSyncAt             *time.Time              `json:"last_sync_at,omitempty"`
	EffectiveConfiguration *EffectiveConfiguration `json:"effective_configuration,omitempty"`
	RuleSync               RuleSyncSummary         `json:"rule_sync"`
}

type EffectiveConfiguration struct {
	ID              int64         `json:"id"`
	Name            string        `json:"name"`
	MatchedViaLabel *MatchedLabel `json:"matched_via_label,omitempty"`
}

type MatchedLabel struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type RuleSyncSummary struct {
	DesiredCount    int        `json:"desired_count"`
	AppliedCount    int        `json:"applied_count"`
	PendingCount    int        `json:"pending_count"`
	LastCleanSyncAt *time.Time `json:"last_clean_sync_at,omitempty"`
}
