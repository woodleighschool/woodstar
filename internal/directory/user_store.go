package directory

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists directory users, groups, memberships, and source snapshots.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) UserExists(ctx context.Context) (bool, error) {
	return s.q.UserExists(ctx)
}

type userCreateRecord struct {
	Email        string
	Name         string
	PasswordHash string
	Role         Role
}

func (s *Store) createUser(ctx context.Context, params userCreateRecord) (*User, error) {
	row, err := s.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        strings.ToLower(params.Email),
		Name:         params.Name,
		PasswordHash: &params.PasswordHash,
		Role:         sqlc.UserRole(params.Role),
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return new(userFromSQLC(row)), nil
}

func (s *Store) GetLoginUserByEmail(ctx context.Context, email string) (*User, error) {
	row, err := s.q.GetLoginUserByEmail(ctx, sqlc.GetLoginUserByEmailParams{Email: strings.ToLower(email)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

func (s *Store) GetSSOUserByEmail(ctx context.Context, email string) (*User, error) {
	row, err := s.q.GetSSOUserByEmail(ctx, sqlc.GetSSOUserByEmailParams{Email: strings.ToLower(email)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row, err := s.q.GetUserByID(ctx, sqlc.GetUserByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

// GetAccountByID returns the signed-in user's self-view, including API key fields.
func (s *Store) GetAccountByID(ctx context.Context, id int64) (*Account, error) {
	row, err := s.q.GetUserByID(ctx, sqlc.GetUserByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(accountFromSQLC(row)), nil
}

func (s *Store) ListUsers(ctx context.Context, params UserListParams) ([]User, int, error) {
	where, args := userWhere(params)
	list, count, err := dbutil.ListWithCount[sqlc.User](ctx, s.db.Pool(), userListQuery(params, where, args))
	if err != nil {
		return nil, 0, err
	}
	out := make([]User, len(list))
	for i, row := range list {
		out[i] = userFromSQLC(row)
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
	var role *sqlc.UserRole
	if params.Role != nil {
		value := sqlc.UserRole(*params.Role)
		role = &value
	}

	row, err := s.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		Name:         params.Name,
		Role:         role,
		PasswordHash: params.PasswordHash,
		ID:           id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return new(userFromSQLC(row)), nil
}

func (s *Store) updateAccount(ctx context.Context, id int64, params accountUpdateRecord) (*Account, error) {
	row, err := s.q.UpdateAccountByID(ctx, sqlc.UpdateAccountByIDParams{
		Name:         params.Name,
		PasswordHash: params.PasswordHash,
		ID:           id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(accountFromSQLC(row)), nil
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.q.DeleteUser(ctx, sqlc.DeleteUserParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func (s *Store) SoftDeleteUser(ctx context.Context, id int64) error {
	_, err := s.q.SoftDeleteDirectoryUser(ctx, sqlc.SoftDeleteDirectoryUserParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func (s *Store) GetUserByAPIKey(ctx context.Context, key string) (*User, error) {
	row, err := s.q.GetUserByAPIKey(ctx, sqlc.GetUserByAPIKeyParams{APIKey: &key})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

func (s *Store) setAccountAPIKey(ctx context.Context, id int64, key string) (*Account, error) {
	row, err := s.q.SetUserAPIKey(ctx, sqlc.SetUserAPIKeyParams{ID: id, APIKey: &key})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return new(accountFromSQLC(row)), nil
}

func (s *Store) clearAccountAPIKey(ctx context.Context, id int64) (*Account, error) {
	row, err := s.q.ClearUserAPIKey(ctx, sqlc.ClearUserAPIKeyParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(accountFromSQLC(row)), nil
}

func userFromSQLC(s sqlc.User) User {
	role := roleFromSQLC(s.Role)
	return User{
		ID:                s.ID,
		Email:             s.Email,
		Name:              s.Name,
		PasswordHash:      stringValue(s.PasswordHash),
		Role:              role,
		Source:            Source(s.Source),
		ExternalID:        stringValue(s.ExternalID),
		UserPrincipalName: stringValue(s.UserPrincipalName),
		MailNickname:      stringValue(s.MailNickname),
		GivenName:         stringValue(s.GivenName),
		FamilyName:        stringValue(s.FamilyName),
		Department:        stringValue(s.Department),
		CanLogin:          s.DeletedAt == nil && role != nil,
		DeletedAt:         s.DeletedAt,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}

func accountFromSQLC(s sqlc.User) Account {
	account := Account{
		User:            userFromSQLC(s),
		APIKeyCreatedAt: s.APIKeyCreatedAt,
	}
	if s.APIKey != nil {
		account.APIKey = *s.APIKey
	}
	return account
}

func roleFromSQLC(role *sqlc.UserRole) *Role {
	if role == nil {
		return nil
	}
	value := Role(*role)
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
	if params.GroupID <= 0 {
		return "SELECT u.* FROM users u"
	}
	return `
SELECT u.*
FROM users u
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
