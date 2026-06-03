// Package entra syncs Microsoft Entra users and groups into Woodstar.
package entra

import (
	"time"
)

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
