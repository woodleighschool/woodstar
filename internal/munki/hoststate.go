// Package munki coordinates Munki client observations and repository delivery.
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

// ItemObservation is one Munki-managed item reported by a host.
type ItemObservation struct {
	HostID           int64
	Name             string
	DisplayName      string
	Installed        bool
	InstalledVersion string
	TargetVersion    string
}

// HostState is the latest Munki run summary reported for a host.
type HostState struct {
	Version         string     `db:"version"          json:"version"`
	ManifestName    string     `db:"manifest_name"    json:"manifest_name"`
	Errors          []string   `db:"errors"           json:"errors"`
	Warnings        []string   `db:"warnings"         json:"warnings"`
	ProblemInstalls []string   `db:"problem_installs" json:"problem_installs"`
	RunStartedAt    *time.Time `db:"run_started_at"   json:"run_started_at,omitempty"`
	RunEndedAt      *time.Time `db:"run_ended_at"     json:"run_ended_at,omitempty"`
}
