// Package munki coordinates client observations and desired package state.
package munki

import "time"

// HostObservation is Munki state observed for an existing host.
type HostObservation struct {
	HostID          int64
	Version         string
	ManifestName    string
	Errors          []string
	Warnings        []string
	ProblemInstalls []string
	RunStartedAt    *time.Time
	RunEndedAt      *time.Time
}

// Item is one Munki-managed item observed on a host.
type Item struct {
	HostID           int64  `json:"-"`
	Name             string `json:"name"`
	Installed        bool   `json:"installed"`
	InstalledVersion string `json:"installed_version"`
}

// HostState is the Munki sub-object attached to host detail responses.
type HostState struct {
	Version         string     `json:"version"`
	ManifestName    string     `json:"manifest_name"`
	Errors          []string   `json:"errors"`
	Warnings        []string   `json:"warnings"`
	ProblemInstalls []string   `json:"problem_installs"`
	RunStartedAt    *time.Time `json:"run_started_at,omitempty"`
	RunEndedAt      *time.Time `json:"run_ended_at,omitempty"`
	Items           []Item     `json:"items"`
}
