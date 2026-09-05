package directory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

// Store persists directory users, groups, memberships, and source snapshots.
type Store struct {
	pool   *pgxpool.Pool
	labels derivedLabelRefresher
}

type derivedLabelRefresher interface {
	RefreshDerivedTx(ctx context.Context, tx pgx.Tx) error
}

// NewStore returns a directory store backed by pool.
func NewStore(pool *pgxpool.Pool, labelRefresher derivedLabelRefresher) *Store {
	return &Store{pool: pool, labels: labelRefresher}
}

type userRow struct {
	ID                int64      `db:"id"`
	Email             string     `db:"email"`
	Name              string     `db:"name"`
	PasswordHash      *string    `db:"password_hash"`
	Role              *string    `db:"role"`
	APIKey            *string    `db:"api_key"`
	APIKeyCreatedAt   *time.Time `db:"api_key_created_at"`
	Source            string     `db:"source"`
	ExternalID        *string    `db:"external_id"`
	UserPrincipalName *string    `db:"user_principal_name"`
	MailNickname      *string    `db:"mail_nickname"`
	GivenName         *string    `db:"given_name"`
	FamilyName        *string    `db:"family_name"`
	Department        *string    `db:"department"`
	DeletedAt         *time.Time `db:"deleted_at"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

func userColumnsSQL(alias string) string {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	columns := []string{
		prefix + "id",
		prefix + "email",
		prefix + "name",
		prefix + "password_hash",
		directRoleKeySQL(prefix+"id") + " AS role",
		prefix + "api_key",
		prefix + "api_key_created_at",
		prefix + "source::text AS source",
		prefix + "external_id",
		prefix + "user_principal_name",
		prefix + "mail_nickname",
		prefix + "given_name",
		prefix + "family_name",
		prefix + "department",
		prefix + "deleted_at",
		prefix + "created_at",
		prefix + "updated_at",
	}
	return strings.Join(columns, ", ")
}

func directRoleKeySQL(userIDExpression string) string {
	return `(SELECT role.key
FROM authz_user_roles AS assignment
JOIN authz_roles AS role ON role.id = assignment.role_id
WHERE assignment.user_id = ` + userIDExpression + `)`
}

func effectiveRoleExistsSQL(userIDExpression string) string {
	return `EXISTS (
    SELECT 1 FROM authz_user_roles AS direct_role WHERE direct_role.user_id = ` + userIDExpression + `
    UNION ALL
    SELECT 1
    FROM directory_group_memberships AS membership
    JOIN authz_group_roles AS group_role ON group_role.group_id = membership.group_id
    WHERE membership.user_id = ` + userIDExpression + `
)`
}

func effectiveRoleKeyExistsSQL(userIDExpression string, roleKeysExpression string) string {
	return `EXISTS (
    SELECT 1
    FROM authz_user_roles AS direct_role
    JOIN authz_roles AS role ON role.id = direct_role.role_id
    WHERE direct_role.user_id = ` + userIDExpression + `
      AND role.key = ANY(` + roleKeysExpression + `::text[])
    UNION ALL
    SELECT 1
    FROM directory_group_memberships AS membership
    JOIN authz_group_roles AS group_role ON group_role.group_id = membership.group_id
    JOIN authz_roles AS role ON role.id = group_role.role_id
    WHERE membership.user_id = ` + userIDExpression + `
      AND role.key = ANY(` + roleKeysExpression + `::text[])
)`
}

func userSelectSQL() string {
	return `
SELECT
	` + userColumnsSQL("u") + `
FROM users AS u`
}

func userFromRow(r userRow) User {
	role := roleFromString(r.Role)
	return User{
		ID:                r.ID,
		Email:             r.Email,
		Name:              r.Name,
		PasswordHash:      derefString(r.PasswordHash),
		Role:              role,
		Source:            Source(r.Source),
		ExternalID:        derefString(r.ExternalID),
		UserPrincipalName: derefString(r.UserPrincipalName),
		MailNickname:      derefString(r.MailNickname),
		GivenName:         derefString(r.GivenName),
		FamilyName:        derefString(r.FamilyName),
		Department:        derefString(r.Department),
		DeletedAt:         r.DeletedAt,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func accountFromRow(r userRow) Account {
	account := Account{
		User:            userFromRow(r),
		APIKeyCreatedAt: r.APIKeyCreatedAt,
	}
	if r.APIKey != nil {
		account.APIKey = *r.APIKey
	}
	return account
}

func roleFromString(role *string) *Role {
	if role == nil {
		return nil
	}
	value := Role(*role)
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type userCreateRecord struct {
	Email        string
	Name         string
	PasswordHash string
	Role         Role
}

func (s *Store) createUser(
	ctx context.Context,
	params userCreateRecord,
) (*User, error) {
	var user User
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var userID int64
		if err := tx.QueryRow(ctx, `
INSERT INTO users (email, name, password_hash, source)
VALUES ($1, $2, $3, 'local')
RETURNING id`, params.Email, params.Name, params.PasswordHash).Scan(&userID); err != nil {
			return postgres.MutationError(err)
		}
		if err := replaceUserRole(ctx, tx, userID, &params.Role); err != nil {
			return err
		}
		row, err := postgres.GetOne[userRow](ctx, tx, userSelectSQL()+`
WHERE u.id = $1`, userID)
		if err != nil {
			return err
		}
		user = userFromRow(row)
		return s.labels.RefreshDerivedTx(ctx, tx)
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row, err := postgres.GetOne[userRow](ctx, s.pool, userSelectSQL()+`
WHERE id = $1
  AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	out := userFromRow(row)
	return &out, nil
}

// GetAccountByID returns the signed-in user's self-view, including API key fields.
func (s *Store) GetAccountByID(ctx context.Context, id int64) (*Account, error) {
	row, err := postgres.GetOne[userRow](ctx, s.pool, userSelectSQL()+`
WHERE id = $1
  AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, postgres.GetError(err)
	}
	out := accountFromRow(row)
	return &out, nil
}

func (s *Store) ListUsers(ctx context.Context, params UserListParams) ([]User, int, error) {
	where, args := userWhere(params)
	rows, count, err := postgres.ListWithCount[userRow](ctx, s.pool, userListQuery(params, where, args))
	if err != nil {
		return nil, 0, err
	}
	out := make([]User, len(rows))
	for i, row := range rows {
		out[i] = userFromRow(row)
	}
	return out, count, nil
}

func (s *Store) ListDepartments(ctx context.Context, params UserListParams) ([]Department, int, error) {
	where, args := departmentWhere(params)
	return postgres.ListWithCount[Department](ctx, s.pool, departmentListQuery(params, where, args))
}

type userUpdateRecord struct {
	Name         string
	Role         *Role
	PasswordHash *string
}

func (s *Store) updateUser(ctx context.Context, id int64, params userUpdateRecord) (*User, error) {
	var user User
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var userID int64
		if err := tx.QueryRow(ctx, `
UPDATE users
SET
    name = CASE WHEN source = 'local' THEN $1 ELSE name END,
    password_hash = CASE
		WHEN source = 'local' THEN COALESCE($2, password_hash)
        ELSE password_hash
    END,
    updated_at = now()
WHERE id = $3
  AND deleted_at IS NULL
RETURNING id`, params.Name, params.PasswordHash, id).Scan(&userID); err != nil {
			return postgres.MutationError(err)
		}
		if err := replaceUserRole(ctx, tx, userID, params.Role); err != nil {
			return err
		}
		row, err := postgres.GetOne[userRow](ctx, tx, userSelectSQL()+`
WHERE u.id = $1`, userID)
		if err != nil {
			return err
		}
		user = userFromRow(row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) setLocalUserPasswordByEmail(
	ctx context.Context,
	email string,
	passwordHash string,
) (*User, error) {
	qrows, err := s.pool.Query(ctx, `
UPDATE users
SET
    password_hash = $1,
    updated_at = now()
WHERE email = $2
  AND source = 'local'
  AND deleted_at IS NULL
RETURNING `+userColumnsSQL("users"), passwordHash, email)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	row, err := pgx.CollectExactlyOneRow(qrows, pgx.RowToStructByName[userRow])
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	user := userFromRow(row)
	return &user, nil
}

func (s *Store) setUserRoleByEmail(ctx context.Context, email string, role Role) (*User, error) {
	var userID int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
SELECT id
FROM users
WHERE lower(email) = lower($1)
  AND deleted_at IS NULL
ORDER BY CASE WHEN source = 'local' THEN 1 ELSE 0 END, id
LIMIT 1
FOR UPDATE`, email).Scan(&userID); err != nil {
			return postgres.GetError(err)
		}
		if err := replaceUserRole(ctx, tx, userID, &role); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE users SET updated_at = now() WHERE id = $1`, userID)
		return postgres.MutationError(err)
	})
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, userID)
}

func replaceUserRole(ctx context.Context, tx pgx.Tx, userID int64, role *Role) error {
	if _, err := tx.Exec(ctx, `DELETE FROM authz_user_roles WHERE user_id = $1`, userID); err != nil {
		return postgres.MutationError(err)
	}
	if role == nil {
		return nil
	}
	tag, err := tx.Exec(ctx, `
INSERT INTO authz_user_roles (user_id, role_id)
SELECT $1, id
FROM authz_roles
WHERE key = $2`, userID, string(*role))
	if err != nil {
		return postgres.MutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: unknown role %q", fault.ErrInvalidInput, *role)
	}
	return nil
}

func (s *Store) updateAccount(ctx context.Context, id int64, params accountUpdateRecord) (*Account, error) {
	qrows, err := s.pool.Query(ctx, `
UPDATE users
SET
    name = CASE WHEN source = 'local' THEN $1 ELSE name END,
    password_hash = CASE
        WHEN source = 'local' THEN COALESCE($2, password_hash)
        ELSE password_hash
    END,
    updated_at = now()
WHERE id = $3
RETURNING `+userColumnsSQL("users"),
		params.Name, params.PasswordHash, id)
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	row, err := pgx.CollectExactlyOneRow(qrows, pgx.RowToStructByName[userRow])
	if err != nil {
		return nil, postgres.MutationError(err)
	}
	out := accountFromRow(row)
	return &out, nil
}

func (s *Store) deleteUser(
	ctx context.Context,
	id int64,
) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var source Source
		if err := tx.QueryRow(ctx, `
SELECT source::text
FROM users
WHERE id = $1
  AND deleted_at IS NULL
FOR UPDATE`, id).Scan(&source); err != nil {
			return postgres.GetError(err)
		}

		var deletedID int64
		if source == SourceLocal {
			if err := tx.QueryRow(ctx, `
DELETE FROM users
WHERE id = $1
RETURNING id`, id).Scan(&deletedID); err != nil {
				return postgres.MutationError(err)
			}
		} else {
			if err := tx.QueryRow(ctx, `
UPDATE users
SET
    deleted_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING id`, id).Scan(&deletedID); err != nil {
				return postgres.MutationError(err)
			}
		}
		return s.labels.RefreshDerivedTx(ctx, tx)
	})
}

func userWhere(params UserListParams) (string, []any) {
	var where postgres.WhereBuilder
	where.Add("u.deleted_at IS NULL")
	if params.GroupID > 0 {
		where.Addf("gm.group_id = %s", params.GroupID)
	}
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add(`(
			u.email ILIKE ` + search + `
			OR u.user_principal_name ILIKE ` + search + `
			OR u.mail_nickname ILIKE ` + search + `
			OR u.name ILIKE ` + search + `
			OR u.given_name ILIKE ` + search + `
			OR u.family_name ILIKE ` + search + `
			OR u.department ILIKE ` + search + `
		)`)
	}
	if len(params.Values) > 0 {
		where.Addf("u.id::text = ANY(%s::text[])", listing.NormalizeValues(params.Values))
	}
	if len(params.Roles) > 0 {
		roles := where.Arg(params.Roles)
		where.Add(`(
			` + effectiveRoleKeyExistsSQL("u.id", roles) + `
			OR ('none' = ANY(` + roles + `::text[]) AND NOT ` + effectiveRoleExistsSQL("u.id") + `)
		)`)
	}
	switch params.Source {
	case string(SourceLocal):
		where.Add("u.source = 'local'")
	case string(SourceEntra):
		where.Add("u.source = 'entra'")
	}
	return where.Build()
}

func departmentWhere(params UserListParams) (string, []any) {
	var where postgres.WhereBuilder
	where.Add("source <> 'local'")
	where.Add("deleted_at IS NULL")
	where.Add("NULLIF(btrim(department), '') IS NOT NULL")
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add("department ILIKE " + search)
	}
	if len(params.Values) > 0 {
		where.Addf("department = ANY(%s::text[])", listing.NormalizeValues(params.Values))
	}
	return where.Build()
}

func userListQuery(params UserListParams, where string, args []any) postgres.ListQuery {
	return postgres.ListQuery{
		SelectSQL: userListSelectSQL(params),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]postgres.OrderExpr{
			"name":       {SQL: "lower(u.name)"},
			"email":      {SQL: "lower(u.email)"},
			"role":       {SQL: directRoleKeySQL("u.id"), NullOrder: postgres.NullsLast},
			"department": {SQL: "lower(u.department)", NullOrder: postgres.NullsLast},
			"created_at": {SQL: "u.created_at"},
			"updated_at": {SQL: "u.updated_at"},
		},
		DefaultOrder: []postgres.OrderExpr{{SQL: "lower(u.name)"}, {SQL: "lower(u.email)"}, {SQL: "u.id"}},
		Params:       params.ListParams,
	}
}

func userListSelectSQL(params UserListParams) string {
	selectSQL := `SELECT ` + userColumnsSQL("u") + `
FROM users u`
	if params.GroupID <= 0 {
		return selectSQL
	}
	return selectSQL + `
JOIN directory_group_memberships gm ON gm.user_id = u.id`
}

func departmentListQuery(params UserListParams, where string, args []any) postgres.ListQuery {
	return postgres.ListQuery{
		SelectSQL: "SELECT DISTINCT department AS value FROM users",
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]postgres.OrderExpr{
			"value": {SQL: "department"},
		},
		DefaultOrder: []postgres.OrderExpr{{SQL: "department"}},
		Params:       params.ListParams,
	}
}
