package models

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// Label kinds. Kind separates system-seeded labels from admin-created ones.
const (
	LabelKindBuiltin = "builtin"
	LabelKindRegular = "regular"
)

// Label membership types. MembershipType controls how membership rows are produced:
//   - dynamic: an osquery query result drives membership
//   - manual: the server writes membership rows (e.g. All Hosts on enroll)
//   - host_vitals: membership is derived from host fields, not osquery
const (
	LabelMembershipTypeDynamic    = "dynamic"
	LabelMembershipTypeManual     = "manual"
	LabelMembershipTypeHostVitals = "host_vitals"
)

// Label is a host grouping and targeting primitive.
type Label struct {
	sqlc.Label
	HostsCount int
}

// LabelListParams filters the admin label list.
type LabelListParams struct {
	ListParams

	Kind           string
	MembershipType string
	Platform       string
}

// LabelCreate contains fields for an admin-created label.
type LabelCreate struct {
	Name           string
	Description    string
	Query          *string
	Kind           string
	MembershipType string
	Platform       *string
}

// LabelUpdate contains the full editable label state.
type LabelUpdate struct {
	Name           string
	Description    string
	Query          *string
	MembershipType string
	Platform       *string
}

// LabelStore persists labels and host memberships.
type LabelStore struct {
	q *sqlc.Queries
}

// NewLabelStore returns a label store backed by db.
func NewLabelStore(db *database.DB) *LabelStore {
	return &LabelStore{q: db.Queries()}
}

// List returns labels and the total count matching params.
func (s *LabelStore) List(ctx context.Context, params LabelListParams) ([]Label, int, error) {
	params = cleanLabelListParams(params)
	rows, err := s.q.ListLabels(ctx, sqlc.ListLabelsParams{
		Q:              params.Q,
		Kind:           params.Kind,
		MembershipType: params.MembershipType,
		Platform:       params.Platform,
		OrderKey:       params.OrderKey,
		OrderDirection: params.OrderDirection,
		LimitRows:      int32(params.PerPage),
		OffsetRows:     int32((params.Page - 1) * params.PerPage),
	})
	if err != nil {
		return nil, 0, err
	}
	count, err := s.q.CountLabels(ctx, sqlc.CountLabelsParams{
		Q:              params.Q,
		Kind:           params.Kind,
		MembershipType: params.MembershipType,
		Platform:       params.Platform,
	})
	if err != nil {
		return nil, 0, err
	}
	labels := make([]Label, len(rows))
	for i, row := range rows {
		labels[i] = Label{Label: row.Label, HostsCount: int(row.HostsCount)}
	}
	return labels, int(count), nil
}

// GetByID returns one label by database ID.
func (s *LabelStore) GetByID(ctx context.Context, id int64) (*Label, error) {
	row, err := s.q.GetLabelByID(ctx, sqlc.GetLabelByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &Label{Label: row.Label, HostsCount: int(row.HostsCount)}, nil
}

// ListForHost returns labels currently matching a host.
func (s *LabelStore) ListForHost(ctx context.Context, hostID int64) ([]Label, error) {
	rows, err := s.q.ListLabelsForHost(ctx, sqlc.ListLabelsForHostParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	labels := make([]Label, len(rows))
	for i, row := range rows {
		labels[i] = Label{Label: row.Label, HostsCount: int(row.HostsCount)}
	}
	return labels, nil
}

// Create inserts a regular label.
func (s *LabelStore) Create(ctx context.Context, params LabelCreate) (*Label, error) {
	params, err := cleanLabelCreate(params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.CreateLabel(ctx, sqlc.CreateLabelParams{
		Name:           params.Name,
		Description:    params.Description,
		Query:          params.Query,
		Kind:           params.Kind,
		MembershipType: params.MembershipType,
		Platform:       params.Platform,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	return &Label{Label: row}, nil
}

// Update replaces editable label fields.
func (s *LabelStore) Update(ctx context.Context, id int64, params LabelUpdate) (*Label, error) {
	params, err := cleanLabelUpdate(params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpdateLabel(ctx, sqlc.UpdateLabelParams{
		Name:           params.Name,
		Description:    params.Description,
		Query:          params.Query,
		MembershipType: params.MembershipType,
		Platform:       params.Platform,
		ID:             id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	return &Label{Label: row}, nil
}

// Delete removes a regular label.
func (s *LabelStore) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteRegularLabel(ctx, sqlc.DeleteRegularLabelParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// ListApplicableDynamic returns dynamic labels that should run for a host platform.
func (s *LabelStore) ListApplicableDynamic(ctx context.Context, platform string) ([]Label, error) {
	rows, err := s.q.ListApplicableDynamicLabels(ctx, sqlc.ListApplicableDynamicLabelsParams{
		Platform: strings.TrimSpace(platform),
	})
	if err != nil {
		return nil, err
	}
	labels := make([]Label, len(rows))
	for i, row := range rows {
		labels[i] = Label{Label: row}
	}
	return labels, nil
}

// ApplicableDynamicIDs returns the subset of ids that are current dynamic labels for platform.
func (s *LabelStore) ApplicableDynamicIDs(
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
func (s *LabelStore) SetMembership(ctx context.Context, labelID int64, hostID int64, matched bool) error {
	if matched {
		return s.q.UpsertLabelMembership(ctx, sqlc.UpsertLabelMembershipParams{LabelID: labelID, HostID: hostID})
	}
	return s.q.DeleteLabelMembership(ctx, sqlc.DeleteLabelMembershipParams{LabelID: labelID, HostID: hostID})
}

// MarkHostLabelsFresh records a successful label evaluation pass.
func (s *LabelStore) MarkHostLabelsFresh(ctx context.Context, hostID int64) error {
	return s.q.MarkHostLabelsFresh(ctx, sqlc.MarkHostLabelsFreshParams{HostID: hostID})
}

func cleanLabelCreate(params LabelCreate) (LabelCreate, error) {
	fields, err := cleanLabelFields(labelFields(params))
	if err != nil {
		return LabelCreate{}, err
	}
	if fields.Kind == LabelKindBuiltin {
		return LabelCreate{}, fmt.Errorf("%w: builtin labels cannot be created", ErrInvalidInput)
	}
	return LabelCreate(fields), nil
}

func cleanLabelUpdate(params LabelUpdate) (LabelUpdate, error) {
	fields, err := cleanLabelFields(labelFields{
		Name:           params.Name,
		Description:    params.Description,
		Query:          params.Query,
		Kind:           LabelKindRegular,
		MembershipType: params.MembershipType,
		Platform:       params.Platform,
	})
	if err != nil {
		return LabelUpdate{}, err
	}
	return LabelUpdate{
		Name:           fields.Name,
		Description:    fields.Description,
		Query:          fields.Query,
		MembershipType: fields.MembershipType,
		Platform:       fields.Platform,
	}, nil
}

type labelFields struct {
	Name           string
	Description    string
	Query          *string
	Kind           string
	MembershipType string
	Platform       *string
}

func cleanLabelFields(params labelFields) (labelFields, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = cleanStringPtr(params.Query)
	params.Platform = cleanStringPtr(params.Platform)
	if params.Kind == "" {
		params.Kind = LabelKindRegular
	}
	if params.MembershipType == "" {
		params.MembershipType = LabelMembershipTypeDynamic
	}
	if err := validateLabelFields(params.Name, params.Query, params.Kind, params.MembershipType); err != nil {
		return labelFields{}, err
	}
	return params, nil
}

func cleanLabelListParams(params LabelListParams) LabelListParams {
	params.ListParams = CleanListParams(params.ListParams)
	params.Kind = strings.TrimSpace(params.Kind)
	params.MembershipType = strings.TrimSpace(params.MembershipType)
	params.Platform = strings.TrimSpace(params.Platform)
	return params
}

func validateLabelFields(name string, query *string, kind, membershipType string) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	switch kind {
	case LabelKindBuiltin, LabelKindRegular:
	default:
		return fmt.Errorf("%w: unknown label kind", ErrInvalidInput)
	}
	switch membershipType {
	case LabelMembershipTypeDynamic:
		if query == nil {
			return fmt.Errorf("%w: query is required for dynamic labels", ErrInvalidInput)
		}
	case LabelMembershipTypeManual, LabelMembershipTypeHostVitals:
		if query != nil {
			return fmt.Errorf("%w: query is only allowed for dynamic labels", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unknown label membership type", ErrInvalidInput)
	}
	return nil
}

func cleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := strings.TrimSpace(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}
