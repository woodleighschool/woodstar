package rbac

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/goodies/auth/authz"
)

// Store reads the application's persisted role assignments.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates an authorization store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Grants returns direct and group-inherited role facts without evaluating them.
func (store *Store) Grants(ctx context.Context, userID int64) ([]authz.Grant, error) {
	rows, err := store.pool.Query(ctx, `
WITH assigned_roles AS (
    SELECT role_id FROM authz_user_roles WHERE user_id = $1
    UNION ALL
    SELECT group_role.role_id
    FROM authz_group_roles AS group_role
    JOIN directory_group_memberships AS membership ON membership.group_id = group_role.group_id
    WHERE membership.user_id = $1
)
SELECT permissions.resource, permissions.access
FROM assigned_roles AS roles
JOIN authz_role_permissions AS permissions ON permissions.role_id = roles.role_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	grants := []authz.Grant{}
	for rows.Next() {
		var grant authz.Grant
		var level int16
		if err := rows.Scan(&grant.Resource, &level); err != nil {
			return nil, err
		}
		grant.Access, err = accessFromLevel(level)
		if err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return grants, nil
}

func accessFromLevel(level int16) (authz.Access, error) {
	switch level {
	case 1:
		return authz.View, nil
	case 2:
		return authz.Edit, nil
	default:
		return "", fmt.Errorf("%w: database level %d", authz.ErrInvalidAccess, level)
	}
}
