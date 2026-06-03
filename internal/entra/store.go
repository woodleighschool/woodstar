package entra

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists Entra group data and applies Entra user enrichment.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

// Apply reconciles the snapshot into the database within a single transaction.
// Missing Entra users are marked inactive instead of deleted.
func (s *Store) Apply(ctx context.Context, snapshot Snapshot) error {
	syncedAt := snapshot.GeneratedAt
	if syncedAt.IsZero() {
		syncedAt = time.Now().UTC()
	}

	return pgx.BeginFunc(ctx, s.db.Pool(), func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)

		groupIDs := make([]string, 0, len(snapshot.Groups))
		for _, g := range snapshot.Groups {
			if _, err := q.UpsertEntraGroup(ctx, sqlc.UpsertEntraGroupParams{
				ExternalID:   g.ExternalID,
				DisplayName:  g.DisplayName,
				MailNickname: dbutil.NullString(g.MailNickname),
				LastSyncedAt: syncedAt,
			}); err != nil {
				return err
			}
			groupIDs = append(groupIDs, g.ExternalID)
		}
		if err := q.DeleteEntraGroupsNotIn(ctx, sqlc.DeleteEntraGroupsNotInParams{
			ExternalIds: groupIDs,
		}); err != nil {
			return err
		}

		userExternalIDs := make([]string, 0, len(snapshot.Users))
		for _, u := range snapshot.Users {
			if err := q.AttachEntraUserByEmail(ctx, sqlc.AttachEntraUserByEmailParams{
				ExternalID:        u.ExternalID,
				Mail:              dbutil.NullString(u.Mail),
				UserPrincipalName: u.UserPrincipalName,
			}); err != nil {
				return err
			}
			row, err := q.UpsertEntraUser(ctx, sqlc.UpsertEntraUserParams{
				ExternalID:        u.ExternalID,
				UserPrincipalName: u.UserPrincipalName,
				Mail:              dbutil.NullString(u.Mail),
				MailNickname:      dbutil.NullString(u.MailNickname),
				DisplayName:       u.DisplayName,
				GivenName:         dbutil.NullString(u.GivenName),
				FamilyName:        dbutil.NullString(u.FamilyName),
				Department:        dbutil.NullString(u.Department),
				Active:            u.Active,
				LastSyncedAt:      syncedAt,
			})
			if err != nil {
				return err
			}
			if err := q.DeleteEntraGroupMembershipsForUser(
				ctx,
				sqlc.DeleteEntraGroupMembershipsForUserParams{UserID: row.ID},
			); err != nil {
				return err
			}
			if err := q.InsertEntraGroupMemberships(ctx, sqlc.InsertEntraGroupMembershipsParams{
				UserID:           row.ID,
				GroupExternalIds: u.GroupExternalIDs,
			}); err != nil {
				return err
			}
			userExternalIDs = append(userExternalIDs, u.ExternalID)
		}
		if err := q.MarkEntraUsersInactiveNotIn(ctx, sqlc.MarkEntraUsersInactiveNotInParams{
			ExternalIds: userExternalIDs,
		}); err != nil {
			return err
		}
		return reconcileLinks(ctx, q)
	})
}
