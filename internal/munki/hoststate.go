package munki

import "time"

type hostObservation struct {
	Version         string
	ManifestName    string
	Errors          []string
	Warnings        []string
	ProblemInstalls []string
	RunStartedAt    *time.Time
	RunEndedAt      *time.Time
}

type itemObservation struct {
	Name             string
	DisplayName      string
	Installed        bool
	InstalledVersion string
	TargetVersion    string
}

// QueryResult is one member of the Munki detail-query collection.
type QueryResult struct {
	Present    bool
	Successful bool
	Rows       []map[string]string
}

// Collection contains the Munki detail-query results from one osquery write.
type Collection struct {
	Info     QueryResult
	Installs QueryResult
}

type collectionUpdate struct {
	HostID      int64
	HasReport   bool
	Observation hostObservation
	Items       []itemObservation
}

// HostState is the latest Munki report for a host.
type HostState struct {
	Version         string     `db:"version"          json:"version"`
	ManifestName    string     `db:"manifest_name"    json:"manifest_name"`
	Errors          []string   `db:"errors"           json:"errors"`
	Warnings        []string   `db:"warnings"         json:"warnings"`
	ProblemInstalls []string   `db:"problem_installs" json:"problem_installs"`
	RunAt           *time.Time `db:"run_at"           json:"run_at,omitempty"`
}
