package santa

import "time"

// SyncToken is a Santa sync bearer token metadata record.
type SyncToken struct {
	ID         int64      `json:"id"`
	ValueHash  string     `json:"value_hash"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// CreatedSyncToken includes the one-time plaintext token value.
type CreatedSyncToken struct {
	SyncToken
	Value string `json:"value"`
}
