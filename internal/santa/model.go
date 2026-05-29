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
	ClientModeReported configurations.ReportedClientMode
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
	ClientModeReported     configurations.ReportedClientMode     `json:"client_mode_reported"`
	LastSyncAt             *time.Time                            `json:"last_sync_at,omitempty"`
	EffectiveConfiguration *configurations.ResolvedConfiguration `json:"effective_configuration,omitempty"`
	RuleSync               RuleSyncSummary                       `json:"rule_sync"`
}

type RuleSyncSummary struct {
	DesiredCount    int32      `json:"desired_count"`
	AppliedCount    int32      `json:"applied_count"`
	PendingCount    int32      `json:"pending_count"`
	LastCleanSyncAt *time.Time `json:"last_clean_sync_at,omitempty"`
}
