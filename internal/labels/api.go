package labels

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
)

const (
	labelsTag     = "Labels"
	labelResource = "label"
	labelIDPath   = "/api/labels/{id}"
)

type labelListOutput struct {
	Body apitypes.Page[Label]
}

type labelOutput struct {
	Body Label
}

type labelListInput struct {
	apitypes.ListQueryInput
	LabelType      LabelType           `query:"label_type,omitempty"`
	MembershipType LabelMembershipType `query:"label_membership_type,omitempty"`
}

type labelGetInput struct {
	ID int64 `path:"id"`
}

type labelCreateInput struct {
	Body LabelMutation
}

type labelPutInput struct {
	ID   int64 `path:"id"`
	Body LabelMutation
}

type labelDeleteInput struct {
	ID int64 `path:"id"`
}

func (i labelListInput) params() LabelListParams {
	return LabelListParams{
		ListParams:          i.ListQueryInput.Params(),
		LabelType:           i.LabelType,
		LabelMembershipType: i.MembershipType,
	}
}

func RegisterAdminRoutes(api huma.API, labelStore *Store) {
	registerListLabels(api, labelStore)
	registerCreateLabel(api, labelStore)
	registerGetLabel(api, labelStore)
	registerUpdateLabel(api, labelStore)
	registerDeleteLabel(api, labelStore)
}

func registerListLabels(api huma.API, labelStore *Store) {
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
			return nil, apitypes.ResourceMutationError(labelResource, err)
		}
		return &labelListOutput{Body: apitypes.Page[Label]{Items: rows, Count: count}}, nil
	})
}

func registerCreateLabel(api huma.API, labelStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-label",
		Method:        http.MethodPost,
		Path:          "/api/labels",
		Tags:          []string{labelsTag},
		Summary:       "Create a label",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *labelCreateInput) (*labelOutput, error) {
		label, err := labelStore.Create(ctx, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(labelResource, err)
		}
		return &labelOutput{Body: *label}, nil
	})
}

func registerGetLabel(api huma.API, labelStore *Store) {
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
			return nil, apitypes.ResourceMutationError(labelResource, err)
		}
		return &labelOutput{Body: *label}, nil
	})
}

func registerUpdateLabel(api huma.API, labelStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-label",
		Method:      http.MethodPut,
		Path:        labelIDPath,
		Tags:        []string{labelsTag},
		Summary:     "Replace a label",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *labelPutInput) (*labelOutput, error) {
		label, err := labelStore.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(labelResource, err)
		}
		return &labelOutput{Body: *label}, nil
	})
}

func registerDeleteLabel(api huma.API, labelStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-label",
		Method:      http.MethodDelete,
		Path:        labelIDPath,
		Tags:        []string{labelsTag},
		Summary:     "Delete a regular label",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *labelDeleteInput) (*struct{}, error) {
		if err := labelStore.Delete(ctx, input.ID); err != nil {
			return nil, apitypes.ResourceMutationError(labelResource, err)
		}
		return &struct{}{}, nil
	})
}
