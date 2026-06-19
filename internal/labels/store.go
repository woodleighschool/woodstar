package labels

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists labels.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) List(ctx context.Context, params LabelListParams) ([]Label, int, error) {
	where, args := labelListWhere(params)
	records, count, err := dbutil.ListWithCount[labelListRecord](
		ctx,
		s.db.Pool(),
		labelListQuery(params, where, args),
	)
	if err != nil {
		return nil, 0, err
	}
	labels := make([]Label, len(records))
	for i, record := range records {
		label, err := labelFromListRecord(record)
		if err != nil {
			return nil, 0, err
		}
		labels[i] = label
	}
	return labels, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Label, error) {
	return getLabelByID(ctx, s.q, id)
}

func (s *Store) ListForHost(ctx context.Context, hostID int64) ([]Label, error) {
	rows, err := s.q.ListLabelsForHost(ctx, sqlc.ListLabelsForHostParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	labels := make([]Label, len(rows))
	for i, row := range rows {
		l, err := labelFromSQLC(row.Label)
		if err != nil {
			return nil, err
		}
		l.HostsCount = row.HostsCount
		labels[i] = *l
	}
	return labels, nil
}

func (s *Store) Create(ctx context.Context, params LabelMutation) (*Label, error) {
	params = params.withDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	var out *Label
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var criteria []byte
		if params.Criteria != nil {
			var err error
			criteria, err = criteriaJSON(*params.Criteria)
			if err != nil {
				return err
			}
		}
		row, err := q.CreateLabel(ctx, sqlc.CreateLabelParams{
			Name:                params.Name,
			Description:         params.Description,
			Query:               params.Query,
			Criteria:            criteria,
			LabelType:           string(LabelTypeRegular),
			LabelMembershipType: string(params.LabelMembershipType),
		})
		if err != nil {
			return err
		}
		if err := s.replaceMembership(ctx, q, row.ID, params); err != nil {
			return err
		}
		out, err = getLabelByID(ctx, q, row.ID)
		return err
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return out, nil
}

func (s *Store) Update(ctx context.Context, id int64, params LabelMutation) (*Label, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	var out *Label
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var criteria []byte
		if params.Criteria != nil {
			var err error
			criteria, err = criteriaJSON(*params.Criteria)
			if err != nil {
				return err
			}
		}
		row, err := q.UpdateLabel(ctx, sqlc.UpdateLabelParams{
			Name:                params.Name,
			Description:         params.Description,
			Query:               params.Query,
			Criteria:            criteria,
			LabelMembershipType: string(params.LabelMembershipType),
			ID:                  id,
		})
		if err != nil {
			return err
		}
		if err := s.replaceMembership(ctx, q, row.ID, LabelMutation{
			Query:               params.Query,
			Criteria:            params.Criteria,
			HostIDs:             params.HostIDs,
			LabelMembershipType: params.LabelMembershipType,
		}); err != nil {
			return err
		}
		out, err = getLabelByID(ctx, q, row.ID)
		return err
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return out, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteRegularLabel(ctx, sqlc.DeleteRegularLabelParams{ID: id})
	return dbutil.GetError(err)
}

func (s *Store) ListApplicableDynamic(ctx context.Context) ([]Label, error) {
	rows, err := s.q.ListApplicableDynamicLabels(ctx)
	if err != nil {
		return nil, err
	}
	labels := make([]Label, len(rows))
	for i, row := range rows {
		label, err := labelFromSQLC(row)
		if err != nil {
			return nil, err
		}
		labels[i] = *label
	}
	return labels, nil
}

func (s *Store) ApplicableDynamicIDs(
	ctx context.Context,
	ids []int64,
) (map[int64]struct{}, error) {
	rows, err := s.q.ListApplicableDynamicLabelIDs(ctx, sqlc.ListApplicableDynamicLabelIDsParams{
		Ids: ids,
	})
	if err != nil {
		return nil, err
	}
	out := make(map[int64]struct{}, len(rows))
	for _, id := range rows {
		out[id] = struct{}{}
	}
	return out, nil
}

func (s *Store) SetMembership(ctx context.Context, labelID int64, hostID int64, matched bool) error {
	if matched {
		return s.q.UpsertLabelMembership(ctx, sqlc.UpsertLabelMembershipParams{LabelID: labelID, HostID: hostID})
	}
	return s.q.DeleteLabelMembership(ctx, sqlc.DeleteLabelMembershipParams{LabelID: labelID, HostID: hostID})
}

func (s *Store) RefreshDerived(ctx context.Context) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		derived, err := q.ListDerivedLabels(ctx)
		if err != nil {
			return err
		}
		for _, label := range derived {
			criteria, err := decodeCriteria(label.Criteria)
			if err != nil {
				return err
			}
			if err := refreshDerivedMembership(ctx, q, label.ID, &criteria); err != nil {
				return err
			}
		}
		return nil
	})
}

// Validate checks the label shape before the DB sees it.
func (p LabelMutation) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	return validateMembershipPairing(p.LabelMembershipType, p.Query, p.Criteria, p.HostIDs)
}

func (p LabelMutation) withDefaults() LabelMutation {
	if p.LabelMembershipType == "" {
		p.LabelMembershipType = LabelMembershipTypeDynamic
	}
	return p
}

func validateMembershipPairing(
	membershipType LabelMembershipType,
	query *string,
	criteria *Criteria,
	hostIDs []int64,
) error {
	switch membershipType {
	case LabelMembershipTypeDynamic:
		return validateDynamicMembership(query, criteria, hostIDs)
	case LabelMembershipTypeManual:
		return validateManualMembership(query, criteria)
	case LabelMembershipTypeDerived:
		return validateDerivedMembership(query, criteria, hostIDs)
	default:
		return fmt.Errorf("%w: membership type must be dynamic, manual, or derived", dbutil.ErrInvalidInput)
	}
}

func validateDynamicMembership(query *string, criteria *Criteria, hostIDs []int64) error {
	if query == nil || strings.TrimSpace(*query) == "" {
		return fmt.Errorf("%w: query is required for dynamic labels", dbutil.ErrInvalidInput)
	}
	if criteria != nil {
		return fmt.Errorf("%w: criteria is only allowed for derived labels", dbutil.ErrInvalidInput)
	}
	if len(hostIDs) > 0 {
		return fmt.Errorf("%w: hosts are only allowed for manual labels", dbutil.ErrInvalidInput)
	}
	return nil
}

func validateManualMembership(query *string, criteria *Criteria) error {
	if query != nil {
		return fmt.Errorf("%w: query is only allowed for dynamic labels", dbutil.ErrInvalidInput)
	}
	if criteria != nil {
		return fmt.Errorf("%w: criteria is only allowed for derived labels", dbutil.ErrInvalidInput)
	}
	return nil
}

func validateDerivedMembership(query *string, criteria *Criteria, hostIDs []int64) error {
	if query != nil {
		return fmt.Errorf("%w: query is only allowed for dynamic labels", dbutil.ErrInvalidInput)
	}
	if len(hostIDs) > 0 {
		return fmt.Errorf("%w: hosts are only allowed for manual labels", dbutil.ErrInvalidInput)
	}
	return validateCriteria(criteria)
}

func validateCriteria(criteria *Criteria) error {
	if criteria == nil {
		return fmt.Errorf("%w: criteria is required for derived labels", dbutil.ErrInvalidInput)
	}
	switch criteria.Attribute {
	case DerivedAttributeUserDepartment, DerivedAttributeDirectoryGroup, DerivedAttributeUser:
	default:
		return fmt.Errorf("%w: unknown derived label attribute", dbutil.ErrInvalidInput)
	}
	if len(cleanCriteriaValues(criteria.Values)) == 0 {
		return fmt.Errorf("%w: derived label values are required", dbutil.ErrInvalidInput)
	}
	return nil
}

func labelListQuery(params LabelListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: `SELECT
	l.id,
	l.name,
	l.builtin_key,
	l.description,
	l.query,
	l.criteria,
	l.label_type,
	l.label_membership_type,
	l.created_at,
	l.updated_at,
	count(lm.host_id)::integer AS hosts_count
FROM labels l
LEFT JOIN label_membership lm ON lm.label_id = l.id`,
		WhereSQL:   where,
		GroupBySQL: "GROUP BY l.id",
		Args:       args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":                  {SQL: "lower(l.name)"},
			"label_type":            {SQL: "l.label_type"},
			"label_membership_type": {SQL: "l.label_membership_type"},
			"hosts_count":           {SQL: "hosts_count"},
			"updated_at":            {SQL: "l.updated_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(l.name)"}, {SQL: "l.id"}},
		Params:       params.ListParams,
	}
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

type labelListRecord struct {
	ID                  int64
	Name                string
	BuiltinKey          *string
	Description         string
	Query               *string
	Criteria            []byte
	LabelType           string
	LabelMembershipType string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	HostsCount          int32
}

func labelFromSQLC(s sqlc.Label) (*Label, error) {
	var criteria *Criteria
	if len(s.Criteria) > 0 {
		decoded, err := decodeCriteria(s.Criteria)
		if err != nil {
			return nil, err
		}
		criteria = &decoded
	}
	var builtinKey *BuiltinKey
	if s.BuiltinKey != nil {
		key := BuiltinKey(*s.BuiltinKey)
		builtinKey = &key
	}
	return &Label{
		ID:                  s.ID,
		Name:                s.Name,
		BuiltinKey:          builtinKey,
		Description:         s.Description,
		Query:               s.Query,
		Criteria:            criteria,
		LabelType:           LabelType(s.LabelType),
		LabelMembershipType: LabelMembershipType(s.LabelMembershipType),
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}, nil
}

func labelFromListRecord(s labelListRecord) (Label, error) {
	label, err := labelFromSQLC(sqlc.Label{
		ID:                  s.ID,
		Name:                s.Name,
		BuiltinKey:          s.BuiltinKey,
		Description:         s.Description,
		Query:               s.Query,
		Criteria:            s.Criteria,
		LabelType:           s.LabelType,
		LabelMembershipType: s.LabelMembershipType,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	})
	if err != nil {
		return Label{}, err
	}
	label.HostsCount = s.HostsCount
	return *label, nil
}

func criteriaJSON(criteria Criteria) ([]byte, error) {
	normalized := Criteria{Attribute: criteria.Attribute, Values: cleanCriteriaValues(criteria.Values)}
	return normalized.json()
}

func decodeCriteria(raw []byte) (Criteria, error) {
	var criteria Criteria
	if err := json.Unmarshal(raw, &criteria); err != nil {
		return Criteria{}, fmt.Errorf("decode label criteria: %w", err)
	}
	return criteria, nil
}

func getLabelByID(ctx context.Context, q *sqlc.Queries, id int64) (*Label, error) {
	row, err := q.GetLabelByID(ctx, sqlc.GetLabelByIDParams{ID: id})
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	label, err := labelFromSQLC(row.Label)
	if err != nil {
		return nil, err
	}
	label.HostsCount = row.HostsCount
	if label.LabelMembershipType == LabelMembershipTypeManual {
		hostIDs, err := q.ListManualLabelHostIDs(ctx, sqlc.ListManualLabelHostIDsParams{LabelID: id})
		if err != nil {
			return nil, err
		}
		label.HostIDs = hostIDs
	}
	return label, nil
}

func (s *Store) replaceMembership(
	ctx context.Context,
	q *sqlc.Queries,
	labelID int64,
	params LabelMutation,
) error {
	switch params.LabelMembershipType {
	case LabelMembershipTypeManual:
		hostIDs := compactHostIDs(params.HostIDs)
		if err := q.DeleteLabelMembershipsForLabel(
			ctx,
			sqlc.DeleteLabelMembershipsForLabelParams{LabelID: labelID},
		); err != nil {
			return err
		}
		if len(hostIDs) == 0 {
			return nil
		}
		return q.InsertLabelMemberships(ctx, sqlc.InsertLabelMembershipsParams{LabelID: labelID, HostIds: hostIDs})
	case LabelMembershipTypeDerived:
		return refreshDerivedMembership(ctx, q, labelID, params.Criteria)
	case LabelMembershipTypeDynamic:
		return q.DeleteLabelMembershipsForLabel(ctx, sqlc.DeleteLabelMembershipsForLabelParams{LabelID: labelID})
	default:
		return nil
	}
}

func compactHostIDs(hostIDs []int64) []int64 {
	if len(hostIDs) == 0 {
		return nil
	}
	return dbutil.Dedup(hostIDs)
}

func refreshDerivedMembership(ctx context.Context, q *sqlc.Queries, labelID int64, criteria *Criteria) error {
	if err := validateCriteria(criteria); err != nil {
		return err
	}
	if err := q.DeleteLabelMembershipsForLabel(
		ctx,
		sqlc.DeleteLabelMembershipsForLabelParams{LabelID: labelID},
	); err != nil {
		return err
	}

	values := cleanCriteriaValues(criteria.Values)
	switch criteria.Attribute {
	case DerivedAttributeUserDepartment:
		return q.InsertUserDepartmentLabelMemberships(
			ctx,
			sqlc.InsertUserDepartmentLabelMembershipsParams{LabelID: labelID, Values: values},
		)
	case DerivedAttributeDirectoryGroup:
		return q.InsertDirectoryGroupLabelMemberships(
			ctx,
			sqlc.InsertDirectoryGroupLabelMembershipsParams{LabelID: labelID, Values: values},
		)
	case DerivedAttributeUser:
		return q.InsertUserLabelMemberships(
			ctx,
			sqlc.InsertUserLabelMembershipsParams{LabelID: labelID, Values: values},
		)
	default:
		return fmt.Errorf("%w: unknown derived label attribute", dbutil.ErrInvalidInput)
	}
}

func cleanCriteriaValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return dbutil.Dedup(out)
}
