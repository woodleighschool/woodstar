package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/apihelpers"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/platform"
)

const (
	labelsTag     = "Labels"
	labelResource = "label"
	labelIDPath   = "/api/labels/{id}"
)

type labelBody struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Query       *string   `json:"query,omitempty"`
	LabelType   string    `json:"label_type"`
	Membership  string    `json:"label_membership_type"`
	Platform    *string   `json:"platform,omitempty"`
	HostsCount  int       `json:"hosts_count"`
	CreatedAt   time.Time `json:"created_at,omitzero"`
	UpdatedAt   time.Time `json:"updated_at,omitzero"`
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
	LabelType      string `query:"label_type,omitempty"`
	MembershipType string `query:"label_membership_type,omitempty"`
	Platform       string `query:"platform,omitempty"`
}

type labelGetInput struct {
	ID string `path:"id"`
}

type labelCreateInput struct {
	Body labelCreateBody
}

type labelPutInput struct {
	ID   string `path:"id"`
	Body labelMutationBody
}

type labelDeleteInput struct {
	ID string `path:"id"`
}

type labelCreateBody struct {
	Name           string  `json:"name"`
	Description    string  `json:"description,omitempty"`
	Query          *string `json:"query,omitempty"`
	LabelType      string  `json:"label_type,omitempty"`
	MembershipType string  `json:"label_membership_type,omitempty"`
	Platform       *string `json:"platform,omitempty"`
}

type labelMutationBody struct {
	Name           string  `json:"name"`
	Description    string  `json:"description,omitempty"`
	Query          *string `json:"query,omitempty"`
	MembershipType string  `json:"label_membership_type,omitempty"`
	Platform       *string `json:"platform,omitempty"`
}

func (i labelListInput) params() labels.LabelListParams {
	return labels.LabelListParams{
		ListParams: dbutil.CleanListParams(dbutil.ListParams{
			Q:              i.Q,
			Page:           i.Page,
			PerPage:        i.PerPage,
			OrderKey:       i.OrderKey,
			OrderDirection: i.OrderDirection,
		}),
		LabelType:           strings.TrimSpace(i.LabelType),
		LabelMembershipType: strings.TrimSpace(i.MembershipType),
		Platform:            strings.TrimSpace(i.Platform),
	}
}

// RegisterLabels registers admin label endpoints.
func RegisterLabels(api huma.API, labelStore *labels.Store) {
	registerListLabels(api, labelStore)
	registerCreateLabel(api, labelStore)
	registerGetLabel(api, labelStore)
	registerUpdateLabel(api, labelStore)
	registerDeleteLabel(api, labelStore)
}

func registerListLabels(api huma.API, labelStore *labels.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-labels",
		Method:      http.MethodGet,
		Path:        "/api/labels",
		Tags:        []string{labelsTag},
		Summary:     "List labels",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *labelListInput) (*labelListOutput, error) {
		labelRows, count, err := labelStore.List(ctx, input.params())
		if err != nil {
			return nil, apihelpers.ResourceMutationError(labelResource, err)
		}
		out := &labelListOutput{Body: labelListBody{Items: make([]labelBody, 0, len(labelRows)), Count: count}}
		for i := range labelRows {
			out.Body.Items = append(out.Body.Items, labelResponse(&labelRows[i]))
		}
		return out, nil
	})
}

func registerCreateLabel(api huma.API, labelStore *labels.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-label",
		Method:        http.MethodPost,
		Path:          "/api/labels",
		Tags:          []string{labelsTag},
		Summary:       "Create a label",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict},
	}, func(ctx context.Context, input *labelCreateInput) (*labelOutput, error) {
		label, err := labelStore.Create(ctx, labels.LabelCreate{
			Name:                input.Body.Name,
			Description:         input.Body.Description,
			Query:               input.Body.Query,
			LabelType:           input.Body.LabelType,
			LabelMembershipType: input.Body.MembershipType,
			Platform:            input.Body.Platform,
		})
		if err != nil {
			return nil, apihelpers.ResourceMutationError(labelResource, err)
		}
		return &labelOutput{Body: labelResponse(label)}, nil
	})
}

func registerGetLabel(api huma.API, labelStore *labels.Store) {
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
		label, err := labelStore.GetByID(ctx, id)
		if err != nil {
			return nil, apihelpers.ResourceMutationError(labelResource, err)
		}
		return &labelOutput{Body: labelResponse(label)}, nil
	})
}

func registerUpdateLabel(api huma.API, labelStore *labels.Store) {
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
		label, err := labelStore.Update(ctx, id, labels.LabelUpdate{
			Name:                input.Body.Name,
			Description:         input.Body.Description,
			Query:               input.Body.Query,
			LabelMembershipType: input.Body.MembershipType,
			Platform:            input.Body.Platform,
		})
		if err != nil {
			return nil, apihelpers.ResourceMutationError(labelResource, err)
		}
		return &labelOutput{Body: labelResponse(label)}, nil
	})
}

func registerDeleteLabel(api huma.API, labelStore *labels.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-label",
		Method:      http.MethodDelete,
		Path:        labelIDPath,
		Tags:        []string{labelsTag},
		Summary:     "Delete a regular label",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *labelDeleteInput) (*struct{}, error) {
		id, err := parseLabelID(input.ID)
		if err != nil {
			return nil, err
		}
		if err := labelStore.Delete(ctx, id); err != nil {
			return nil, apihelpers.ResourceMutationError(labelResource, err)
		}
		return &struct{}{}, nil
	})
}

func labelResponse(label *labels.Label) labelBody {
	return labelBody{
		ID:          label.ID,
		Name:        label.Name,
		Description: label.Description,
		Query:       label.Query,
		LabelType:   label.LabelType,
		Membership:  label.LabelMembershipType,
		Platform:    labelPlatform(label.Platform),
		HostsCount:  label.HostsCount,
		CreatedAt:   label.CreatedAt,
		UpdatedAt:   label.UpdatedAt,
	}
}

func labelPlatform(platform *platform.Platform) *string {
	if platform == nil {
		return nil
	}
	value := string(*platform)
	return &value
}

func parseLabelID(id string) (int64, error) {
	return apihelpers.ParseResourceID(id, labelResource)
}
