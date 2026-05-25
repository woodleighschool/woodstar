package santa

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

// HostObservation is Santa state for an existing host.
type HostObservation struct {
	HostID             int64
	MachineID          string
	SerialNumber       string
	Version            string
	ClientModeReported configurations.ClientMode
	PrimaryUser        string
	PrimaryUserGroups  []string
	SIPStatus          *int16
	OSBuild            string
	ModelIdentifier    string
	LastSeenAt         *time.Time
}

// HostState is the Santa sub-object attached to host detail responses.
type HostState struct {
	Version                string                                `json:"version"`
	ClientModeReported     configurations.ClientMode             `json:"client_mode_reported"              enum:"unknown,monitor,lockdown,standalone"`
	LastSyncAt             *time.Time                            `json:"last_sync_at,omitempty"`
	EffectiveConfiguration *configurations.ResolvedConfiguration `json:"effective_configuration,omitempty"`
	RuleSync               RuleSyncSummary                       `json:"rule_sync"`
}

type RuleSyncSummary struct {
	DesiredCount    int        `json:"desired_count"`
	AppliedCount    int        `json:"applied_count"`
	PendingCount    int        `json:"pending_count"`
	LastCleanSyncAt *time.Time `json:"last_clean_sync_at,omitempty"`
}
