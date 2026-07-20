package syncstate

import "time"

type santaPendingStateRow struct {
	PendingPayloadRuleCount uint32
	PendingFullSync         bool
	PendingPreflightAt      *time.Time
	PreflightRulesHash      string
}
