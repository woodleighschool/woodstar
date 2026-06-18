package directory

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// ApplyProviderSnapshot reconciles a source-owned snapshot in one transaction.
func (s *Store) ApplyProviderSnapshot(ctx context.Context, source Source, snapshot ProviderSnapshot) error {
	if source == SourceLocal {
		return errors.New("directory: local source cannot apply provider snapshot")
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := applyGroupSnapshot(ctx, q, source, snapshot.Groups); err != nil {
			return err
		}
		if err := applyUserSnapshot(ctx, q, source, snapshot.Users); err != nil {
			return err
		}
		return reconcileLinks(ctx, q)
	})
}

// applyGroupSnapshot upserts every snapshot group and deletes groups the
// source no longer reports.
func applyGroupSnapshot(ctx context.Context, q *sqlc.Queries, source Source, groups []ProviderGroup) error {
	groupIDs := make([]string, 0, len(groups))
	for _, g := range groups {
		if _, err := q.UpsertDirectoryGroup(ctx, sqlc.UpsertDirectoryGroupParams{
			Source:       sqlc.DirectorySource(source),
			ExternalID:   g.ExternalID,
			DisplayName:  g.DisplayName,
			MailNickname: dbutil.NullString(g.MailNickname),
		}); err != nil {
			return err
		}
		groupIDs = append(groupIDs, g.ExternalID)
	}
	return q.DeleteDirectoryGroupsNotIn(ctx, sqlc.DeleteDirectoryGroupsNotInParams{
		Source:      sqlc.DirectorySource(source),
		ExternalIds: groupIDs,
	})
}

// applyUserSnapshot upserts every snapshot user with refreshed group
// memberships and soft-deletes users the source no longer reports.
func applyUserSnapshot(ctx context.Context, q *sqlc.Queries, source Source, users []ProviderUser) error {
	userExternalIDs := make([]string, 0, len(users))
	for _, u := range users {
		if err := upsertSnapshotUser(ctx, q, source, u); err != nil {
			return err
		}
		userExternalIDs = append(userExternalIDs, u.ExternalID)
	}
	return q.SoftDeleteDirectoryUsersNotIn(ctx, sqlc.SoftDeleteDirectoryUsersNotInParams{
		Source:      sqlc.DirectorySource(source),
		ExternalIds: userExternalIDs,
	})
}

// upsertSnapshotUser attaches an existing enabled user by email, upserts the
// user row, then replaces its group memberships.
func upsertSnapshotUser(ctx context.Context, q *sqlc.Queries, source Source, u ProviderUser) error {
	if u.Enabled {
		if err := q.AttachDirectoryUserByEmail(ctx, sqlc.AttachDirectoryUserByEmailParams{
			Source:            sqlc.DirectorySource(source),
			ExternalID:        u.ExternalID,
			Mail:              dbutil.NullString(u.Mail),
			UserPrincipalName: u.UserPrincipalName,
		}); err != nil {
			return err
		}
	}
	row, err := q.UpsertDirectoryUser(ctx, sqlc.UpsertDirectoryUserParams{
		Source:            sqlc.DirectorySource(source),
		ExternalID:        u.ExternalID,
		UserPrincipalName: u.UserPrincipalName,
		Mail:              dbutil.NullString(u.Mail),
		MailNickname:      dbutil.NullString(u.MailNickname),
		DisplayName:       u.DisplayName,
		GivenName:         dbutil.NullString(u.GivenName),
		FamilyName:        dbutil.NullString(u.FamilyName),
		Department:        dbutil.NullString(u.Department),
		Enabled:           u.Enabled,
	})
	if err != nil {
		return err
	}
	return replaceUserGroupMemberships(ctx, q, source, row.ID, u.GroupExternalIDs)
}

// replaceUserGroupMemberships clears a user's group memberships and inserts the
// snapshot set.
func replaceUserGroupMemberships(
	ctx context.Context,
	q *sqlc.Queries,
	source Source,
	userID int64,
	groupExternalIDs []string,
) error {
	if err := q.DeleteDirectoryGroupMembershipsForUser(
		ctx,
		sqlc.DeleteDirectoryGroupMembershipsForUserParams{UserID: userID},
	); err != nil {
		return err
	}
	return q.InsertDirectoryGroupMemberships(ctx, sqlc.InsertDirectoryGroupMembershipsParams{
		Source:           sqlc.DirectorySource(source),
		UserID:           userID,
		GroupExternalIds: groupExternalIDs,
	})
}
