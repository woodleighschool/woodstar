package directory

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Group is one directory group.
type Group struct {
	ID           int64     `json:"id"`
	Source       Source    `json:"source"`
	ExternalID   string    `json:"external_id"`
	DisplayName  string    `json:"display_name"`
	MailNickname string    `json:"mail_nickname,omitempty"`
	MemberCount  int       `json:"member_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GroupListParams filters paginated group lists.
type GroupListParams struct {
	dbutil.ListParams

	Values []string
}
