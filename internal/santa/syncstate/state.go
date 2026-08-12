package syncstate

type santaPendingStateRow struct {
	PendingSyncType     SyncType
	PendingPolicyDigest string
	PreflightRulesHash  string
}
