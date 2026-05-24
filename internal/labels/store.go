package labels

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/platforms"
)

// Store persists labels and host memberships.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) List(ctx context.Context, params ListParams) ([]Label, int, error) {
	params = cleanListParams(params)
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
		labels[i] = labelFromListRecord(record)
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
	label := new(labelFromSQLC(row.Label))
	label.HostsCount = int(row.HostsCount)
	return label, nil
}

func (s *Store) ListForHost(ctx context.Context, hostID int64) ([]Label, error) {
	rows, err := s.q.ListLabelsForHost(ctx, sqlc.ListLabelsForHostParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	labels := make([]Label, len(rows))
	for i, row := range rows {
		l := labelFromSQLC(row.Label)
		l.HostsCount = int(row.HostsCount)
		labels[i] = l
	}
	return labels, nil
}

func (s *Store) Create(ctx context.Context, params LabelCreate) (*Label, error) {
	params, err := cleanLabelCreate(params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.CreateLabel(ctx, sqlc.CreateLabelParams{
		Name:                params.Name,
		Description:         params.Description,
		Query:               params.Query,
		LabelType:           string(params.LabelType),
		LabelMembershipType: params.LabelMembershipType,
		Platforms:           params.Platforms,
	})
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return nil, dbutil.ErrAlreadyExists
		}
		return nil, err
	}
	return new(labelFromSQLC(row)), nil
}

func (s *Store) Update(ctx context.Context, id int64, params LabelUpdate) (*Label, error) {
	params, err := cleanLabelUpdate(params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpdateLabel(ctx, sqlc.UpdateLabelParams{
		Name:                params.Name,
		Description:         params.Description,
		Query:               params.Query,
		LabelMembershipType: params.LabelMembershipType,
		Platforms:           params.Platforms,
		ID:                  id,
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
	return new(labelFromSQLC(row)), nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteRegularLabel(ctx, sqlc.DeleteRegularLabelParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func (s *Store) ListApplicableDynamic(ctx context.Context, platform string) ([]Label, error) {
	rows, err := s.q.ListApplicableDynamicLabels(ctx, sqlc.ListApplicableDynamicLabelsParams{
		Platform: strings.TrimSpace(platform),
	})
	if err != nil {
		return nil, err
	}
	labels := make([]Label, len(rows))
	for i, row := range rows {
		labels[i] = labelFromSQLC(row)
	}
	return labels, nil
}

func (s *Store) ApplicableDynamicIDs(
	ctx context.Context,
	ids []int64,
	platform string,
) (map[int64]struct{}, error) {
	rows, err := s.q.ListApplicableDynamicLabelIDs(ctx, sqlc.ListApplicableDynamicLabelIDsParams{
		Ids:      ids,
		Platform: strings.TrimSpace(platform),
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

func (s *Store) MarkHostLabelsFresh(ctx context.Context, hostID int64) error {
	return s.q.MarkHostLabelsFresh(ctx, sqlc.MarkHostLabelsFreshParams{HostID: hostID})
}

func cleanLabelCreate(params LabelCreate) (LabelCreate, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = dbutil.CleanStringPtr(params.Query)
	params.LabelType = cleanLabelType(params.LabelType)
	params.LabelMembershipType = cleanMembershipType(params.LabelMembershipType)
	targets, err := platforms.CleanTargets(params.Platforms)
	if err != nil {
		return LabelCreate{}, fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	params.Platforms = targets
	if params.LabelType == LabelTypeBuiltin {
		return LabelCreate{}, fmt.Errorf("%w: builtin labels cannot be created", dbutil.ErrInvalidInput)
	}
	if err := validateLabelFields(params.Name, params.Query, params.LabelType, params.LabelMembershipType); err != nil {
		return LabelCreate{}, err
	}
	return params, nil
}

func cleanLabelUpdate(params LabelUpdate) (LabelUpdate, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = dbutil.CleanStringPtr(params.Query)
	params.LabelMembershipType = cleanMembershipType(params.LabelMembershipType)
	targets, err := platforms.CleanTargets(params.Platforms)
	if err != nil {
		return LabelUpdate{}, fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	params.Platforms = targets
	if err := validateLabelFields(params.Name, params.Query, LabelTypeRegular, params.LabelMembershipType); err != nil {
		return LabelUpdate{}, err
	}
	return params, nil
}

func cleanLabelType(labelType LabelType) LabelType {
	labelType = LabelType(strings.TrimSpace(string(labelType)))
	if labelType == "" {
		return LabelTypeRegular
	}
	return labelType
}

func cleanMembershipType(membershipType string) string {
	membershipType = strings.TrimSpace(membershipType)
	if membershipType == "" {
		return LabelMembershipTypeDynamic
	}
	return membershipType
}

func cleanListParams(params ListParams) ListParams {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	params.LabelType = LabelType(strings.TrimSpace(string(params.LabelType)))
	params.LabelMembershipType = strings.TrimSpace(params.LabelMembershipType)
	params.Platform = platforms.CleanPlatform(params.Platform)
	return params
}

func labelListSQLWithWhere(params ListParams, where string, args []any) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL: `SELECT
	l.id,
	l.name,
	l.description,
	l.query,
	l.label_type,
	l.label_membership_type,
	l.platforms,
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
			"platform":              {SQL: "l.platforms::text"},
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
		where.Add("l.label_membership_type = " + where.Arg(params.LabelMembershipType))
	}
	if params.Platform != "" {
		where.Add(where.Arg(params.Platform) + " = ANY(l.platforms::text[])")
	}
	return where.Build()
}

type labelListRecord struct {
	ID                  int64
	Name                string
	Description         string
	Query               *string
	LabelType           string
	LabelMembershipType string
	Platforms           []platforms.Platform
	CreatedAt           time.Time
	UpdatedAt           time.Time
	HostsCount          int32
}

func validateLabelFields(name string, query *string, labelType LabelType, labelMembershipType string) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	switch labelType {
	case LabelTypeBuiltin, LabelTypeRegular:
	default:
		return fmt.Errorf("%w: unknown label type", dbutil.ErrInvalidInput)
	}
	switch labelMembershipType {
	case LabelMembershipTypeDynamic:
		if query == nil {
			return fmt.Errorf("%w: query is required for dynamic labels", dbutil.ErrInvalidInput)
		}
	case LabelMembershipTypeManual, LabelMembershipTypeDerived:
		if query != nil {
			return fmt.Errorf("%w: query is only allowed for dynamic labels", dbutil.ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unknown label membership type", dbutil.ErrInvalidInput)
	}
	return nil
}

func labelFromSQLC(s sqlc.Label) Label {
	return Label{
		ID:                  s.ID,
		Name:                s.Name,
		Description:         s.Description,
		Query:               s.Query,
		LabelType:           LabelType(s.LabelType),
		LabelMembershipType: s.LabelMembershipType,
		Platforms:           s.Platforms,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
}

func labelFromListRecord(s labelListRecord) Label {
	label := labelFromSQLC(sqlc.Label{
		ID:                  s.ID,
		Name:                s.Name,
		Description:         s.Description,
		Query:               s.Query,
		LabelType:           s.LabelType,
		LabelMembershipType: s.LabelMembershipType,
		Platforms:           s.Platforms,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	})
	label.HostsCount = int(s.HostsCount)
	return label
}
