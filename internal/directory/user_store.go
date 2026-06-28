package directory

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists directory users, groups, memberships, and source snapshots.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
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

var userColumnExprs = []string{
	"id",
	"email",
	"name",
	"password_hash",
	"role::text AS role",
	"api_key",
	"api_key_created_at",
	"source::text AS source",
	"external_id",
	"user_principal_name",
	"mail_nickname",
	"given_name",
	"family_name",
	"department",
	"deleted_at",
	"created_at",
	"updated_at",
}

var userSelectSQL = `
SELECT
    ` + userColumnsSQL("") + `
FROM users`

func userColumnsSQL(alias string) string {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	columns := make([]string, len(userColumnExprs))
	for i, column := range userColumnExprs {
		columns[i] = prefix + column
	}
	return strings.Join(columns, ", ")
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
		CanLogin:          r.DeletedAt == nil && role != nil,
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

func (s *Store) UserExists(ctx context.Context) (bool, error) {
	var exists bool
	err := s.db.Pool().QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users)`).Scan(&exists)
	return exists, err
}

type userCreateRecord struct {
	Email        string
	Name         string
	PasswordHash string
	Role         Role
}

func (s *Store) createUser(ctx context.Context, params userCreateRecord) (*User, error) {
	var id int64
	err := s.db.Pool().QueryRow(ctx, `
INSERT INTO users (email, name, password_hash, role, source)
VALUES ($1, $2, $3, $4::user_role, 'local')
RETURNING id`,
		strings.ToLower(params.Email), params.Name, params.PasswordHash, string(params.Role),
	).Scan(&id)
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return s.GetUserByID(ctx, id)
}

func (s *Store) GetLoginUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.getUserByEmail(ctx, email, `
WHERE deleted_at IS NULL
  AND source = 'local'
  AND role IS NOT NULL
  AND password_hash IS NOT NULL
  AND (
      lower(email) = lower($1)
      OR lower(COALESCE(user_principal_name, '')) = lower($1)
  )
ORDER BY CASE WHEN lower(email) = lower($1) THEN 0 ELSE 1 END, id
LIMIT 1`)
}

func (s *Store) GetSSOUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.getUserByEmail(ctx, email, `
WHERE deleted_at IS NULL
  AND role IS NOT NULL
  AND (
      lower(email) = lower($1)
      OR lower(COALESCE(user_principal_name, '')) = lower($1)
)
ORDER BY CASE WHEN lower(email) = lower($1) THEN 0 ELSE 1 END, id
LIMIT 1`)
}

func (s *Store) getUserByEmail(ctx context.Context, email string, whereSQL string) (*User, error) {
	row, err := dbutil.GetOne[userRow](ctx, s.db.Pool(), userSelectSQL+whereSQL, strings.ToLower(email))
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	out := userFromRow(row)
	return &out, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row, err := dbutil.GetOne[userRow](ctx, s.db.Pool(), userSelectSQL+`
WHERE id = $1
  AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	out := userFromRow(row)
	return &out, nil
}

// GetAccountByID returns the signed-in user's self-view, including API key fields.
func (s *Store) GetAccountByID(ctx context.Context, id int64) (*Account, error) {
	row, err := dbutil.GetOne[userRow](ctx, s.db.Pool(), userSelectSQL+`
WHERE id = $1
  AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	out := accountFromRow(row)
	return &out, nil
}

func (s *Store) ListUsers(ctx context.Context, params UserListParams) ([]User, int, error) {
	where, args := userWhere(params)
	rows, count, err := dbutil.ListWithCount[userRow](ctx, s.db.Pool(), userListQuery(params, where, args))
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
	return dbutil.ListWithCount[Department](ctx, s.db.Pool(), departmentListQuery(params, where, args))
}

type userUpdateRecord struct {
	Name         string
	Role         *Role
	PasswordHash *string
}

func (s *Store) updateUser(ctx context.Context, id int64, params userUpdateRecord) (*User, error) {
	var roleStr *string
	if params.Role != nil {
		v := string(*params.Role)
		roleStr = &v
	}
	qrows, err := s.db.Pool().Query(ctx, `
UPDATE users
SET
    name = CASE WHEN source = 'local' THEN $1 ELSE name END,
    role = $2::user_role,
    password_hash = CASE
        WHEN source = 'local' THEN COALESCE($3, password_hash)
        ELSE password_hash
    END,
    updated_at = now()
WHERE id = $4
RETURNING `+userColumnsSQL(""),
		params.Name, roleStr, params.PasswordHash, id)
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	row, err := pgx.CollectExactlyOneRow(qrows, pgx.RowToStructByName[userRow])
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	out := userFromRow(row)
	return &out, nil
}

func (s *Store) updateAccount(ctx context.Context, id int64, params accountUpdateRecord) (*Account, error) {
	qrows, err := s.db.Pool().Query(ctx, `
UPDATE users
SET
    name = CASE WHEN source = 'local' THEN $1 ELSE name END,
    password_hash = CASE
        WHEN source = 'local' THEN COALESCE($2, password_hash)
        ELSE password_hash
    END,
    updated_at = now()
WHERE id = $3
RETURNING `+userColumnsSQL(""),
		params.Name, params.PasswordHash, id)
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	row, err := pgx.CollectExactlyOneRow(qrows, pgx.RowToStructByName[userRow])
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	out := accountFromRow(row)
	return &out, nil
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	var deletedID int64
	err := s.db.Pool().QueryRow(ctx, `
DELETE FROM users
WHERE id = $1
  AND source = 'local'
  AND deleted_at IS NULL
RETURNING id`, id).Scan(&deletedID)
	return dbutil.GetError(err)
}

func (s *Store) SoftDeleteUser(ctx context.Context, id int64) error {
	var updatedID int64
	err := s.db.Pool().QueryRow(ctx, `
UPDATE users
SET
    deleted_at = now(),
    updated_at = now()
WHERE id = $1
  AND source <> 'local'
  AND deleted_at IS NULL
RETURNING id`, id).Scan(&updatedID)
	return dbutil.GetError(err)
}

func (s *Store) GetUserByAPIKey(ctx context.Context, key string) (*User, error) {
	row, err := dbutil.GetOne[userRow](ctx, s.db.Pool(), userSelectSQL+`
WHERE api_key = $1
  AND deleted_at IS NULL
  AND role IS NOT NULL`, key)
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	out := userFromRow(row)
	return &out, nil
}

func (s *Store) setAccountAPIKey(ctx context.Context, id int64, key string) (*Account, error) {
	qrows, err := s.db.Pool().Query(ctx, `
UPDATE users
SET
    api_key = $1,
    api_key_created_at = now(),
    updated_at = now()
WHERE id = $2
  AND deleted_at IS NULL
  AND role IS NOT NULL
RETURNING `+userColumnsSQL(""),
		key, id)
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	row, err := pgx.CollectExactlyOneRow(qrows, pgx.RowToStructByName[userRow])
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	out := accountFromRow(row)
	return &out, nil
}

func (s *Store) clearAccountAPIKey(ctx context.Context, id int64) (*Account, error) {
	qrows, err := s.db.Pool().Query(ctx, `
UPDATE users
SET
    api_key = NULL,
    api_key_created_at = NULL,
    updated_at = now()
WHERE id = $1
RETURNING `+userColumnsSQL(""),
		id)
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	row, err := pgx.CollectExactlyOneRow(qrows, pgx.RowToStructByName[userRow])
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	out := accountFromRow(row)
	return &out, nil
}

func userWhere(params UserListParams) (string, []any) {
	var where dbutil.WhereBuilder
	where.Add("u.deleted_at IS NULL")
	if params.GroupID > 0 {
		where.Addf("gm.group_id = %s", params.GroupID)
	}
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
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
		where.Addf("u.id::text = ANY(%s::text[])", dbutil.SplitListValues(params.Values))
	}
	switch params.Role {
	case string(RoleAdmin), string(RoleViewer):
		where.Addf("u.role = %s::user_role", params.Role)
	case "none":
		where.Add("u.role IS NULL")
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
	var where dbutil.WhereBuilder
	where.Add("source <> 'local'")
	where.Add("deleted_at IS NULL")
	where.Add("NULLIF(btrim(department), '') IS NOT NULL")
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add("department ILIKE " + search)
	}
	if len(params.Values) > 0 {
		where.Addf("department = ANY(%s::text[])", dbutil.SplitListValues(params.Values))
	}
	return where.Build()
}

func userListQuery(params UserListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: userListSelectSQL(params),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":       {SQL: "lower(u.name)"},
			"email":      {SQL: "lower(u.email)"},
			"role":       {SQL: "u.role", NullOrder: dbutil.NullsLast},
			"department": {SQL: "lower(u.department)", NullOrder: dbutil.NullsLast},
			"created_at": {SQL: "u.created_at"},
			"updated_at": {SQL: "u.updated_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(u.name)"}, {SQL: "lower(u.email)"}, {SQL: "u.id"}},
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

func departmentListQuery(params UserListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: "SELECT DISTINCT department AS value FROM users",
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"value": {SQL: "department"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "department"}},
		Params:       params.ListParams,
	}
}
