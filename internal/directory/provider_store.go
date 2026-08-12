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
		if err := stageProviderSnapshot(ctx, tx, snapshot); err != nil {
			return err
		}
		if err := reconcileGroupSnapshot(ctx, tx, source); err != nil {
			return err
		}
		if err := reconcileUserSnapshot(ctx, tx, source); err != nil {
			return err
		}
		if err := reconcileGroupMembershipSnapshot(ctx, tx, source); err != nil {
			return err
		}
		return s.labels.RefreshDerivedTx(ctx, tx)
	})
}

func stageProviderSnapshot(ctx context.Context, tx pgx.Tx, snapshot ProviderSnapshot) error {
	if _, err := tx.Exec(ctx, `
CREATE TEMP TABLE IF NOT EXISTS directory_group_snapshot (
    position bigint NOT NULL,
    external_id text NOT NULL,
    display_name text NOT NULL,
    mail_nickname text
) ON COMMIT DELETE ROWS`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
CREATE TEMP TABLE IF NOT EXISTS directory_user_snapshot (
    position bigint NOT NULL,
    external_id text NOT NULL,
    user_principal_name text NOT NULL,
    mail text,
    mail_nickname text,
    display_name text NOT NULL,
    given_name text,
    family_name text,
    department text,
    enabled boolean NOT NULL,
    group_external_ids text[] NOT NULL
) ON COMMIT DELETE ROWS`); err != nil {
		return err
	}

	if len(snapshot.Groups) > 0 {
		if _, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{"directory_group_snapshot"},
			[]string{"position", "external_id", "display_name", "mail_nickname"},
			pgx.CopyFromSlice(len(snapshot.Groups), func(i int) ([]any, error) {
				group := snapshot.Groups[i]
				return []any{
					int64(i),
					group.ExternalID,
					group.DisplayName,
					postgres.NullString(group.MailNickname),
				}, nil
			}),
		); err != nil {
			return err
		}
	}

	if len(snapshot.Users) == 0 {
		return nil
	}
	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"directory_user_snapshot"},
		[]string{
			"position",
			"external_id",
			"user_principal_name",
			"mail",
			"mail_nickname",
			"display_name",
			"given_name",
			"family_name",
			"department",
			"enabled",
			"group_external_ids",
		},
		pgx.CopyFromSlice(len(snapshot.Users), func(i int) ([]any, error) {
			user := snapshot.Users[i]
			groupExternalIDs := user.GroupExternalIDs
			if groupExternalIDs == nil {
				groupExternalIDs = []string{}
			}
			return []any{
				int64(i),
				user.ExternalID,
				strings.TrimSpace(user.UserPrincipalName),
				postgres.NullString(strings.TrimSpace(user.Mail)),
				postgres.NullString(user.MailNickname),
				user.DisplayName,
				postgres.NullString(user.GivenName),
				postgres.NullString(user.FamilyName),
				postgres.NullString(user.Department),
				user.Enabled,
				groupExternalIDs,
			}, nil
		}),
	)
	return err
}

func reconcileGroupSnapshot(ctx context.Context, tx pgx.Tx, source Source) error {
	if _, err := tx.Exec(ctx, `
INSERT INTO directory_groups (source, external_id, display_name, mail_nickname)
SELECT
    $1::directory_source,
    external_id,
    display_name,
    mail_nickname
FROM (
    SELECT DISTINCT ON (external_id)
        position,
        external_id,
        display_name,
        mail_nickname
    FROM directory_group_snapshot
    ORDER BY external_id, position DESC
) snapshot
ON CONFLICT (source, external_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    mail_nickname = EXCLUDED.mail_nickname,
    updated_at = now()`, string(source)); err != nil {
		return err
	}

	_, err := tx.Exec(ctx, `
DELETE FROM directory_groups groups
WHERE groups.source = $1::directory_source
  AND NOT EXISTS (
      SELECT 1
      FROM directory_group_snapshot snapshot
      WHERE snapshot.external_id = groups.external_id
  )`, string(source))
	return err
}

func reconcileUserSnapshot(ctx context.Context, tx pgx.Tx, source Source) error {
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

	_, err := tx.Exec(ctx, `
INSERT INTO users (
    email, name, source, external_id, user_principal_name,
    mail_nickname, given_name, family_name, department, deleted_at
)
SELECT
    COALESCE(mail, user_principal_name),
    display_name,
    $1::directory_source,
    external_id,
    user_principal_name,
    mail_nickname,
    given_name,
    family_name,
    department,
    CASE WHEN enabled THEN NULL ELSE now() END
FROM (
    SELECT DISTINCT ON (external_id)
        position,
        external_id,
        user_principal_name,
        mail,
        mail_nickname,
        display_name,
        given_name,
        family_name,
        department,
        enabled
    FROM directory_user_snapshot
    ORDER BY external_id, position DESC
) snapshot
ON CONFLICT (source, external_id) DO UPDATE SET
    email = EXCLUDED.email,
    name = EXCLUDED.name,
    user_principal_name = EXCLUDED.user_principal_name,
    mail_nickname = EXCLUDED.mail_nickname,
    given_name = EXCLUDED.given_name,
    family_name = EXCLUDED.family_name,
    department = EXCLUDED.department,
    deleted_at = EXCLUDED.deleted_at,
    updated_at = now()`, string(source))
	return err
}

func reconcileGroupMembershipSnapshot(ctx context.Context, tx pgx.Tx, source Source) error {
	if _, err := tx.Exec(ctx, `
DELETE FROM directory_group_memberships membership
USING users, directory_groups groups
WHERE membership.user_id = users.id
  AND membership.group_id = groups.id
  AND users.source = $1::directory_source
  AND groups.source = $1::directory_source`, string(source)); err != nil {
		return err
	}

	_, err := tx.Exec(ctx, `
WITH current_users AS (
    SELECT DISTINCT ON (external_id)
        external_id,
        group_external_ids
    FROM directory_user_snapshot
    ORDER BY external_id, position DESC
)
INSERT INTO directory_group_memberships (user_id, group_id)
SELECT DISTINCT users.id, groups.id
FROM current_users snapshot
CROSS JOIN LATERAL unnest(snapshot.group_external_ids) AS membership(group_external_id)
JOIN users
  ON users.source = $1::directory_source
 AND users.external_id = snapshot.external_id
JOIN directory_groups groups
  ON groups.source = $1::directory_source
 AND groups.external_id = membership.group_external_id
ON CONFLICT DO NOTHING`, string(source))
	return err
}
