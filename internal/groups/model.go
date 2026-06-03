// Package groups exposes synced directory groups as Woodstar resources.
package groups

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Group is one synced directory group.
type Group struct {
	ID           int64     `json:"id"`
	ExternalID   string    `json:"external_id"`
	DisplayName  string    `json:"display_name"`
	MailNickname string    `json:"mail_nickname,omitempty"`
	MemberCount  int       `json:"member_count"`
	LastSyncedAt time.Time `json:"last_synced_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ListParams filters paginated group lists.
type ListParams struct {
	dbutil.ListParams

	Values []string
}
