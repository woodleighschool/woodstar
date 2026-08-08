package munki

import "time"

// ReportState describes the freshness of the host's Munki report.
type ReportState string

const (
	ReportNeverCollected   ReportState = "never_collected"
	ReportNoReport         ReportState = "no_report"
	ReportCurrent          ReportState = "current"
	ReportCollectionFailed ReportState = "collection_failed"
)

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

// QueryResult is one osquery result in the Munki detail-query family.
type QueryResult struct {
	Present bool
	Status  int
	Message string
	Rows    []map[string]string
}

// EnvelopeInput is the complete Munki detail-query family observed in one
// osquery distributed-write pass.
type EnvelopeInput struct {
	Info     QueryResult
	Installs QueryResult
}

// EnvelopeResult is one authoritative Munki collection attempt for a host.
type EnvelopeResult struct {
	HostID          int64
	AttemptedAt     time.Time
	Complete        bool
	CollectionError string
	HasReport       bool
	Observation     HostObservation
	Items           []ItemObservation
}

// HostState is the latest Munki report state for a host.
type HostState struct {
	ReportState      ReportState `json:"report_state"`
	LastAttemptAt    *time.Time  `db:"last_attempt_at" json:"last_attempt_at,omitempty"`
	LastSuccessfulAt *time.Time  `db:"last_successful_at" json:"last_successful_at,omitempty"`
	CollectionError  string      `db:"collection_error" json:"collection_error,omitempty"`
	HasReport        bool        `db:"has_report" json:"-"`
	Version          string      `db:"version" json:"version"`
	ManifestName     string      `db:"manifest_name" json:"manifest_name"`
	Errors           []string    `db:"errors" json:"errors"`
	Warnings         []string    `db:"warnings" json:"warnings"`
	ProblemInstalls  []string    `db:"problem_installs" json:"problem_installs"`
	RunStartedAt     *time.Time  `db:"run_started_at" json:"run_started_at,omitempty"`
	RunEndedAt       *time.Time  `db:"run_ended_at" json:"run_ended_at,omitempty"`
}
