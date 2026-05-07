package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/models"
)

const (
	labelsTag   = "Labels"
	labelIDPath = "/api/labels/{id}"
)

type labelBody struct {
	ID             string                     `json:"id"`
	Name           string                     `json:"name"`
	Description    string                     `json:"description"`
	Query          *string                    `json:"query,omitempty"`
	Kind           models.LabelKind           `json:"kind"`
	MembershipType models.LabelMembershipType `json:"membership_type"`
	Platform       *string                    `json:"platform,omitempty"`
	HostsCount     int                        `json:"hosts_count"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

type labelListOutput struct {
	Body labelListBody
}

type labelOutput struct {
	Body labelBody
}

type labelListBody struct {
	Items []labelBody `json:"items"`
	Count int         `json:"count"`
}

type labelListInput struct {
	Q              string `query:"q,omitempty"`
	Page           int    `query:"page,omitempty"`
	PerPage        int    `query:"per_page,omitempty"`
	OrderKey       string `query:"order_key,omitempty"`
	OrderDirection string `query:"order_direction,omitempty"`
	Kind           string `query:"kind,omitempty"`
	MembershipType string `query:"membership_type,omitempty"`
	Platform       string `query:"platform,omitempty"`
}

type labelGetInput struct {
	ID string `path:"id"`
}

type labelCreateInput struct {
	Body labelMutationBody
}

// labelPutBody mirrors labelBody so the autopatch round-trip (GET → merge →
// PUT) sends a body the handler accepts. Read-only fields are accepted but
// ignored.
type labelPutBody struct {
	ID             string                     `json:"id,omitempty"`
	Name           string                     `json:"name"`
	Description    string                     `json:"description,omitempty"`
	Query          *string                    `json:"query,omitempty"`
	Kind           models.LabelKind           `json:"kind"            enum:"custom,builtin"`
	MembershipType models.LabelMembershipType `json:"membership_type" enum:"dynamic,static,identity"`
	Platform       *string                    `json:"platform,omitempty"`
	HostsCount     int                        `json:"hosts_count,omitempty"`
	CreatedAt      time.Time                  `json:"created_at,omitempty"`
	UpdatedAt      time.Time                  `json:"updated_at,omitempty"`
}

type labelPutInput struct {
	ID   string `path:"id"`
	Body labelPutBody
}

type labelDeleteInput struct {
	ID string `path:"id"`
}

type labelMutationBody struct {
	Name           string                     `json:"name"`
	Description    string                     `json:"description,omitempty"`
	Query          *string                    `json:"query,omitempty"`
	Kind           models.LabelKind           `json:"kind,omitempty"            enum:"custom,builtin"`
	MembershipType models.LabelMembershipType `json:"membership_type,omitempty" enum:"dynamic,static,identity"`
	Platform       *string                    `json:"platform,omitempty"`
}

func (i labelListInput) params() models.LabelListParams {
	return models.LabelListParams{
		ListParams: models.CleanListParams(models.ListParams{
			Q:              i.Q,
			Page:           i.Page,
			PerPage:        i.PerPage,
			OrderKey:       i.OrderKey,
			OrderDirection: i.OrderDirection,
		}),
		Kind:           models.LabelKind(strings.TrimSpace(i.Kind)),
		MembershipType: models.LabelMembershipType(strings.TrimSpace(i.MembershipType)),
		Platform:       strings.TrimSpace(i.Platform),
	}
}

// RegisterLabels registers admin label endpoints.
func RegisterLabels(api huma.API, store *models.LabelStore) {
	registerListLabels(api, store)
	registerCreateLabel(api, store)
	registerGetLabel(api, store)
	registerUpdateLabel(api, store)
	registerDeleteLabel(api, store)
}

func registerListLabels(api huma.API, store *models.LabelStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-labels",
		Method:      http.MethodGet,
		Path:        "/api/labels",
		Tags:        []string{labelsTag},
		Summary:     "List labels",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *labelListInput) (*labelListOutput, error) {
		labels, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, err
		}
		out := &labelListOutput{Body: labelListBody{Items: make([]labelBody, 0, len(labels)), Count: count}}
		for i := range labels {
			out.Body.Items = append(out.Body.Items, labelResponse(&labels[i]))
		}
		return out, nil
	})
}

func registerCreateLabel(api huma.API, store *models.LabelStore) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-label",
		Method:        http.MethodPost,
		Path:          "/api/labels",
		Tags:          []string{labelsTag},
		Summary:       "Create a label",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict},
	}, func(ctx context.Context, input *labelCreateInput) (*labelOutput, error) {
		label, err := store.Create(ctx, models.LabelCreate{
			Name:           input.Body.Name,
			Description:    input.Body.Description,
			Query:          input.Body.Query,
			Kind:           input.Body.Kind,
			MembershipType: input.Body.MembershipType,
			Platform:       input.Body.Platform,
		})
		if err != nil {
			return nil, labelMutationError(err)
		}
		return &labelOutput{Body: labelResponse(label)}, nil
	})
}

func registerGetLabel(api huma.API, store *models.LabelStore) {
	huma.Register(api, huma.Operation{
		OperationID: "get-label",
		Method:      http.MethodGet,
		Path:        labelIDPath,
		Tags:        []string{labelsTag},
		Summary:     "Get a label",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *labelGetInput) (*labelOutput, error) {
		id, err := parseLabelID(input.ID)
		if err != nil {
			return nil, err
		}
		label, err := store.GetByID(ctx, id)
		if err != nil {
			return nil, labelMutationError(err)
		}
		return &labelOutput{Body: labelResponse(label)}, nil
	})
}

func registerUpdateLabel(api huma.API, store *models.LabelStore) {
	huma.Register(api, huma.Operation{
		OperationID: "put-label",
		Method:      http.MethodPut,
		Path:        labelIDPath,
		Tags:        []string{labelsTag},
		Summary:     "Replace a label",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *labelPutInput) (*labelOutput, error) {
		id, err := parseLabelID(input.ID)
		if err != nil {
			return nil, err
		}
		label, err := store.Update(ctx, id, models.LabelUpdate{
			Name:           input.Body.Name,
			Description:    input.Body.Description,
			Query:          input.Body.Query,
			Kind:           input.Body.Kind,
			MembershipType: input.Body.MembershipType,
			Platform:       input.Body.Platform,
		})
		if err != nil {
			return nil, labelMutationError(err)
		}
		return &labelOutput{Body: labelResponse(label)}, nil
	})
}

func registerDeleteLabel(api huma.API, store *models.LabelStore) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-label",
		Method:      http.MethodDelete,
		Path:        labelIDPath,
		Tags:        []string{labelsTag},
		Summary:     "Delete a custom label",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *labelDeleteInput) (*struct{}, error) {
		id, err := parseLabelID(input.ID)
		if err != nil {
			return nil, err
		}
		if err := store.Delete(ctx, id); err != nil {
			return nil, labelMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func labelResponse(label *models.Label) labelBody {
	return labelBody{
		ID:             strconv.FormatInt(label.ID, 10),
		Name:           label.Name,
		Description:    label.Description,
		Query:          label.Query,
		Kind:           label.Kind,
		MembershipType: label.MembershipType,
		Platform:       label.Platform,
		HostsCount:     label.HostsCount,
		CreatedAt:      label.CreatedAt,
		UpdatedAt:      label.UpdatedAt,
	}
}

func parseLabelID(id string) (int64, error) {
	return parseResourceID(id, "label")
}

func labelMutationError(err error) error {
	switch {
	case errors.Is(err, models.ErrNotFound):
		return huma.Error404NotFound("label not found")
	case errors.Is(err, models.ErrAlreadyExists):
		return huma.Error409Conflict("label already exists")
	case errors.Is(err, models.ErrInvalidInput):
		return huma.Error400BadRequest(strings.TrimPrefix(err.Error(), models.ErrInvalidInput.Error()+": "))
	default:
		return err
	}
}
