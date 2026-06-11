package directory

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) ListGroups(ctx context.Context, params GroupListParams) ([]Group, int, error) {
	where, args := groupWhere(params)
	return dbutil.ListWithCount[Group](ctx, s.db.Pool(), groupListQuery(params, where, args))
}

func (s *Store) GetGroupByID(ctx context.Context, id int64) (*Group, error) {
	var group Group
	err := s.db.Pool().QueryRow(ctx, groupSelectSQL+`
WHERE g.id = $1
GROUP BY g.id`, id).Scan(
		&group.ID,
		&group.Source,
		&group.ExternalID,
		&group.DisplayName,
		&group.MailNickname,
		&group.MemberCount,
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
	g.source,
	g.external_id,
	g.display_name,
	COALESCE(g.mail_nickname, '') AS mail_nickname,
	count(u.id)::integer AS member_count,
	g.created_at,
	g.updated_at
FROM directory_groups g
LEFT JOIN directory_group_memberships gm ON gm.group_id = g.id
LEFT JOIN users u ON u.id = gm.user_id AND u.deleted_at IS NULL
`

func groupWhere(params GroupListParams) (string, []any) {
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

func groupListQuery(params GroupListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL:  groupSelectSQL,
		WhereSQL:   where,
		GroupBySQL: "GROUP BY g.id",
		Args:       args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"display_name":  {SQL: "lower(g.display_name)"},
			"mail_nickname": {SQL: "lower(g.mail_nickname)", NullOrder: dbutil.NullsLast},
			"member_count":  {SQL: "member_count"},
			"source":        {SQL: "g.source"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(g.display_name)"}, {SQL: "g.id"}},
		Params:       params.ListParams,
	}
}
