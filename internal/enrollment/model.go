package enrollment

import "time"

// EnrollSecret is a reusable enrollment credential accepted by Orbit and osquery.
type EnrollSecret struct {
	ID        int64     `json:"id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}
