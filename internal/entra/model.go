// Package entra syncs Microsoft Entra users and groups into Woodstar.
package entra

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// EntraUser is one canonical user populated by Entra.
type EntraUser struct {
	ID                int64     `json:"id"`
	EntraID           string    `json:"entra_id"`
	Email             string    `json:"email" format:"email"`
	UserPrincipalName string    `json:"user_principal_name"`
	MailNickname      string    `json:"mail_nickname,omitempty"`
	Name              string    `json:"name"`
	GivenName         string    `json:"given_name,omitempty"`
	FamilyName        string    `json:"family_name,omitempty"`
	Department        string    `json:"department,omitempty"`
	Active            bool      `json:"active"`
	LastSyncedAt      time.Time `json:"last_synced_at"`
}

// EntraGroup is one synced Entra group.
type EntraGroup struct {
	ID           int64     `json:"id"`
	ExternalID   string    `json:"external_id"`
	DisplayName  string    `json:"display_name"`
	MailNickname string    `json:"mail_nickname,omitempty"`
	LastSyncedAt time.Time `json:"last_synced_at"`
}

// EntraDepartment is one non-empty department observed on Entra-populated users.
type EntraDepartment struct {
	Value string `json:"value"`
}

// ListParams filters paginated Entra selector lists.
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
