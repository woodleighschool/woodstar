package labels

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists labels.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(ctx context.Context, params LabelListParams) ([]Label, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	where, args := labelListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL:    labelSelectSQL(),
		WhereSQL:     where,
		GroupBySQL:   "GROUP BY l.id",
		Args:         args,
		OrderKeys:    labelOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(l.name)"}, {SQL: "l.id"}},
		Params:       params.ListParams,
	}
	rows, count, err := dbutil.ListWithCount[labelRow](ctx, s.db.Pool(), listQuery)
	if err != nil {
		return nil, 0, err
	}
	labels := make([]Label, len(rows))
	for i, row := range rows {
		labels[i] = labelFromRow(row)
	}
	return labels, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Label, error) {
	return getLabelByID(ctx, s.db.Pool(), id)
}

func (s *Store) ListForHost(ctx context.Context, hostID int64) ([]Label, error) {
	rows, err := s.db.Pool().Query(ctx, labelSelectSQL()+`
JOIN label_membership lm_host ON lm_host.label_id = l.id AND lm_host.host_id = $1
GROUP BY l.id
ORDER BY lower(l.name), l.id`, hostID)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[labelRow])
	if err != nil {
		return nil, err
	}
	labels := make([]Label, len(records))
	for i, record := range records {
		labels[i] = labelFromRow(record)
	}
	return labels, nil
}

func (s *Store) Create(ctx context.Context, params LabelMutation) (*Label, error) {
	params.normalize()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	write := newLabelWrite(params)
	write.LabelType = string(LabelTypeRegular)

	var out *Label
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var id int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO labels (
				name,
				description,
				query,
				criteria,
				label_type,
				label_membership_type
			)
			VALUES (
				@name,
				@description,
				@query,
				@criteria::jsonb,
				@label_type,
				@label_membership_type
			)
			RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
			return dbutil.MutationError(err)
		}
		if err := replaceMembership(ctx, tx, id, params); err != nil {
			return err
		}
		var err error
		out, err = getLabelByID(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) Update(ctx context.Context, id int64, params LabelMutation) (*Label, error) {
	params.normalize()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	write := newLabelWrite(params)
	write.ID = id

	var out *Label
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var updatedID int64
		if err := tx.QueryRow(ctx, `
			UPDATE labels
			SET
				name = @name,
				description = @description,
				query = @query,
				criteria = @criteria::jsonb,
				label_membership_type = @label_membership_type,
				updated_at = now()
			WHERE id = @id AND label_type = 'regular'
			RETURNING id`, pgx.StructArgs(write)).Scan(&updatedID); err != nil {
			return dbutil.MutationError(err)
		}
		if err := replaceMembership(ctx, tx, updatedID, params); err != nil {
			return err
		}
		var err error
		out, err = getLabelByID(ctx, tx, updatedID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(
		ctx,
		`DELETE FROM labels WHERE id = $1 AND label_type = 'regular'`,
		id,
	)
	if err != nil {
		return dbutil.DeleteConflict(err, "Label is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

func (s *Store) ListApplicableDynamic(ctx context.Context) ([]Label, error) {
	rows, err := s.db.Pool().Query(ctx, labelSelectSQL()+`
WHERE l.label_membership_type = 'dynamic'
GROUP BY l.id
ORDER BY l.id`)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[labelRow])
	if err != nil {
		return nil, err
	}
	labels := make([]Label, len(records))
	for i, record := range records {
		labels[i] = labelFromRow(record)
	}
	return labels, nil
}

func (s *Store) ApplicableDynamicIDs(
	ctx context.Context,
	ids []int64,
) (map[int64]struct{}, error) {
	rows, err := s.db.Pool().Query(ctx, `
SELECT id
FROM labels
WHERE id = ANY($1::bigint[]) AND label_membership_type = 'dynamic'
ORDER BY id`, ids)
	if err != nil {
		return nil, err
	}
	matched, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, err
	}
	out := make(map[int64]struct{}, len(matched))
	for _, id := range matched {
		out[id] = struct{}{}
	}
	return out, nil
}

func (s *Store) SetMembership(ctx context.Context, labelID int64, hostID int64, matched bool) error {
	if matched {
		_, err := s.db.Pool().Exec(ctx, `
INSERT INTO label_membership (label_id, host_id)
VALUES ($1, $2)
ON CONFLICT (label_id, host_id) DO UPDATE SET updated_at = now()`, labelID, hostID)
		return err
	}
	_, err := s.db.Pool().Exec(
		ctx,
		`DELETE FROM label_membership WHERE label_id = $1 AND host_id = $2`,
		labelID,
		hostID,
	)
	return err
}

func (s *Store) RefreshDerived(ctx context.Context) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT id, criteria
FROM labels
WHERE label_membership_type = 'derived'
ORDER BY id`)
		if err != nil {
			return err
		}
		derived, err := pgx.CollectRows(rows, pgx.RowToStructByName[derivedLabelRow])
		if err != nil {
			return err
		}
		for _, label := range derived {
			if err := refreshDerivedMembership(ctx, tx, label.ID, label.Criteria.value); err != nil {
				return err
			}
		}
		return nil
	})
}

func labelListWhere(params LabelListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		where.Addf("(l.name ILIKE %s OR l.description ILIKE %s)", "%"+params.Q+"%", "%"+params.Q+"%")
	}
	if params.LabelType != "" {
		where.Addf("l.label_type = %s", string(params.LabelType))
	}
	if params.LabelMembershipType != "" {
		where.Addf("l.label_membership_type = %s", string(params.LabelMembershipType))
	}
	return where.Build()
}

func labelOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":                  {SQL: "lower(l.name)"},
		"label_type":            {SQL: "l.label_type"},
		"label_membership_type": {SQL: "l.label_membership_type"},
		"hosts_count":           {SQL: "hosts_count"},
		"updated_at":            {SQL: "l.updated_at"},
	}
}

type labelRow struct {
	ID                  int64         `db:"id"`
	Name                string        `db:"name"`
	BuiltinKey          *string       `db:"builtin_key"`
	Description         string        `db:"description"`
	Query               *string       `db:"query"`
	Criteria            labelCriteria `db:"criteria"`
	LabelType           string        `db:"label_type"`
	LabelMembershipType string        `db:"label_membership_type"`
	HostsCount          int32         `db:"hosts_count"`
	CreatedAt           time.Time     `db:"created_at"`
	UpdatedAt           time.Time     `db:"updated_at"`
}

func labelFromRow(row labelRow) Label {
	var builtinKey *BuiltinKey
	if row.BuiltinKey != nil {
		key := BuiltinKey(*row.BuiltinKey)
		builtinKey = &key
	}
	return Label{
		ID:                  row.ID,
		Name:                row.Name,
		BuiltinKey:          builtinKey,
		Description:         row.Description,
		Query:               row.Query,
		Criteria:            row.Criteria.value,
		LabelType:           LabelType(row.LabelType),
		LabelMembershipType: LabelMembershipType(row.LabelMembershipType),
		HostsCount:          row.HostsCount,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

type derivedLabelRow struct {
	ID       int64         `db:"id"`
	Criteria labelCriteria `db:"criteria"`
}

type labelWrite struct {
	ID                  int64         `db:"id"`
	Name                string        `db:"name"`
	Description         string        `db:"description"`
	Query               *string       `db:"query"`
	Criteria            labelCriteria `db:"criteria"`
	LabelType           string        `db:"label_type"`
	LabelMembershipType string        `db:"label_membership_type"`
}

func newLabelWrite(params LabelMutation) labelWrite {
	return labelWrite{
		Name:                params.Name,
		Description:         params.Description,
		Query:               params.Query,
		Criteria:            labelCriteria{value: params.Criteria},
		LabelMembershipType: string(params.LabelMembershipType),
	}
}

func getLabelByID(ctx context.Context, q dbutil.Queryer, id int64) (*Label, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := dbutil.GetOne[labelRow](ctx, q, labelSelectSQL()+"\nWHERE l.id = $1\nGROUP BY l.id", id)
	if err != nil {
		return nil, err
	}
	label := labelFromRow(row)
	if label.LabelMembershipType == LabelMembershipTypeManual {
		hostIDs, err := manualLabelHostIDs(ctx, q, id)
		if err != nil {
			return nil, err
		}
		label.HostIDs = hostIDs
	}
	return &label, nil
}

func manualLabelHostIDs(ctx context.Context, q dbutil.Queryer, labelID int64) ([]int64, error) {
	rows, err := q.Query(ctx, `
SELECT host_id
FROM label_membership lm
JOIN labels l ON l.id = lm.label_id
WHERE lm.label_id = $1 AND l.label_membership_type = 'manual'
ORDER BY lm.host_id`, labelID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

func replaceMembership(
	ctx context.Context,
	tx pgx.Tx,
	labelID int64,
	params LabelMutation,
) error {
	switch params.LabelMembershipType {
	case LabelMembershipTypeManual:
		return replaceManualMembership(ctx, tx, labelID, params.HostIDs)
	case LabelMembershipTypeDerived:
		return refreshDerivedMembership(ctx, tx, labelID, params.Criteria)
	case LabelMembershipTypeDynamic:
		return deleteMembership(ctx, tx, labelID)
	default:
		return nil
	}
}

func replaceManualMembership(ctx context.Context, tx pgx.Tx, labelID int64, hostIDs []int64) error {
	if err := deleteMembership(ctx, tx, labelID); err != nil {
		return err
	}
	if len(hostIDs) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO label_membership (label_id, host_id)
SELECT $1, unnest($2::bigint[])`, labelID, hostIDs); err != nil {
		return dbutil.MutationError(err)
	}
	return nil
}

func deleteMembership(ctx context.Context, tx pgx.Tx, labelID int64) error {
	_, err := tx.Exec(ctx, `DELETE FROM label_membership WHERE label_id = $1`, labelID)
	return err
}

func refreshDerivedMembership(ctx context.Context, tx pgx.Tx, labelID int64, criteria *Criteria) error {
	if err := validateCriteria(criteria); err != nil {
		return err
	}
	if err := deleteMembership(ctx, tx, labelID); err != nil {
		return err
	}

	values := normalizeCriteriaValues(criteria.Values)
	switch criteria.Attribute {
	case DerivedAttributeUserDepartment:
		_, err := tx.Exec(ctx, insertUserDepartmentMembershipSQL(), labelID, values)
		return err
	case DerivedAttributeDirectoryGroup:
		_, err := tx.Exec(ctx, insertDirectoryGroupMembershipSQL(), labelID, values)
		return err
	case DerivedAttributeUser:
		_, err := tx.Exec(ctx, insertUserMembershipSQL(), labelID, values)
		return err
	default:
		return fmt.Errorf("%w: unknown derived label attribute", dbutil.ErrInvalidInput)
	}
}

func labelSelectSQL() string {
	return `
SELECT
	l.id,
	l.name,
	l.builtin_key,
	l.description,
	l.query,
	l.criteria,
	l.label_type,
	l.label_membership_type,
	count(lm.host_id)::integer AS hosts_count,
	l.created_at,
	l.updated_at
FROM labels l
LEFT JOIN label_membership lm ON lm.label_id = l.id`
}

func primaryUserSourceOrderSQL() string {
	return `CASE source
		WHEN 'manual' THEN 0
		WHEN 'orbit_profile' THEN 1
		ELSE 10
	END`
}

func resolvedPrimaryUsersSQL() string {
	return `
WITH preferred AS (
	SELECT DISTINCT ON (host_id)
		host_id,
		email,
		source::text AS source
	FROM host_primary_user_sources
	ORDER BY host_id, ` + primaryUserSourceOrderSQL() + `, source
),
resolved AS (
	SELECT
		p.host_id,
		u.id AS user_id,
		COALESCE(u.department, '') AS department
	FROM preferred p
	JOIN LATERAL (
		SELECT u.id, u.department
		FROM users u
		WHERE u.deleted_at IS NULL
		  AND (
			lower(u.email) = lower(p.email)
			OR (
				u.user_principal_name IS NOT NULL
				AND lower(u.user_principal_name) = lower(p.email)
			)
		  )
		ORDER BY CASE WHEN lower(u.email) = lower(p.email) THEN 0 ELSE 1 END, u.id
		LIMIT 1
	) u ON true
)`
}

func insertUserDepartmentMembershipSQL() string {
	return resolvedPrimaryUsersSQL() + `
INSERT INTO label_membership (label_id, host_id)
SELECT DISTINCT $1::bigint, resolved.host_id
FROM resolved
WHERE resolved.department = ANY($2::text[])
ON CONFLICT (label_id, host_id) DO UPDATE SET updated_at = now()`
}

func insertDirectoryGroupMembershipSQL() string {
	return resolvedPrimaryUsersSQL() + `
INSERT INTO label_membership (label_id, host_id)
SELECT DISTINCT $1::bigint, resolved.host_id
FROM resolved
JOIN directory_group_memberships dgm ON dgm.user_id = resolved.user_id
JOIN directory_groups dg ON dg.id = dgm.group_id
WHERE dg.external_id = ANY($2::text[])
ON CONFLICT (label_id, host_id) DO UPDATE SET updated_at = now()`
}

func insertUserMembershipSQL() string {
	return resolvedPrimaryUsersSQL() + `
INSERT INTO label_membership (label_id, host_id)
SELECT DISTINCT $1::bigint, resolved.host_id
FROM resolved
WHERE resolved.user_id::text = ANY($2::text[])
ON CONFLICT (label_id, host_id) DO UPDATE SET updated_at = now()`
}
