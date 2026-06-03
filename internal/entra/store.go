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

// ListUsers returns Entra-populated users for admin selectors.
func (s *Store) ListUsers(ctx context.Context, params ListParams) ([]EntraUser, int, error) {
	where, args := entraUserWhere(params)
	listQuery := entraUserListQuery(params, where, args)
	countSQL, countArgs := listQuery.BuildCount()
	var count int
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := listQuery.Build()
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (EntraUser, error) {
		var user EntraUser
		var entraID, userPrincipalName, mailNickname, givenName, familyName, department *string
		err := row.Scan(
			&user.ID,
			&entraID,
			&user.Email,
			&userPrincipalName,
			&mailNickname,
			&user.Name,
			&givenName,
			&familyName,
			&department,
			&user.Active,
			&user.LastSyncedAt,
		)
		user.EntraID = stringValue(entraID)
		user.UserPrincipalName = stringValue(userPrincipalName)
		user.MailNickname = stringValue(mailNickname)
		user.GivenName = stringValue(givenName)
		user.FamilyName = stringValue(familyName)
		user.Department = stringValue(department)
		return user, err
	})
	return users, count, err
}

// ListGroups returns Entra groups for admin selectors.
func (s *Store) ListGroups(ctx context.Context, params ListParams) ([]EntraGroup, int, error) {
	where, args := entraGroupWhere(params)
	listQuery := entraGroupListQuery(params, where, args)
	countSQL, countArgs := listQuery.BuildCount()
	var count int
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := listQuery.Build()
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	groups, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (EntraGroup, error) {
		var group EntraGroup
		var mailNickname *string
		err := row.Scan(&group.ID, &group.ExternalID, &group.DisplayName, &mailNickname, &group.LastSyncedAt)
		group.MailNickname = stringValue(mailNickname)
		return group, err
	})
	return groups, count, err
}

// ListDepartments returns distinct non-empty Entra-populated user departments.
func (s *Store) ListDepartments(ctx context.Context, params ListParams) ([]EntraDepartment, int, error) {
	where, args := entraDepartmentWhere(params)
	listQuery := entraDepartmentListQuery(params, where, args)
	countSQL, countArgs := listQuery.BuildCount()
	var count int
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := listQuery.Build()
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	departments, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (EntraDepartment, error) {
		var department EntraDepartment
		err := row.Scan(&department.Value)
		return department, err
	})
	return departments, count, err
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

func entraUserWhere(params ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	where.Add("entra_id IS NOT NULL")
	where.Add("last_synced_at IS NOT NULL")
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			user_principal_name ILIKE ` + search + `
			OR email ILIKE ` + search + `
			OR mail_nickname ILIKE ` + search + `
			OR name ILIKE ` + search + `
			OR department ILIKE ` + search + `
		)`)
	}
	if len(params.Values) > 0 {
		values := where.Arg(cleanValues(params.Values))
		where.Add("id::text = ANY(" + values + "::text[])")
	}
	return where.Build()
}

func entraGroupWhere(params ListParams) (string, []any) {
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

func entraDepartmentWhere(params ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	where.Add("entra_id IS NOT NULL")
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

func entraUserListQuery(params ListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: `SELECT id, entra_id, email, user_principal_name, mail_nickname, name,
	given_name, family_name, department, active, last_synced_at
FROM users`,
		WhereSQL: where,
		Args:     args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":                {SQL: "lower(name)"},
			"user_principal_name": {SQL: "lower(user_principal_name)"},
			"department":          {SQL: "lower(department)", NullOrder: dbutil.NullsLast},
			"last_synced_at":      {SQL: "last_synced_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(name)"}, {SQL: "id"}},
		Params:       params.ListParams,
	}
}

func entraGroupListQuery(params ListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: "SELECT id, external_id, display_name, mail_nickname, last_synced_at FROM entra_groups",
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"display_name":   {SQL: "lower(display_name)"},
			"mail_nickname":  {SQL: "lower(mail_nickname)", NullOrder: dbutil.NullsLast},
			"last_synced_at": {SQL: "last_synced_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(display_name)"}, {SQL: "id"}},
		Params:       params.ListParams,
	}
}

func entraDepartmentListQuery(params ListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: "SELECT DISTINCT department FROM users",
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"value": {SQL: "department"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "department"}},
		Params:       params.ListParams,
	}
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
