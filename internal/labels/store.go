package labels

import (
	"context"
	"encoding/json"
	"errors"
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

func (s *Store) List(ctx context.Context, params ListParams) ([]Label, int, error) {
	where, args := labelListWhere(params)
	var count int
	if err := s.db.Pool().
		QueryRow(ctx, "SELECT count(*)::integer FROM labels l "+where, args...).
		Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := labelListSQLWithWhere(params, where, args)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[labelListRecord])
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
	row, err := s.q.GetLabelByID(ctx, sqlc.GetLabelByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	label, err := labelFromSQLC(row.Label)
	if err != nil {
		return nil, err
	}
	label.HostsCount = row.HostsCount
	if label.LabelMembershipType == LabelMembershipTypeManual {
		hostIDs, err := s.q.ListManualLabelHostIDs(ctx, sqlc.ListManualLabelHostIDsParams{LabelID: id})
		if err != nil {
			return nil, err
		}
		label.HostIDs = hostIDs
	}
	return label, nil
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

func (s *Store) Create(ctx context.Context, params LabelCreate) (*Label, error) {
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
			LabelType:           string(params.LabelType),
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
		if dbutil.IsUniqueViolation(err) {
			return nil, dbutil.ErrAlreadyExists
		}
		return nil, err
	}
	return out, nil
}

func (s *Store) Update(ctx context.Context, id int64, params LabelUpdate) (*Label, error) {
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
		if err := s.replaceMembership(ctx, q, row.ID, LabelCreate{
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
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return nil, dbutil.ErrAlreadyExists
		}
		return nil, err
	}
	return out, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteRegularLabel(ctx, sqlc.DeleteRegularLabelParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
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
func (p LabelCreate) Validate() error {
	if p.LabelType == LabelTypeBuiltin {
		return fmt.Errorf("%w: builtin labels cannot be created", dbutil.ErrInvalidInput)
	}
	return validateMembershipPairing(p.LabelMembershipType, p.Query, p.Criteria, p.HostIDs)
}

// Validate checks the update shape.
func (p LabelUpdate) Validate() error {
	return validateMembershipPairing(p.LabelMembershipType, p.Query, p.Criteria, p.HostIDs)
}

func (p LabelCreate) withDefaults() LabelCreate {
	if p.LabelType == "" {
		p.LabelType = LabelTypeRegular
	}
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
		if query == nil || strings.TrimSpace(*query) == "" {
			return fmt.Errorf("%w: query is required for dynamic labels", dbutil.ErrInvalidInput)
		}
		if criteria != nil {
			return fmt.Errorf("%w: criteria is only allowed for derived labels", dbutil.ErrInvalidInput)
		}
		if len(hostIDs) > 0 {
			return fmt.Errorf("%w: hosts are only allowed for manual labels", dbutil.ErrInvalidInput)
		}
	case LabelMembershipTypeManual:
		if query != nil {
			return fmt.Errorf("%w: query is only allowed for dynamic labels", dbutil.ErrInvalidInput)
		}
		if criteria != nil {
			return fmt.Errorf("%w: criteria is only allowed for derived labels", dbutil.ErrInvalidInput)
		}
	case LabelMembershipTypeDerived:
		if query != nil {
			return fmt.Errorf("%w: query is only allowed for dynamic labels", dbutil.ErrInvalidInput)
		}
		if len(hostIDs) > 0 {
			return fmt.Errorf("%w: hosts are only allowed for manual labels", dbutil.ErrInvalidInput)
		}
		if err := validateCriteria(criteria); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: membership type must be dynamic, manual, or derived", dbutil.ErrInvalidInput)
	}
	for _, hostID := range hostIDs {
		if hostID <= 0 {
			return fmt.Errorf("%w: host IDs must be positive", dbutil.ErrInvalidInput)
		}
	}
	return nil
}

func validateCriteria(criteria *Criteria) error {
	if criteria == nil {
		return fmt.Errorf("%w: criteria is required for derived labels", dbutil.ErrInvalidInput)
	}
	switch criteria.Attribute {
	case DerivedAttributeDirectoryDepartment, DerivedAttributeDirectoryGroup, DerivedAttributeDirectoryUser:
	default:
		return fmt.Errorf("%w: unknown derived label attribute", dbutil.ErrInvalidInput)
	}
	if len(cleanCriteriaValues(criteria.Values)) == 0 {
		return fmt.Errorf("%w: derived label values are required", dbutil.ErrInvalidInput)
	}
	return nil
}

func labelListSQLWithWhere(params ListParams, where string, args []any) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL: `SELECT
	l.id,
	l.name,
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
	}.Build()
}

func labelListWhere(params ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add("(l.name ILIKE " + search + " OR l.description ILIKE " + search + ")")
	}
	if params.LabelType != "" {
		where.Add("l.label_type = " + where.Arg(string(params.LabelType)))
	}
	if params.LabelMembershipType != "" {
		where.Add("l.label_membership_type = " + where.Arg(string(params.LabelMembershipType)))
	}
	return where.Build()
}

type labelListRecord struct {
	ID                  int64
	Name                string
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
	return &Label{
		ID:                  s.ID,
		Name:                s.Name,
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
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
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
	params LabelCreate,
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
	seen := make(map[int64]struct{}, len(hostIDs))
	out := make([]int64, 0, len(hostIDs))
	for _, hostID := range hostIDs {
		if _, ok := seen[hostID]; ok {
			continue
		}
		seen[hostID] = struct{}{}
		out = append(out, hostID)
	}
	return out
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
	case DerivedAttributeDirectoryDepartment:
		return q.InsertDirectoryDepartmentLabelMemberships(
			ctx,
			sqlc.InsertDirectoryDepartmentLabelMembershipsParams{LabelID: labelID, Values: values},
		)
	case DerivedAttributeDirectoryGroup:
		return q.InsertDirectoryGroupLabelMemberships(
			ctx,
			sqlc.InsertDirectoryGroupLabelMembershipsParams{LabelID: labelID, Values: values},
		)
	case DerivedAttributeDirectoryUser:
		return q.InsertDirectoryUserLabelMemberships(
			ctx,
			sqlc.InsertDirectoryUserLabelMembershipsParams{LabelID: labelID, Values: values},
		)
	default:
		return fmt.Errorf("%w: unknown derived label attribute", dbutil.ErrInvalidInput)
	}
}

func cleanCriteriaValues(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
