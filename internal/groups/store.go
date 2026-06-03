package groups

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store reads synced directory groups and memberships.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(ctx context.Context, params ListParams) ([]Group, int, error) {
	where, args := groupWhere(params)
	listQuery := groupListQuery(params, where, args)
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
	groups, err := pgx.CollectRows(rows, pgx.RowToStructByName[Group])
	return groups, count, err
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Group, error) {
	var group Group
	err := s.db.Pool().QueryRow(ctx, groupSelectSQL+`
WHERE g.id = $1
GROUP BY g.id`, id).Scan(
		&group.ID,
		&group.ExternalID,
		&group.DisplayName,
		&group.MailNickname,
		&group.MemberCount,
		&group.LastSyncedAt,
		&group.CreatedAt,
		&group.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &group, nil
}

const groupSelectSQL = `SELECT
	g.id,
	g.external_id,
	g.display_name,
	COALESCE(g.mail_nickname, '') AS mail_nickname,
	count(gm.user_id)::integer AS member_count,
	g.last_synced_at,
	g.created_at,
	g.updated_at
FROM entra_groups g
LEFT JOIN entra_group_memberships gm ON gm.group_id = g.id
`

func groupWhere(params ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			g.display_name ILIKE ` + search + `
			OR g.mail_nickname ILIKE ` + search + `
			OR g.external_id ILIKE ` + search + `
		)`)
	}
	if len(params.Values) > 0 {
		values := where.Arg(dbutil.SplitListValues(params.Values))
		where.Add("g.external_id = ANY(" + values + "::text[])")
	}
	return where.Build()
}

func groupListQuery(params ListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL:  groupSelectSQL,
		WhereSQL:   where,
		GroupBySQL: "GROUP BY g.id",
		Args:       args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"display_name":   {SQL: "lower(g.display_name)"},
			"mail_nickname":  {SQL: "lower(g.mail_nickname)", NullOrder: dbutil.NullsLast},
			"member_count":   {SQL: "member_count"},
			"last_synced_at": {SQL: "g.last_synced_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(g.display_name)"}, {SQL: "g.id"}},
		Params:       params.ListParams,
	}
}
