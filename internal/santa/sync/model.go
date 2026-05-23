package sync

import "time"

// SyncToken is a reusable Santa sync bearer token.
type SyncToken struct {
	ID        int64     `json:"id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}
