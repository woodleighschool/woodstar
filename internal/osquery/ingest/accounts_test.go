package ingest

import (
	"maps"
	"testing"

	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
)

func TestAccountSecurityObservations(t *testing.T) {
	base := []map[string]string{
		{"uid": "501", "username": "alex", "uuid": "ABC", "is_admin": "1", "is_hidden": "1", "logged_in": "1"},
		{"uid": "502", "username": "sam", "uuid": "DEF", "is_admin": "0", "is_hidden": "0", "logged_in": "0"},
		{"uid": "0", "username": "root", "is_admin": "0", "is_hidden": "0"},
	}
	tests := []struct {
		name                    string
		optional                map[string][]map[string]string
		filevault, token, owner *bool
	}{
		{name: "unsupported or failed"},
		{name: "successful empty FileVault", optional: map[string][]map[string]string{catalog.QueryAccountFileVault: {}}, filevault: new(false)},
		{name: "empty APFS enumeration", optional: map[string][]map[string]string{catalog.QueryAccountAPFS: {}}},
		{name: "recovery user only", optional: map[string][]map[string]string{catalog.QueryAccountAPFS: {{"uuid": "recovery", "type": "PersonalRecovery"}}}, token: new(false), owner: new(false)},
		{name: "nonlocal UUID match", optional: map[string][]map[string]string{catalog.QueryAccountAPFS: {{"uuid": "abc", "type": "PersonalRecovery", "volume_owner": "1"}}}, token: new(false), owner: new(false)},
		{name: "matching UUID", optional: map[string][]map[string]string{
			catalog.QueryAccountFileVault: {{"uuid": "abc"}},
			catalog.QueryAccountAPFS:      {{"uuid": "abc", "type": "LocalOpenDirectory", "volume_owner": "0"}},
		}, filevault: new(true), token: new(true), owner: new(false)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := map[string][]map[string]string{catalog.QueryAccounts: base}
			maps.Copy(rows, tt.optional)
			accounts := parseHostAccounts(rows)
			if len(accounts) != 3 || !accounts[0].IsAdmin || !accounts[0].IsHidden {
				t.Fatalf("accounts = %+v", accounts)
			}
			if !accounts[0].LoggedIn || accounts[1].LoggedIn || accounts[2].LoggedIn {
				t.Fatalf("logged-in accounts = %+v", accounts)
			}
			for _, field := range []struct {
				name      string
				got, want *bool
			}{
				{"FileVault", accounts[0].FileVaultUnlock, tt.filevault},
				{"Secure Token", accounts[0].SecureToken, tt.token},
				{"volume owner", accounts[0].VolumeOwner, tt.owner},
			} {
				if (field.got == nil) != (field.want == nil) || field.got != nil && *field.got != *field.want {
					t.Errorf("%s = %v, want %v", field.name, field.got, field.want)
				}
			}
			if accounts[2].SecureToken != nil || accounts[2].FileVaultUnlock != nil || accounts[2].VolumeOwner != nil {
				t.Fatal("account without UUID must have unknown security state")
			}
		})
	}
}
