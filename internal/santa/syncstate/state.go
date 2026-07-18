package syncstate

import "time"

type santaPendingStateRow struct {
	PendingPayloadRuleCount int32
	PendingFullSync         bool
	PendingPreflightAt      *time.Time
	PreflightRulesHash      string
}
