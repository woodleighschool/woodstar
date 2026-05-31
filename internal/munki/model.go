package munki

import "time"

// HostStatusObservation is Munki state observed for an existing host.
type HostStatusObservation struct {
	HostID          int64
	Version         string
	ManifestName    string
	ConsoleUser     string
	Success         *bool
	Errors          []string
	Warnings        []string
	ProblemInstalls []string
	RunStartedAt    string
	RunEndedAt      string
}

// HostItem is one Munki-managed item observed on a host.
type HostItem struct {
	HostID           int64     `json:"-"`
	Name             string    `json:"name"`
	Installed        bool      `json:"installed"`
	InstalledVersion string    `json:"installed_version"`
	RunEndedAt       string    `json:"run_ended_at,omitempty"`
	LastSeenAt       time.Time `json:"last_seen_at"`
}

// HostMunkiState is the Munki sub-object attached to host detail responses.
type HostMunkiState struct {
	Version         string     `json:"version"`
	ManifestName    string     `json:"manifest_name"`
	ConsoleUser     string     `json:"console_user,omitempty"`
	Success         *bool      `json:"success,omitempty"`
	Errors          []string   `json:"errors"`
	Warnings        []string   `json:"warnings"`
	ProblemInstalls []string   `json:"problem_installs"`
	RunStartedAt    string     `json:"run_started_at,omitempty"`
	RunEndedAt      string     `json:"run_ended_at,omitempty"`
	LastSeenAt      time.Time  `json:"last_seen_at"`
	Items           []HostItem `json:"items"`
}
