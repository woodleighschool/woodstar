package hosts

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ReplaceAccounts(ctx context.Context, hostID int64, accounts []HostAccount) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT id FROM hosts WHERE id = $1 FOR UPDATE`, hostID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM host_accounts WHERE host_id = $1`, hostID); err != nil {
			return err
		}
		_, err := tx.CopyFrom(ctx, pgx.Identifier{"host_accounts"}, []string{
			"host_id", "uid", "username", "uuid", "display_name", "home_directory", "shell",
			"is_admin", "is_hidden", "filevault_unlock", "secure_token", "volume_owner", "logged_in",
		}, pgx.CopyFromSlice(len(accounts), func(i int) ([]any, error) {
			a := accounts[i]
			return []any{hostID, a.UID, a.Username, a.UUID, a.DisplayName, a.HomeDirectory, a.Shell,
				a.IsAdmin, a.IsHidden, a.FileVaultUnlock, a.SecureToken, a.VolumeOwner, a.LoggedIn}, nil
		}))
		return err
	})
}

func (s *Store) ListAccounts(ctx context.Context, hostID int64) ([]HostAccount, error) {
	rows, err := s.pool.Query(ctx, `
SELECT uid, username, uuid, display_name, home_directory, shell,
    is_admin, is_hidden, filevault_unlock, secure_token, volume_owner, logged_in
FROM host_accounts
WHERE host_id = $1
ORDER BY username, uid`, hostID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[HostAccount])
}
