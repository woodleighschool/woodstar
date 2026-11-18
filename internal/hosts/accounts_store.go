package hosts

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/postgres"
)

func (s *Store) ReplaceAccounts(ctx context.Context, hostID int64, accounts []HostAccount) error {
	seen := make(map[[2]string]struct{}, len(accounts))
	for _, account := range accounts {
		key := [2]string{account.UID, account.Username}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate host account %q with UID %q", account.Username, account.UID)
		}
		seen[key] = struct{}{}
	}
	return s.reconcileHostInventory(ctx, hostID, reconcileAccountsSQL, postgres.JSONSlice[HostAccount](accounts))
}

const reconcileAccountsSQL = `
WITH observed AS (
    SELECT uid, username, uuid, display_name, home_directory, shell, is_admin, is_hidden,
           filevault_unlock, secure_token, volume_owner, logged_in
    FROM jsonb_populate_recordset(NULL::host_accounts, $2::jsonb)
), removed AS (
    DELETE FROM host_accounts t
    WHERE t.host_id = $1
      AND NOT EXISTS (SELECT FROM observed s WHERE t.uid = s.uid AND t.username = s.username)
)
MERGE INTO host_accounts t
USING observed s ON t.host_id = $1 AND t.uid = s.uid AND t.username = s.username
WHEN MATCHED AND (
    t.uuid, t.display_name, t.home_directory, t.shell, t.is_admin, t.is_hidden,
    t.filevault_unlock, t.secure_token, t.volume_owner, t.logged_in
) IS DISTINCT FROM (
    s.uuid, s.display_name, s.home_directory, s.shell, s.is_admin, s.is_hidden,
    s.filevault_unlock, s.secure_token, s.volume_owner, s.logged_in
) THEN UPDATE SET
    (uuid, display_name, home_directory, shell, is_admin, is_hidden, filevault_unlock,
     secure_token, volume_owner, logged_in) =
    (s.uuid, s.display_name, s.home_directory, s.shell, s.is_admin, s.is_hidden,
     s.filevault_unlock, s.secure_token, s.volume_owner, s.logged_in)
WHEN NOT MATCHED THEN INSERT (
    host_id, uid, username, uuid, display_name, home_directory, shell, is_admin, is_hidden,
    filevault_unlock, secure_token, volume_owner, logged_in
) VALUES (
    $1, s.uid, s.username, s.uuid, s.display_name, s.home_directory, s.shell, s.is_admin,
    s.is_hidden, s.filevault_unlock, s.secure_token, s.volume_owner, s.logged_in
)`

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
