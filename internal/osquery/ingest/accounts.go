package ingest

import (
	"context"
	"strings"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
)

func (p *Projector) IngestAccounts(ctx context.Context, hostID int64, rows map[string][]map[string]string) error {
	return p.hostStore.ReplaceAccounts(ctx, hostID, parseHostAccounts(rows))
}

func parseHostAccounts(rows map[string][]map[string]string) []hosts.HostAccount {
	fileVaultRows, fileVaultKnown := rows[catalog.QueryAccountFileVault]
	fileVault := make(map[string]bool, len(fileVaultRows))
	for _, row := range fileVaultRows {
		fileVault[strings.ToUpper(row["uuid"])] = true
	}
	cryptoUsers := make(map[string]*bool)
	apfsKnown := false
	for _, row := range rows[catalog.QueryAccountAPFS] {
		uuid := row["uuid"]
		if uuid == "" {
			continue
		}
		// The extension can return no rows when crypto-user enumeration fails.
		apfsKnown = true
		if row["type"] == "LocalOpenDirectory" {
			var owner *bool
			if value := row["volume_owner"]; value == "0" || value == "1" {
				owner = new(value == "1")
			}
			cryptoUsers[strings.ToUpper(uuid)] = owner
		}
	}
	accounts := make([]hosts.HostAccount, 0, len(rows[catalog.QueryAccounts]))
	for _, row := range rows[catalog.QueryAccounts] {
		if row["uid"] == "" || row["username"] == "" {
			continue
		}
		account := hosts.HostAccount{
			UID: row["uid"], Username: row["username"], UUID: row["uuid"],
			DisplayName: row["display_name"], HomeDirectory: row["home_directory"], Shell: row["shell"],
			IsAdmin: row["is_admin"] == "1", IsHidden: row["is_hidden"] == "1",
			LoggedIn: row["logged_in"] == "1",
		}
		if uuid := strings.ToUpper(account.UUID); uuid != "" {
			if fileVaultKnown {
				account.FileVaultUnlock = new(fileVault[uuid])
			}
			if apfsKnown {
				owner, token := cryptoUsers[uuid]
				account.SecureToken = new(token)
				account.VolumeOwner = owner
				if !token {
					account.VolumeOwner = new(false)
				}
			}
		}
		accounts = append(accounts, account)
	}
	return accounts
}
