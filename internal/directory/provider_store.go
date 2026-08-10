package directory

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/postgres"
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
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
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
			string(source), g.ExternalID, g.DisplayName, postgres.NullString(g.MailNickname),
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
	normalizedUsers := make([]ProviderUser, len(users))
	for i, u := range users {
		u.Mail = strings.TrimSpace(u.Mail)
		u.UserPrincipalName = strings.TrimSpace(u.UserPrincipalName)
		normalizedUsers[i] = u
	}

	// Provider object IDs own identity. Tombstoning the current generation first
	// lets current attributes move between objects without reassigning old rows.
	if _, err := tx.Exec(ctx, `
UPDATE users
SET
    deleted_at = now(),
    updated_at = now()
WHERE source = $1::directory_source
  AND deleted_at IS NULL`, string(source)); err != nil {
		return err
	}

	for _, u := range normalizedUsers {
		if err := upsertSnapshotUser(ctx, tx, source, u); err != nil {
			return err
		}
	}
	return nil
}

// upsertSnapshotUser upserts by provider object ID, then replaces the user's
// group memberships. Provider objects never merge into local password users.
func upsertSnapshotUser(ctx context.Context, tx pgx.Tx, source Source, u ProviderUser) error {
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
		postgres.NullString(u.Mail), u.UserPrincipalName,
		u.DisplayName,
		string(source), u.ExternalID,
		postgres.NullString(u.MailNickname),
		postgres.NullString(u.GivenName),
		postgres.NullString(u.FamilyName),
		postgres.NullString(u.Department),
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
