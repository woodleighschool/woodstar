// Package directory syncs people and groups from an identity provider.
package directory

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// User is one synced directory account.
type User struct {
	ID                int64     `json:"id"`
	ExternalID        string    `json:"external_id"`
	UserPrincipalName string    `json:"user_principal_name"`
	Mail              string    `json:"mail,omitempty"`
	MailNickname      string    `json:"mail_nickname,omitempty"`
	DisplayName       string    `json:"display_name"`
	GivenName         string    `json:"given_name,omitempty"`
	FamilyName        string    `json:"family_name,omitempty"`
	Department        string    `json:"department,omitempty"`
	Active            bool      `json:"active"`
	LastSyncedAt      time.Time `json:"last_synced_at"`
}

// Group is one synced directory group.
type Group struct {
	ID           int64     `json:"id"`
	ExternalID   string    `json:"external_id"`
	DisplayName  string    `json:"display_name"`
	MailNickname string    `json:"mail_nickname,omitempty"`
	LastSyncedAt time.Time `json:"last_synced_at"`
}

// Department is one non-empty department observed on synced directory users.
type Department struct {
	Value string `json:"value"`
}

// ListParams filters paginated directory selector lists.
type ListParams struct {
	dbutil.ListParams

	Values []string
}

// Snapshot is one provider sync result.
type Snapshot struct {
	Users       []SnapshotUser
	Groups      []SnapshotGroup
	GeneratedAt time.Time
}

// SnapshotUser is a synced user.
type SnapshotUser struct {
	ExternalID        string
	UserPrincipalName string
	Mail              string
	MailNickname      string
	DisplayName       string
	GivenName         string
	FamilyName        string
	Department        string
	Active            bool
	// GroupExternalIDs are this user's synced groups.
	GroupExternalIDs []string
}

// SnapshotGroup is a synced group.
type SnapshotGroup struct {
	ExternalID   string
	DisplayName  string
	MailNickname string
}
