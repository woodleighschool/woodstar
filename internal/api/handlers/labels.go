package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/labels"
)

const (
	labelResource = "label"
	labelIDPath   = "/api/labels/{id}"
)

type labelListOutput struct {
	Body Page[labels.Label]
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
	Body labels.LabelMutation
}

type labelPutInput struct {
	ID   int64 `path:"id"`
	Body labels.LabelMutation
}

type labelDeleteInput struct {
	ID int64 `path:"id"`
}

func (i labelListInput) params() labels.LabelListParams {
	return labels.LabelListParams{
		ListParams:          i.ListQueryInput.params(),
		LabelType:           i.LabelType,
		LabelMembershipType: i.MembershipType,
	}
}

// RegisterLabels mounts label management endpoints.
func RegisterLabels(api huma.API, labelStore *labels.Store, logger *slog.Logger) {
	registerListLabels(api, labelStore, logger)
	registerCreateLabel(api, labelStore, logger)
	registerGetLabel(api, labelStore, logger)
	registerUpdateLabel(api, labelStore, logger)
	registerDeleteLabel(api, labelStore, logger)
}

func registerListLabels(api huma.API, labelStore *labels.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-labels",
		Method:      http.MethodGet,
		Path:        "/api/labels",
		Tags:        []string{labelsTag},
		Summary:     "List labels",
	}, func(ctx context.Context, input *labelListInput) (*labelListOutput, error) {
		rows, count, err := labelStore.List(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-labels", labelResource, err)
		}
		return &labelListOutput{Body: Page[labels.Label]{Items: rows, Count: count}}, nil
	})
}

func registerCreateLabel(api huma.API, labelStore *labels.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-label",
		Method:        http.MethodPost,
		Path:          "/api/labels",
		Tags:          []string{labelsTag},
		Summary:       "Create a label",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *labelCreateInput) (*labelOutput, error) {
		label, err := labelStore.Create(ctx, input.Body)
		if err != nil {
			return nil, resourceError(ctx, logger, "create-label", labelResource, err)
		}
		return &labelOutput{Body: *label}, nil
	})
}

func registerGetLabel(api huma.API, labelStore *labels.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-label",
		Method:      http.MethodGet,
		Path:        labelIDPath,
		Tags:        []string{labelsTag},
		Summary:     "Get a label",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *labelGetInput) (*labelOutput, error) {
		label, err := labelStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceError(ctx, logger, "get-label", labelResource, err, "label_id", input.ID)
		}
		return &labelOutput{Body: *label}, nil
	})
}

func registerUpdateLabel(api huma.API, labelStore *labels.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "update-label",
		Method:      http.MethodPut,
		Path:        labelIDPath,
		Tags:        []string{labelsTag},
		Summary:     "Update a label",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *labelPutInput) (*labelOutput, error) {
		label, err := labelStore.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceError(ctx, logger, "update-label", labelResource, err, "label_id", input.ID)
		}
		return &labelOutput{Body: *label}, nil
	})
}

func registerDeleteLabel(api huma.API, labelStore *labels.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-label",
		Method:      http.MethodDelete,
		Path:        labelIDPath,
		Tags:        []string{labelsTag},
		Summary:     "Delete a label",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *labelDeleteInput) (*struct{}, error) {
		if err := labelStore.Delete(ctx, input.ID); err != nil {
			return nil, resourceError(ctx, logger, "delete-label", labelResource, err, "label_id", input.ID)
		}
		return &struct{}{}, nil
	})
}
