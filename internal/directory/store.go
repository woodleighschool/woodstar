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

// ListUsers returns directory users for admin selectors.
func (s *Store) ListUsers(ctx context.Context, params ListParams) ([]User, int, error) {
	where, args := directoryUserWhere(params)
	var count int
	if err := s.db.Pool().
		QueryRow(ctx, "SELECT count(*)::integer FROM directory_users "+where, args...).
		Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := directoryUserListSQL(params, where, args)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) {
		var user User
		var mail, mailNickname, givenName, familyName, department *string
		err := row.Scan(
			&user.ID,
			&user.ExternalID,
			&user.UserPrincipalName,
			&mail,
			&mailNickname,
			&user.DisplayName,
			&givenName,
			&familyName,
			&department,
			&user.Active,
			&user.LastSyncedAt,
		)
		user.Mail = stringValue(mail)
		user.MailNickname = stringValue(mailNickname)
		user.GivenName = stringValue(givenName)
		user.FamilyName = stringValue(familyName)
		user.Department = stringValue(department)
		return user, err
	})
	return users, count, err
}

// ListGroups returns directory groups for admin selectors.
func (s *Store) ListGroups(ctx context.Context, params ListParams) ([]Group, int, error) {
	where, args := directoryGroupWhere(params)
	var count int
	if err := s.db.Pool().
		QueryRow(ctx, "SELECT count(*)::integer FROM directory_groups "+where, args...).
		Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := directoryGroupListSQL(params, where, args)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	groups, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Group, error) {
		var group Group
		var mailNickname *string
		err := row.Scan(&group.ID, &group.ExternalID, &group.DisplayName, &mailNickname, &group.LastSyncedAt)
		group.MailNickname = stringValue(mailNickname)
		return group, err
	})
	return groups, count, err
}

// ListDepartments returns distinct non-empty directory departments for admin selectors.
func (s *Store) ListDepartments(ctx context.Context, params ListParams) ([]Department, int, error) {
	where, args := directoryDepartmentWhere(params)
	countSQL := "SELECT count(*)::integer FROM (SELECT DISTINCT department FROM directory_users " + where + ") d"
	var count int
	if err := s.db.Pool().QueryRow(ctx, countSQL, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := directoryDepartmentListSQL(params, where, args)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	departments, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Department, error) {
		var department Department
		err := row.Scan(&department.Value)
		return department, err
	})
	return departments, count, err
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
				MailNickname: dbutil.NullString(g.MailNickname),
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

func directoryUserWhere(params ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			user_principal_name ILIKE ` + search + `
			OR mail ILIKE ` + search + `
			OR mail_nickname ILIKE ` + search + `
			OR display_name ILIKE ` + search + `
			OR department ILIKE ` + search + `
		)`)
	}
	if len(params.Values) > 0 {
		values := where.Arg(cleanValues(params.Values))
		where.Add("external_id = ANY(" + values + "::text[])")
	}
	return where.Build()
}

func directoryGroupWhere(params ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			display_name ILIKE ` + search + `
			OR mail_nickname ILIKE ` + search + `
		)`)
	}
	if len(params.Values) > 0 {
		values := where.Arg(cleanValues(params.Values))
		where.Add("external_id = ANY(" + values + "::text[])")
	}
	return where.Build()
}

func directoryDepartmentWhere(params ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	where.Add("NULLIF(btrim(department), '') IS NOT NULL")
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add("department ILIKE " + search)
	}
	if len(params.Values) > 0 {
		values := where.Arg(cleanValues(params.Values))
		where.Add("department = ANY(" + values + "::text[])")
	}
	return where.Build()
}

func directoryUserListSQL(params ListParams, where string, args []any) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL: `SELECT id, external_id, user_principal_name, mail, mail_nickname, display_name,
	given_name, family_name, department, active, last_synced_at
FROM directory_users`,
		WhereSQL: where,
		Args:     args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"display_name":        {SQL: "lower(display_name)"},
			"user_principal_name": {SQL: "lower(user_principal_name)"},
			"department":          {SQL: "lower(department)", NullOrder: dbutil.NullsLast},
			"last_synced_at":      {SQL: "last_synced_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(display_name)"}, {SQL: "id"}},
		Params:       params.ListParams,
	}.Build()
}

func directoryGroupListSQL(params ListParams, where string, args []any) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL: "SELECT id, external_id, display_name, mail_nickname, last_synced_at FROM directory_groups",
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"display_name":   {SQL: "lower(display_name)"},
			"mail_nickname":  {SQL: "lower(mail_nickname)", NullOrder: dbutil.NullsLast},
			"last_synced_at": {SQL: "last_synced_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(display_name)"}, {SQL: "id"}},
		Params:       params.ListParams,
	}.Build()
}

func directoryDepartmentListSQL(params ListParams, where string, args []any) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL: "SELECT DISTINCT department FROM directory_users",
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"value": {SQL: "department"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "department"}},
		Params:       params.ListParams,
	}.Build()
}

func cleanValues(values []string) []string {
	return dbutil.SplitListValues(values)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
