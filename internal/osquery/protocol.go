// Package osquery implements service behavior for Orbit-managed osquery.
package osquery

import (
	"encoding/json"
)

// EnrollRequest is the body posted by osquery to enroll.
type EnrollRequest struct {
	EnrollSecret   string                       `json:"enroll_secret"`
	HostIdentifier string                       `json:"host_identifier"`
	HostDetails    map[string]map[string]string `json:"host_details"`
}

// EnrollResponse is returned to osquery after enrollment.
type EnrollResponse struct {
	NodeKey     string `json:"node_key,omitempty"`
	NodeInvalid bool   `json:"node_invalid"`
}

// ConfigRequest carries the osquery node key.
type ConfigRequest struct {
	NodeKey string `json:"node_key"`
}

// ConfigResponse is a minimal osquery config.
type ConfigResponse struct {
	NodeInvalid bool                     `json:"node_invalid"`
	Schedule    map[string]ScheduleEntry `json:"schedule"`
	Options     map[string]string        `json:"options"`
	Decorators  map[string][]string      `json:"decorators"`
}

// DistributedReadRequest asks for work.
type DistributedReadRequest struct {
	NodeKey string `json:"node_key"`
}

// DistributedReadResponse returns distributed detail queries.
type DistributedReadResponse struct {
	NodeInvalid bool              `json:"node_invalid"`
	Queries     map[string]string `json:"queries"`
	Discovery   map[string]string `json:"discovery"`
	Accelerate  uint              `json:"accelerate"`
}

// DistributedWriteRequest contains query results from osquery.
type DistributedWriteRequest struct {
	NodeKey  string                         `json:"node_key"`
	Queries  map[string][]map[string]string `json:"queries"`
	Statuses map[string]json.RawMessage     `json:"statuses"`
	Messages map[string]string              `json:"messages"`
}

// DistributedWriteResponse acknowledges query results.
type DistributedWriteResponse struct {
	NodeInvalid bool `json:"node_invalid"`
}

// LogRequest contains osquery logs.
type LogRequest struct {
	NodeKey string          `json:"node_key"`
	LogType string          `json:"log_type"`
	Data    json.RawMessage `json:"data"`
}

// LogResponse acknowledges osquery logs.
type LogResponse struct {
	NodeInvalid bool `json:"node_invalid"`
}
