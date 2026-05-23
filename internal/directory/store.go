package directory

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists directory users and groups synced from an external IdP.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

// Apply reconciles the snapshot into the database within a single
// transaction: every user and group present in the snapshot is upserted,
// memberships are replaced per-user, matching host links are refreshed, and
// any rows whose external_id is no longer in the snapshot are hard-deleted
// (cascading through memberships and host_directory_user when that table
// exists).
func (s *Store) Apply(ctx context.Context, snapshot Snapshot) error {
	syncedAt := snapshot.GeneratedAt
	if syncedAt.IsZero() {
		syncedAt = time.Now().UTC()
	}

	return pgx.BeginFunc(ctx, s.db.Pool(), func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)

		groupIDs := make([]string, 0, len(snapshot.Groups))
		for _, g := range snapshot.Groups {
			if _, err := q.UpsertDirectoryGroup(ctx, sqlc.UpsertDirectoryGroupParams{
				ExternalID:   g.ExternalID,
				DisplayName:  g.DisplayName,
				MailNickname: dbutil.StringPtrOrNil(g.MailNickname),
				LastSyncedAt: syncedAt,
			}); err != nil {
				return err
			}
			groupIDs = append(groupIDs, g.ExternalID)
		}
		if err := q.DeleteDirectoryGroupsNotIn(ctx, sqlc.DeleteDirectoryGroupsNotInParams{
			ExternalIds: groupIDs,
		}); err != nil {
			return err
		}

		userIDs := make([]string, 0, len(snapshot.Users))
		for _, u := range snapshot.Users {
			row, err := q.UpsertDirectoryUser(ctx, sqlc.UpsertDirectoryUserParams{
				ExternalID:        u.ExternalID,
				UserPrincipalName: u.UserPrincipalName,
				Mail:              dbutil.StringPtrOrNil(u.Mail),
				MailNickname:      dbutil.StringPtrOrNil(u.MailNickname),
				DisplayName:       u.DisplayName,
				GivenName:         dbutil.StringPtrOrNil(u.GivenName),
				FamilyName:        dbutil.StringPtrOrNil(u.FamilyName),
				Department:        dbutil.StringPtrOrNil(u.Department),
				Active:            u.Active,
				LastSyncedAt:      syncedAt,
			})
			if err != nil {
				return err
			}
			if err := q.ReplaceDirectoryUserGroups(ctx, sqlc.ReplaceDirectoryUserGroupsParams{
				DirectoryUserID:  row.ID,
				GroupExternalIds: u.GroupExternalIDs,
			}); err != nil {
				return err
			}
			userIDs = append(userIDs, u.ExternalID)
		}
		if err := q.DeleteDirectoryUsersNotIn(ctx, sqlc.DeleteDirectoryUsersNotInParams{
			ExternalIds: userIDs,
		}); err != nil {
			return err
		}
		return reconcileLinks(ctx, q)
	})
}
