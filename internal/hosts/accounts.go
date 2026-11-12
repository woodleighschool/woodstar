package hosts

// HostAccount is a local account in the latest inventory snapshot.
type HostAccount struct {
	UID             string `json:"uid" db:"uid"`
	Username        string `json:"username" db:"username"`
	UUID            string `json:"uuid" db:"uuid"`
	DisplayName     string `json:"display_name" db:"display_name"`
	HomeDirectory   string `json:"home_directory" db:"home_directory"`
	Shell           string `json:"shell" db:"shell"`
	IsAdmin         bool   `json:"is_admin" db:"is_admin"`
	IsHidden        bool   `json:"is_hidden" db:"is_hidden"`
	FileVaultUnlock *bool  `json:"filevault_unlock" db:"filevault_unlock"`
	SecureToken     *bool  `json:"secure_token" db:"secure_token"`
	VolumeOwner     *bool  `json:"volume_owner" db:"volume_owner"`
	LoggedIn        bool   `json:"logged_in" db:"logged_in"`
}
