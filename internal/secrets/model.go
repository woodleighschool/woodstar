package secrets

import "time"

// Secret is a reusable shared credential shown to admins.
type Secret struct {
	ID        int64     `json:"id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}
