package directory

import (
	"time"
)

// ProviderSnapshot is one source-owned directory object snapshot.
type ProviderSnapshot struct {
	Users       []ProviderUser
	Groups      []ProviderGroup
	GeneratedAt time.Time
}

// ProviderUser is a source-owned directory user.
type ProviderUser struct {
	ExternalID        string
	UserPrincipalName string
	Mail              string
	MailNickname      string
	DisplayName       string
	GivenName         string
	FamilyName        string
	Department        string
	Enabled           bool
	// GroupExternalIDs are this user's source-owned directory groups.
	GroupExternalIDs []string
}

// ProviderGroup is a source-owned directory group.
type ProviderGroup struct {
	ExternalID   string
	DisplayName  string
	MailNickname string
}
