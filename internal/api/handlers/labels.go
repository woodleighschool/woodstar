package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/labels"
)

const (
	labelsTag     = "Labels"
	labelResource = "label"
	labelIDPath   = "/api/labels/{id}"
)

type labelListOutput struct {
	Body paginatedBody[labels.Label]
}

type labelOutput struct {
	Body labels.Label
}

type labelListInput struct {
	ListQueryInput
	LabelType      labels.LabelType           `query:"label_type,omitempty"`
	MembershipType labels.LabelMembershipType `query:"label_membership_type,omitempty"`
}

type labelGetInput struct {
	ID int64 `path:"id"`
}

type labelCreateInput struct {
	Body labelCreateBody
}

type labelPutInput struct {
	ID   int64 `path:"id"`
	Body labelMutationBody
}

type labelDeleteInput struct {
	ID int64 `path:"id"`
}

type labelCreateBody struct {
	Name           string                     `json:"name"`
	Description    string                     `json:"description,omitempty"`
	Query          *string                    `json:"query,omitempty"`
	Criteria       *labels.Criteria           `json:"criteria,omitempty"`
	HostIDs        []int64                    `json:"host_ids,omitempty"`
	LabelType      labels.LabelType           `json:"label_type,omitempty"`
	MembershipType labels.LabelMembershipType `json:"label_membership_type,omitempty"`
}

type labelMutationBody struct {
	Name           string                     `json:"name"`
	Description    string                     `json:"description,omitempty"`
	Query          *string                    `json:"query,omitempty"`
	Criteria       *labels.Criteria           `json:"criteria,omitempty"`
	HostIDs        []int64                    `json:"host_ids,omitempty"`
	MembershipType labels.LabelMembershipType `json:"label_membership_type,omitempty"`
}

func (i labelListInput) params() labels.ListParams {
	return labels.ListParams{
		ListParams:          i.ListQueryInput.params(),
		LabelType:           i.LabelType,
		LabelMembershipType: i.MembershipType,
	}
}

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
		rows, count, err := labelStore.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(labelResource, err)
		}
		return &labelListOutput{Body: paginatedBody[labels.Label]{Items: rows, Count: count}}, nil
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
			Criteria:            input.Body.Criteria,
			HostIDs:             input.Body.HostIDs,
			LabelType:           input.Body.LabelType,
			LabelMembershipType: input.Body.MembershipType,
		})
		if err != nil {
			return nil, resourceMutationError(labelResource, err)
		}
		return &labelOutput{Body: *label}, nil
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
		label, err := labelStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(labelResource, err)
		}
		return &labelOutput{Body: *label}, nil
	})
}

func registerUpdateLabel(api huma.API, labelStore *labels.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-label",
		Method:      http.MethodPut,
		Path:        labelIDPath,
		Tags:        []string{labelsTag},
		Summary:     "Replace a label",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *labelPutInput) (*labelOutput, error) {
		label, err := labelStore.Update(ctx, input.ID, labels.LabelUpdate{
			Name:                input.Body.Name,
			Description:         input.Body.Description,
			Query:               input.Body.Query,
			Criteria:            input.Body.Criteria,
			HostIDs:             input.Body.HostIDs,
			LabelMembershipType: input.Body.MembershipType,
		})
		if err != nil {
			return nil, resourceMutationError(labelResource, err)
		}
		return &labelOutput{Body: *label}, nil
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
		if err := labelStore.Delete(ctx, input.ID); err != nil {
			return nil, resourceMutationError(labelResource, err)
		}
		return &struct{}{}, nil
	})
}
