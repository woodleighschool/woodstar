package users

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists users and their Woodstar access fields.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

// UserCreate contains fields needed to create a user.
type UserCreate struct {
	Email    string `json:"email"          format:"email"`
	Name     string `json:"name,omitempty"`
	Role     Role   `json:"role"`
	Password string `json:"password"                      minLength:"12"`
}

// UserMutation replaces the writable fields of a user.
type UserMutation struct {
	Name     string  `json:"name"`
	Role     *Role   `json:"role,omitempty"`
	Password *string `json:"password,omitempty"`
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) Exists(ctx context.Context) (bool, error) {
	return s.q.UserExists(ctx)
}

func (s *Store) Create(ctx context.Context, params UserCreate) (*User, error) {
	hash, err := HashPassword(params.Password)
	if err != nil {
		return nil, err
	}

	row, err := s.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        strings.ToLower(params.Email),
		Name:         params.Name,
		PasswordHash: &hash,
		Role:         sqlc.UserRole(params.Role),
	})
	if err != nil {
		return nil, mapUserMutationError(err)
	}
	return new(userFromSQLC(row)), nil
}

func (s *Store) GetLoginByEmail(ctx context.Context, email string) (*User, error) {
	row, err := s.q.GetLoginUserByEmail(ctx, sqlc.GetLoginUserByEmailParams{Email: strings.ToLower(email)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

func (s *Store) GetSSOByEmail(ctx context.Context, email string) (*User, error) {
	row, err := s.q.GetSSOUserByEmail(ctx, sqlc.GetSSOUserByEmailParams{Email: strings.ToLower(email)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*User, error) {
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

func (s *Store) List(ctx context.Context, params ListParams) ([]User, int, error) {
	where, args := userWhere(params)
	listQuery := userListQuery(params, where, args)
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
	list, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.User])
	if err != nil {
		return nil, 0, err
	}
	out := make([]User, len(list))
	for i, row := range list {
		out[i] = userFromSQLC(row)
	}
	return out, count, nil
}

func (s *Store) ListDepartments(ctx context.Context, params ListParams) ([]Department, int, error) {
	where, args := departmentWhere(params)
	listQuery := departmentListQuery(params, where, args)
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
	departments, err := pgx.CollectRows(rows, pgx.RowToStructByName[Department])
	return departments, count, err
}

func (s *Store) Update(ctx context.Context, id int64, params UserMutation) (*User, error) {
	var role *sqlc.UserRole
	if params.Role != nil {
		value := sqlc.UserRole(*params.Role)
		role = &value
	}
	var passwordHash *string
	if params.Password != nil {
		hash, err := HashPassword(*params.Password)
		if err != nil {
			return nil, err
		}
		passwordHash = &hash
	}

	row, err := s.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		Name:         params.Name,
		Role:         role,
		PasswordHash: passwordHash,
		ID:           id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, mapUserMutationError(err)
	}
	return new(userFromSQLC(row)), nil
}

// UpdateAccount writes the signed-in user's editable account fields and
// returns the fresh account view in a single round trip.
func (s *Store) UpdateAccount(ctx context.Context, id int64, params AccountMutation) (*Account, error) {
	var passwordHash *string
	if params.Password != nil {
		hash, err := HashPassword(*params.Password)
		if err != nil {
			return nil, err
		}
		passwordHash = &hash
	}
	row, err := s.q.UpdateAccountByID(ctx, sqlc.UpdateAccountByIDParams{
		Name:         params.Name,
		PasswordHash: passwordHash,
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

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteUser(ctx, sqlc.DeleteUserParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func (s *Store) DeactivateSynced(ctx context.Context, id int64) error {
	_, err := s.q.DeactivateSyncedUser(ctx, sqlc.DeactivateSyncedUserParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func (s *Store) GetByAPIKey(ctx context.Context, key string) (*User, error) {
	row, err := s.q.GetUserByAPIKey(ctx, sqlc.GetUserByAPIKeyParams{APIKey: &key})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return new(userFromSQLC(row)), nil
}

// SetAPIKey writes a freshly generated API key for id and resets the
// created_at and last_used_at timestamps.
func (s *Store) SetAPIKey(ctx context.Context, id int64, key string) (*Account, error) {
	row, err := s.q.SetUserAPIKey(ctx, sqlc.SetUserAPIKeyParams{ID: id, APIKey: &key})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, mapUserMutationError(err)
	}
	return new(accountFromSQLC(row)), nil
}

func (s *Store) ClearAPIKey(ctx context.Context, id int64) (*Account, error) {
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
		EntraID:           stringValue(s.EntraID),
		UserPrincipalName: stringValue(s.UserPrincipalName),
		MailNickname:      stringValue(s.MailNickname),
		GivenName:         stringValue(s.GivenName),
		FamilyName:        stringValue(s.FamilyName),
		Department:        stringValue(s.Department),
		Active:            s.Active,
		Synced:            s.EntraID != nil,
		CanLogin:          s.Active && role != nil,
		LastSyncedAt:      s.LastSyncedAt,
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

func mapUserMutationError(err error) error {
	if database.SQLState(err) == pgerrcode.UniqueViolation {
		return dbutil.ErrAlreadyExists
	}
	return err
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

func userWhere(params ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.GroupID > 0 {
		groupID := where.Arg(params.GroupID)
		where.Add("gm.group_id = " + groupID)
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
		values := where.Arg(dbutil.SplitListValues(params.Values))
		where.Add("u.id::text = ANY(" + values + "::text[])")
	}
	switch params.Role {
	case "admin", "viewer":
		role := where.Arg(params.Role)
		where.Add("u.role = " + role + "::user_role")
	case "none":
		where.Add("u.role IS NULL")
	}
	switch params.Source {
	case "local":
		where.Add("u.entra_id IS NULL")
	case "synced":
		where.Add("u.entra_id IS NOT NULL")
	}
	switch params.Status {
	case "active":
		where.Add("u.active")
	case "inactive":
		where.Add("NOT u.active")
	}
	return where.Build()
}

func departmentWhere(params ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	where.Add("entra_id IS NOT NULL")
	where.Add("NULLIF(btrim(department), '') IS NOT NULL")
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add("department ILIKE " + search)
	}
	if len(params.Values) > 0 {
		values := where.Arg(dbutil.SplitListValues(params.Values))
		where.Add("department = ANY(" + values + "::text[])")
	}
	return where.Build()
}

func userListQuery(params ListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: userListSelectSQL(params),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":           {SQL: "lower(u.name)"},
			"email":          {SQL: "lower(u.email)"},
			"role":           {SQL: "u.role", NullOrder: dbutil.NullsLast},
			"department":     {SQL: "lower(u.department)", NullOrder: dbutil.NullsLast},
			"created_at":     {SQL: "u.created_at"},
			"updated_at":     {SQL: "u.updated_at"},
			"last_synced_at": {SQL: "u.last_synced_at", NullOrder: dbutil.NullsLast},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(u.name)"}, {SQL: "lower(u.email)"}, {SQL: "u.id"}},
		Params:       params.ListParams,
	}
}

func userListSelectSQL(params ListParams) string {
	if params.GroupID <= 0 {
		return "SELECT u.* FROM users u"
	}
	return `
SELECT u.*
FROM users u
JOIN entra_group_memberships gm ON gm.user_id = u.id`
}

func departmentListQuery(params ListParams, where string, args []any) dbutil.ListQuery {
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
