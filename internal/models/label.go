package models

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// LabelKind separates system-seeded labels from admin-created labels.
type LabelKind string

const (
	LabelKindBuiltin LabelKind = "builtin"
	LabelKindCustom  LabelKind = "custom"
)

// LabelMembershipType controls how membership rows are produced.
type LabelMembershipType string

const (
	LabelMembershipTypeDynamic  LabelMembershipType = "dynamic"
	LabelMembershipTypeStatic   LabelMembershipType = "static"
	LabelMembershipTypeIdentity LabelMembershipType = "identity"
)

// Label is a host grouping and targeting primitive.
type Label struct {
	ID             int64
	Name           string
	Description    string
	Query          *string
	Kind           LabelKind
	MembershipType LabelMembershipType
	Platform       *string
	HostsCount     int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// LabelListParams filters the admin label list.
type LabelListParams struct {
	ListParams

	Kind           LabelKind
	MembershipType LabelMembershipType
	Platform       string
}

// LabelCreate contains fields for an admin-created label.
type LabelCreate struct {
	Name           string
	Description    string
	Query          *string
	Kind           LabelKind
	MembershipType LabelMembershipType
	Platform       *string
}

// LabelUpdate contains the full editable label state.
type LabelUpdate struct {
	Name           string
	Description    string
	Query          *string
	Kind           LabelKind
	MembershipType LabelMembershipType
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
		Kind:           string(params.Kind),
		MembershipType: string(params.MembershipType),
		Platform:       params.Platform,
		OrderKey:       params.OrderKey,
		OrderDirection: params.OrderDirection,
		LimitRows:      int32(params.PerPage),
		OffsetRows:     int32(params.Page * params.PerPage),
	})
	if err != nil {
		return nil, 0, err
	}
	count, err := s.q.CountLabels(ctx, sqlc.CountLabelsParams{
		Q:              params.Q,
		Kind:           string(params.Kind),
		MembershipType: string(params.MembershipType),
		Platform:       params.Platform,
	})
	if err != nil {
		return nil, 0, err
	}
	labels := make([]Label, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, labelFromListRow(row))
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
	label := labelFromGetRow(row)
	return &label, nil
}

// ListForHost returns labels currently matching a host.
func (s *LabelStore) ListForHost(ctx context.Context, hostID int64) ([]Label, error) {
	rows, err := s.q.ListLabelsForHost(ctx, sqlc.ListLabelsForHostParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	labels := make([]Label, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, labelFromHostRow(row))
	}
	return labels, nil
}

// Create inserts a custom label.
func (s *LabelStore) Create(ctx context.Context, params LabelCreate) (*Label, error) {
	params, err := cleanLabelCreate(params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.CreateLabel(ctx, sqlc.CreateLabelParams{
		Name:           params.Name,
		Description:    params.Description,
		Query:          params.Query,
		Kind:           string(params.Kind),
		MembershipType: string(params.MembershipType),
		Platform:       params.Platform,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	label := labelFromCreateRow(row)
	return &label, nil
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
		Kind:           string(params.Kind),
		MembershipType: string(params.MembershipType),
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
	label := labelFromUpdateRow(row)
	return &label, nil
}

// Delete removes a custom label.
func (s *LabelStore) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteCustomLabel(ctx, sqlc.DeleteCustomLabelParams{ID: id})
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
	labels := make([]Label, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, Label{
			ID:             row.ID,
			Name:           row.Name,
			Query:          row.Query,
			Kind:           LabelKind(row.Kind),
			MembershipType: LabelMembershipType(row.MembershipType),
			Platform:       row.Platform,
		})
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
	fields, err := cleanAdminLabelFields(labelFields(params), "created")
	if err != nil {
		return LabelCreate{}, err
	}
	return LabelCreate(fields), nil
}

func cleanLabelUpdate(params LabelUpdate) (LabelUpdate, error) {
	fields, err := cleanAdminLabelFields(labelFields(params), "updated")
	if err != nil {
		return LabelUpdate{}, err
	}
	return LabelUpdate(fields), nil
}

func cleanAdminLabelFields(params labelFields, action string) (labelFields, error) {
	fields, err := cleanLabelFields(params)
	if err != nil {
		return labelFields{}, err
	}
	if fields.Kind == LabelKindBuiltin {
		return labelFields{}, fmt.Errorf("%w: builtin labels cannot be %s", ErrInvalidInput, action)
	}
	return fields, nil
}

type labelFields struct {
	Name           string
	Description    string
	Query          *string
	Kind           LabelKind
	MembershipType LabelMembershipType
	Platform       *string
}

func cleanLabelFields(params labelFields) (labelFields, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = cleanStringPtr(params.Query)
	params.Platform = cleanStringPtr(params.Platform)
	if params.Kind == "" {
		params.Kind = LabelKindCustom
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
	params.Platform = strings.TrimSpace(params.Platform)
	return params
}

func validateLabelFields(name string, query *string, kind LabelKind, membershipType LabelMembershipType) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	switch kind {
	case LabelKindBuiltin, LabelKindCustom:
	default:
		return fmt.Errorf("%w: unknown label kind", ErrInvalidInput)
	}
	switch membershipType {
	case LabelMembershipTypeDynamic:
		if query == nil {
			return fmt.Errorf("%w: query is required for dynamic labels", ErrInvalidInput)
		}
	case LabelMembershipTypeStatic, LabelMembershipTypeIdentity:
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

type labelRecord struct {
	ID             int64
	Name           string
	Description    string
	Query          *string
	Kind           string
	MembershipType string
	Platform       *string
	HostsCount     int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func labelFromRecord(row labelRecord) Label {
	return Label{
		ID:             row.ID,
		Name:           row.Name,
		Description:    row.Description,
		Query:          row.Query,
		Kind:           LabelKind(row.Kind),
		MembershipType: LabelMembershipType(row.MembershipType),
		Platform:       row.Platform,
		HostsCount:     int(row.HostsCount),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func labelFromListRow(row sqlc.ListLabelsRow) Label {
	return labelFromRecord(labelRecord(row))
}

func labelFromGetRow(row sqlc.GetLabelByIDRow) Label {
	return labelFromRecord(labelRecord(row))
}

func labelFromCreateRow(row sqlc.CreateLabelRow) Label {
	return labelFromRecord(labelRecord(row))
}

func labelFromUpdateRow(row sqlc.UpdateLabelRow) Label {
	return labelFromRecord(labelRecord(row))
}

func labelFromHostRow(row sqlc.ListLabelsForHostRow) Label {
	return labelFromRecord(labelRecord(row))
}
