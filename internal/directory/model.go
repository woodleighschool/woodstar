// Package directory mirrors human identities and group memberships from an
// external identity provider (Entra ID for the MVP) into local tables so
// the rest of Woodstar can target hosts by directory metadata. Directory
// users are observed state separate from local Woodstar accounts; they
// never receive UI access.
package directory

import "time"

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

// Snapshot is the result of a single Entra fetch, fed into Service.Apply.
type Snapshot struct {
	Users       []SnapshotUser
	Groups      []SnapshotGroup
	GeneratedAt time.Time
}

// SnapshotUser is the provider-facing user shape (group memberships nested).
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
	// GroupExternalIDs are the IDs of every directory group the user
	// belongs to, direct or transitive depending on the sync mode.
	GroupExternalIDs []string
}

// SnapshotGroup is the provider-facing group shape.
type SnapshotGroup struct {
	ExternalID   string
	DisplayName  string
	MailNickname string
}
