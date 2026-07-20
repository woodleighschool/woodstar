package directory

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// ApplyProviderSnapshot reconciles a source-owned snapshot and derived label
// memberships in one transaction.
func (s *Store) ApplyProviderSnapshot(
	ctx context.Context,
	source Source,
	snapshot ProviderSnapshot,
) error {
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
		return s.labels.RefreshDerivedTx(ctx, tx)
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
		u.Mail = strings.TrimSpace(u.Mail)
		u.UserPrincipalName = strings.TrimSpace(u.UserPrincipalName)
		if err := upsertSnapshotUser(ctx, tx, source, u); err != nil {
			return err
		}
		userExternalIDs = append(userExternalIDs, u.ExternalID)
	}
	if err := removeMissingLocalUserLinks(ctx, tx, source, userExternalIDs); err != nil {
		return err
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

// upsertSnapshotUser reuses a deleted identity from the same provider, upserts
// the user row, then replaces its group memberships. Local identities remain
// local even when a provider reports the same email address.
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
	      source = $1::directory_source AND deleted_at IS NOT NULL
	  )
  AND email = COALESCE($3::text, $4::text)`,
			string(source), u.ExternalID, dbutil.NullString(u.Mail), u.UserPrincipalName,
		); err != nil {
			return err
		}
	}
	linked, err := upsertLocalSnapshotUser(ctx, tx, source, u)
	if err != nil || linked {
		return err
	}
	var userID int64
	err = tx.QueryRow(ctx, `
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

func upsertLocalSnapshotUser(
	ctx context.Context,
	tx pgx.Tx,
	source Source,
	u ProviderUser,
) (bool, error) {
	var userID int64
	err := tx.QueryRow(ctx, `
SELECT u.id
FROM users u
LEFT JOIN directory_user_links l
  ON l.user_id = u.id
 AND l.source = $1::directory_source
WHERE u.source = 'local'
  AND u.deleted_at IS NULL
  AND (
      l.external_id = $2
      OR u.email = COALESCE($3::text, $4::text)
  )
ORDER BY CASE WHEN l.external_id = $2 THEN 0 ELSE 1 END, u.id
LIMIT 1`,
		string(source), u.ExternalID, dbutil.NullString(u.Mail), u.UserPrincipalName,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !u.Enabled {
		return true, removeLocalUserLink(ctx, tx, source, userID)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO directory_user_links (user_id, source, external_id)
VALUES ($1, $2::directory_source, $3)
ON CONFLICT (user_id, source) DO UPDATE SET
    external_id = EXCLUDED.external_id,
    updated_at = now()`,
		userID, string(source), u.ExternalID,
	); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE users
SET
    user_principal_name = $2,
    mail_nickname = $3,
    given_name = $4,
    family_name = $5,
    department = $6,
    updated_at = now()
WHERE id = $1`,
		userID,
		u.UserPrincipalName,
		dbutil.NullString(u.MailNickname),
		dbutil.NullString(u.GivenName),
		dbutil.NullString(u.FamilyName),
		dbutil.NullString(u.Department),
	); err != nil {
		return false, err
	}
	return true, replaceUserGroupMemberships(ctx, tx, source, userID, u.GroupExternalIDs)
}

func removeMissingLocalUserLinks(
	ctx context.Context,
	tx pgx.Tx,
	source Source,
	externalIDs []string,
) error {
	_, err := tx.Exec(ctx, `
WITH removed AS (
    DELETE FROM directory_user_links
    WHERE source = $1::directory_source
      AND external_id <> ALL($2::text[])
    RETURNING user_id
), removed_memberships AS (
    DELETE FROM directory_group_memberships gm
    USING directory_groups g
    WHERE gm.group_id = g.id
      AND g.source = $1::directory_source
      AND gm.user_id IN (SELECT user_id FROM removed)
)
UPDATE users
SET
    user_principal_name = NULL,
    mail_nickname = NULL,
    given_name = NULL,
    family_name = NULL,
    department = NULL,
    updated_at = now()
WHERE id IN (SELECT user_id FROM removed)`,
		string(source), externalIDs,
	)
	return err
}

func removeLocalUserLink(ctx context.Context, tx pgx.Tx, source Source, userID int64) error {
	if _, err := tx.Exec(ctx, `
DELETE FROM directory_user_links
WHERE user_id = $1
  AND source = $2::directory_source`, userID, string(source)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM directory_group_memberships gm
USING directory_groups g
WHERE gm.group_id = g.id
  AND gm.user_id = $1
  AND g.source = $2::directory_source`, userID, string(source)); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
UPDATE users
SET
    user_principal_name = NULL,
    mail_nickname = NULL,
    given_name = NULL,
    family_name = NULL,
    department = NULL,
    updated_at = now()
WHERE id = $1`, userID)
	return err
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
	if _, err := tx.Exec(ctx, `
DELETE FROM directory_group_memberships gm
USING directory_groups g
WHERE gm.group_id = g.id
  AND gm.user_id = $1
  AND g.source = $2::directory_source`, userID, string(source)); err != nil {
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
