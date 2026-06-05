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
	return pgx.BeginFunc(ctx, s.db.Pool(), func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)

		groupIDs := make([]string, 0, len(snapshot.Groups))
		for _, g := range snapshot.Groups {
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
		if err := q.DeleteDirectoryGroupsNotIn(ctx, sqlc.DeleteDirectoryGroupsNotInParams{
			Source:      sqlc.DirectorySource(source),
			ExternalIds: groupIDs,
		}); err != nil {
			return err
		}

		userExternalIDs := make([]string, 0, len(snapshot.Users))
		for _, u := range snapshot.Users {
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
			if err := q.DeleteDirectoryGroupMembershipsForUser(
				ctx,
				sqlc.DeleteDirectoryGroupMembershipsForUserParams{UserID: row.ID},
			); err != nil {
				return err
			}
			if err := q.InsertDirectoryGroupMemberships(ctx, sqlc.InsertDirectoryGroupMembershipsParams{
				Source:           sqlc.DirectorySource(source),
				UserID:           row.ID,
				GroupExternalIds: u.GroupExternalIDs,
			}); err != nil {
				return err
			}
			userExternalIDs = append(userExternalIDs, u.ExternalID)
		}
		if err := q.SoftDeleteDirectoryUsersNotIn(ctx, sqlc.SoftDeleteDirectoryUsersNotInParams{
			Source:      sqlc.DirectorySource(source),
			ExternalIds: userExternalIDs,
		}); err != nil {
			return err
		}
		return reconcileLinks(ctx, q)
	})
}
