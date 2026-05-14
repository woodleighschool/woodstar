package directory

import (
	"context"
	"errors"
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

// NewStore returns a directory store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

// ListUsers returns every directory user ordered by UPN.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.q.ListDirectoryUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]User, len(rows))
	for i, row := range rows {
		out[i] = userFromSQLC(row)
	}
	return out, nil
}

// ListGroups returns every directory group ordered by display name.
func (s *Store) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := s.q.ListDirectoryGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Group, len(rows))
	for i, row := range rows {
		out[i] = groupFromSQLC(row)
	}
	return out, nil
}

// GetUserByUPN returns one directory user by user_principal_name.
func (s *Store) GetUserByUPN(ctx context.Context, upn string) (*User, error) {
	row, err := s.q.GetDirectoryUserByUPN(ctx, sqlc.GetDirectoryUserByUPNParams{UserPrincipalName: upn})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u := userFromSQLC(row)
	return &u, nil
}

// Apply reconciles the snapshot into the database within a single
// transaction: every user and group present in the snapshot is upserted,
// memberships are replaced per-user, and any rows whose external_id is no
// longer in the snapshot are hard-deleted (cascading through memberships
// and host_directory_user when that table exists).
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
				MailNickname: nilIfEmpty(g.MailNickname),
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
				Mail:              nilIfEmpty(u.Mail),
				MailNickname:      nilIfEmpty(u.MailNickname),
				DisplayName:       u.DisplayName,
				GivenName:         nilIfEmpty(u.GivenName),
				FamilyName:        nilIfEmpty(u.FamilyName),
				Department:        nilIfEmpty(u.Department),
				Active:            u.Active,
				LastSyncedAt:      syncedAt,
			})
			if err != nil {
				return err
			}
			if err := q.ReplaceDirectoryUserGroups(ctx, sqlc.ReplaceDirectoryUserGroupsParams{
				DirectoryUserID:   row.ID,
				GroupExternalIds:  u.GroupExternalIDs,
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
		return nil
	})
}

func userFromSQLC(s sqlc.DirectoryUser) User {
	return User{
		ID:                s.ID,
		ExternalID:        s.ExternalID,
		UserPrincipalName: s.UserPrincipalName,
		Mail:              stringPtrOrZero(s.Mail),
		MailNickname:      stringPtrOrZero(s.MailNickname),
		DisplayName:       s.DisplayName,
		GivenName:         stringPtrOrZero(s.GivenName),
		FamilyName:        stringPtrOrZero(s.FamilyName),
		Department:        stringPtrOrZero(s.Department),
		Active:            s.Active,
		LastSyncedAt:      s.LastSyncedAt,
	}
}

func groupFromSQLC(s sqlc.DirectoryGroup) Group {
	return Group{
		ID:           s.ID,
		ExternalID:   s.ExternalID,
		DisplayName:  s.DisplayName,
		MailNickname: stringPtrOrZero(s.MailNickname),
		LastSyncedAt: s.LastSyncedAt,
	}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringPtrOrZero(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
