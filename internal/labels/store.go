package labels

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// NewStore returns a label store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

// List returns labels and the total count matching params.
func (s *Store) List(ctx context.Context, params LabelListParams) ([]Label, int, error) {
	params = cleanLabelListParams(params)
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
	labels := make([]Label, 0)
	for rows.Next() {
		var label Label
		if err := rows.Scan(
			&label.ID,
			&label.Name,
			&label.Description,
			&label.Query,
			&label.LabelType,
			&label.LabelMembershipType,
			&label.Platforms,
			&label.CreatedAt,
			&label.UpdatedAt,
			&label.HostsCount,
		); err != nil {
			return nil, 0, err
		}
		labels = append(labels, label)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return labels, count, nil
}

// GetByID returns one label by database ID.
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

// ListForHost returns labels currently matching a host.
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

// Create inserts a regular label.
func (s *Store) Create(ctx context.Context, params LabelCreate) (*Label, error) {
	params, err := cleanLabelCreate(params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.CreateLabel(ctx, sqlc.CreateLabelParams{
		Name:                params.Name,
		Description:         params.Description,
		Query:               params.Query,
		LabelType:           params.LabelType,
		LabelMembershipType: params.LabelMembershipType,
		Platforms:           toSQLCPlatforms(params.Platforms),
	})
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return nil, dbutil.ErrAlreadyExists
		}
		return nil, err
	}
	return new(labelFromSQLC(row)), nil
}

// Update replaces editable label fields.
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
		Platforms:           toSQLCPlatforms(params.Platforms),
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

// Delete removes a regular label.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteRegularLabel(ctx, sqlc.DeleteRegularLabelParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

// ListApplicableDynamic returns dynamic labels that should run for a host platform.
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

// ApplicableDynamicIDs returns the subset of ids that are current dynamic labels for a host platform.
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

// SetMembership records whether hostID currently matches labelID.
func (s *Store) SetMembership(ctx context.Context, labelID int64, hostID int64, matched bool) error {
	if matched {
		return s.q.UpsertLabelMembership(ctx, sqlc.UpsertLabelMembershipParams{LabelID: labelID, HostID: hostID})
	}
	return s.q.DeleteLabelMembership(ctx, sqlc.DeleteLabelMembershipParams{LabelID: labelID, HostID: hostID})
}

// MarkHostLabelsFresh records a successful label evaluation pass.
func (s *Store) MarkHostLabelsFresh(ctx context.Context, hostID int64) error {
	return s.q.MarkHostLabelsFresh(ctx, sqlc.MarkHostLabelsFreshParams{HostID: hostID})
}

func cleanLabelCreate(params LabelCreate) (LabelCreate, error) {
	fields, err := cleanLabelFields(labelFields(params))
	if err != nil {
		return LabelCreate{}, err
	}
	if fields.LabelType == LabelTypeBuiltin {
		return LabelCreate{}, fmt.Errorf("%w: builtin labels cannot be created", dbutil.ErrInvalidInput)
	}
	return LabelCreate(fields), nil
}

func cleanLabelUpdate(params LabelUpdate) (LabelUpdate, error) {
	fields, err := cleanLabelFields(labelFields{
		Name:                params.Name,
		Description:         params.Description,
		Query:               params.Query,
		LabelType:           LabelTypeRegular,
		LabelMembershipType: params.LabelMembershipType,
		Platforms:           params.Platforms,
	})
	if err != nil {
		return LabelUpdate{}, err
	}
	return LabelUpdate{
		Name:                fields.Name,
		Description:         fields.Description,
		Query:               fields.Query,
		LabelMembershipType: fields.LabelMembershipType,
		Platforms:           fields.Platforms,
	}, nil
}

type labelFields struct {
	Name                string
	Description         string
	Query               *string
	LabelType           string
	LabelMembershipType string
	Platforms           []platforms.Platform
}

func cleanLabelFields(params labelFields) (labelFields, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = dbutil.CleanStringPtr(params.Query)
	targets, err := platforms.CleanTargets(params.Platforms)
	if err != nil {
		return labelFields{}, fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	params.Platforms = targets
	if params.LabelType == "" {
		params.LabelType = LabelTypeRegular
	}
	if params.LabelMembershipType == "" {
		params.LabelMembershipType = LabelMembershipTypeDynamic
	}
	if err := validateLabelFields(params.Name, params.Query, params.LabelType, params.LabelMembershipType); err != nil {
		return labelFields{}, err
	}
	return params, nil
}

func cleanLabelListParams(params LabelListParams) LabelListParams {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	params.LabelType = strings.TrimSpace(params.LabelType)
	params.LabelMembershipType = strings.TrimSpace(params.LabelMembershipType)
	params.Platform = platforms.CleanPlatform(params.Platform)
	return params
}

func labelListSQLWithWhere(params LabelListParams, where string, args []any) (string, []any, error) {
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

func labelListWhere(params LabelListParams) (string, []any) {
	clauses := make([]string, 0, 4)
	args := make([]any, 0)
	if params.Q != "" {
		args = append(args, "%"+params.Q+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, "(l.name ILIKE "+placeholder+" OR l.description ILIKE "+placeholder+")")
	}
	if params.LabelType != "" {
		args = append(args, params.LabelType)
		clauses = append(clauses, fmt.Sprintf("l.label_type = $%d", len(args)))
	}
	if params.LabelMembershipType != "" {
		args = append(args, params.LabelMembershipType)
		clauses = append(clauses, fmt.Sprintf("l.label_membership_type = $%d", len(args)))
	}
	if params.Platform != "" {
		args = append(args, params.Platform)
		clauses = append(clauses, fmt.Sprintf("$%d = ANY(l.platforms::text[])", len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func validateLabelFields(name string, query *string, labelType, labelMembershipType string) error {
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
		LabelType:           s.LabelType,
		LabelMembershipType: s.LabelMembershipType,
		Platforms:           platformsFromSQLC(s.Platforms),
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
}

func toSQLCPlatforms(values []platforms.Platform) []sqlc.Platform {
	out := make([]sqlc.Platform, len(values))
	for i, value := range values {
		out[i] = sqlc.Platform(value)
	}
	return out
}

func platformsFromSQLC(values []sqlc.Platform) []platforms.Platform {
	out := make([]platforms.Platform, len(values))
	for i, value := range values {
		out[i] = platforms.Platform(value)
	}
	return out
}
