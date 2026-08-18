// Package history stores the small set of historical osquery status aggregates.
package history

import "time"

// BucketInterval is the fixed resolution of osquery history.
const BucketInterval = 5 * time.Minute

// HostStatusPoint is one sampled online/offline host total.
type HostStatusPoint struct {
	Bucket       time.Time `json:"bucket"`
	OnlineCount  int32     `json:"online_count"`
	OfflineCount int32     `json:"offline_count"`
}

// PolicyStatusPoint is one sampled result total for a policy.
type PolicyStatusPoint struct {
	Bucket       time.Time `json:"bucket"`
	PassCount    int32     `json:"pass_count"`
	FailCount    int32     `json:"fail_count"`
	ErrorCount   int32     `json:"error_count"`
	PendingCount int32     `json:"pending_count"`
}

// CleanupResult reports how many historical points were removed.
type CleanupResult struct {
	HostPoints   int `json:"host_points"`
	PolicyPoints int `json:"policy_points"`
}
