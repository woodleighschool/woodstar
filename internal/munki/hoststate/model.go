package hoststate

import "time"

// Observation is Munki state observed for an existing host.
type Observation struct {
	HostID          int64
	Version         string
	ManifestName    string
	Success         *bool
	Errors          []string
	Warnings        []string
	ProblemInstalls []string
	RunStartedAt    string
	RunEndedAt      string
}

// Item is one Munki-managed item observed on a host.
type Item struct {
	HostID           int64     `json:"-"`
	Name             string    `json:"name"`
	Installed        bool      `json:"installed"`
	InstalledVersion string    `json:"installed_version"`
	RunEndedAt       string    `json:"run_ended_at,omitempty"`
	LastSeenAt       time.Time `json:"last_seen_at"`
}

// State is the Munki sub-object attached to host detail responses.
type State struct {
	Version         string    `json:"version"`
	ManifestName    string    `json:"manifest_name"`
	Success         *bool     `json:"success,omitempty"`
	Errors          []string  `json:"errors"`
	Warnings        []string  `json:"warnings"`
	ProblemInstalls []string  `json:"problem_installs"`
	RunStartedAt    string    `json:"run_started_at,omitempty"`
	RunEndedAt      string    `json:"run_ended_at,omitempty"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	Items           []Item    `json:"items"`
}
