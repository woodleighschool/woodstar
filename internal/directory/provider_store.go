package directory

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// ApplyProviderSnapshot reconciles a source-owned snapshot in one transaction.
func (s *Store) ApplyProviderSnapshot(ctx context.Context, source Source, snapshot ProviderSnapshot) error {
	if source == SourceLocal {
		return errors.New("directory: local source cannot apply provider snapshot")
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := applyGroupSnapshot(ctx, tx, source, snapshot.Groups); err != nil {
			return err
		}
		if err := applyUserSnapshot(ctx, tx, source, snapshot.Users); err != nil {
			return err
		}
		return reconcileLinks(ctx, tx)
	})
}

// applyGroupSnapshot upserts every snapshot group and deletes groups the
// source no longer reports.
func applyGroupSnapshot(ctx context.Context, tx pgx.Tx, source Source, groups []ProviderGroup) error {
	groupIDs := make([]string, 0, len(groups))
	for _, g := range groups {
		if _, err := tx.Exec(ctx, `
INSERT INTO directory_groups (source, external_id, display_name, mail_nickname)
VALUES ($1::directory_source, $2, $3, $4)
ON CONFLICT (source, external_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    mail_nickname = EXCLUDED.mail_nickname,
    updated_at = now()`,
			string(source), g.ExternalID, g.DisplayName, dbutil.NullString(g.MailNickname),
		); err != nil {
			return err
		}
		groupIDs = append(groupIDs, g.ExternalID)
	}
	_, err := tx.Exec(ctx, `
DELETE FROM directory_groups
WHERE source = $1::directory_source
  AND external_id <> ALL($2::text[])`,
		string(source), groupIDs,
	)
	return err
}

// applyUserSnapshot upserts every snapshot user with refreshed group
// memberships and soft-deletes users the source no longer reports.
func applyUserSnapshot(ctx context.Context, tx pgx.Tx, source Source, users []ProviderUser) error {
	userExternalIDs := make([]string, 0, len(users))
	for _, u := range users {
		if err := upsertSnapshotUser(ctx, tx, source, u); err != nil {
			return err
		}
		userExternalIDs = append(userExternalIDs, u.ExternalID)
	}
	_, err := tx.Exec(ctx, `
UPDATE users
SET
    deleted_at = now(),
    updated_at = now()
WHERE source = $1::directory_source
  AND deleted_at IS NULL
  AND external_id <> ALL($2::text[])`,
		string(source), userExternalIDs,
	)
	return err
}

// upsertSnapshotUser attaches an existing enabled user by email, upserts the
// user row, then replaces its group memberships.
func upsertSnapshotUser(ctx context.Context, tx pgx.Tx, source Source, u ProviderUser) error {
	if u.Enabled {
		if _, err := tx.Exec(ctx, `
UPDATE users
SET
    source = $1::directory_source,
    external_id = $2::text,
    deleted_at = NULL,
    updated_at = now()
WHERE (
      (source = 'local' AND deleted_at IS NULL)
      OR (source = $1::directory_source AND deleted_at IS NOT NULL)
  )
  AND (
      lower(email) = lower(COALESCE($3::text, $4::text))
      OR lower(COALESCE(user_principal_name, '')) = lower($4::text)
  )`,
			string(source), u.ExternalID, dbutil.NullString(u.Mail), u.UserPrincipalName,
		); err != nil {
			return err
		}
	}
	var userID int64
	err := tx.QueryRow(ctx, `
INSERT INTO users (
    email, name, source, external_id, user_principal_name,
    mail_nickname, given_name, family_name, department, deleted_at
)
VALUES (
    COALESCE($1::text, $2::text),
    $3::text,
    $4::directory_source,
    $5::text,
    $2::text,
    $6::text,
    $7::text,
    $8::text,
    $9::text,
    CASE WHEN $10::boolean THEN NULL ELSE now() END
)
ON CONFLICT (source, external_id) DO UPDATE SET
    email = EXCLUDED.email,
    name = EXCLUDED.name,
    user_principal_name = EXCLUDED.user_principal_name,
    mail_nickname = EXCLUDED.mail_nickname,
    given_name = EXCLUDED.given_name,
    family_name = EXCLUDED.family_name,
    department = EXCLUDED.department,
    deleted_at = EXCLUDED.deleted_at,
    updated_at = now()
RETURNING id`,
		dbutil.NullString(u.Mail), u.UserPrincipalName,
		u.DisplayName,
		string(source), u.ExternalID,
		dbutil.NullString(u.MailNickname),
		dbutil.NullString(u.GivenName),
		dbutil.NullString(u.FamilyName),
		dbutil.NullString(u.Department),
		u.Enabled,
	).Scan(&userID)
	if err != nil {
		return err
	}
	return replaceUserGroupMemberships(ctx, tx, source, userID, u.GroupExternalIDs)
}

// replaceUserGroupMemberships clears a user's group memberships and inserts the
// snapshot set.
func replaceUserGroupMemberships(
	ctx context.Context,
	tx pgx.Tx,
	source Source,
	userID int64,
	groupExternalIDs []string,
) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM directory_group_memberships WHERE user_id = $1`,
		userID,
	); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
INSERT INTO directory_group_memberships (user_id, group_id)
SELECT $1, g.id
FROM directory_groups g
WHERE g.source = $2::directory_source
  AND g.external_id = ANY($3::text[])
ON CONFLICT DO NOTHING`,
		userID, string(source), groupExternalIDs,
	)
	return err
}
