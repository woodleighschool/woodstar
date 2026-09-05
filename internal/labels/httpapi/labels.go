package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/rbac"
)

const (
	labelResource = "label"
	labelIDPath   = "/api/labels/{id}"
)

type labelListOutput struct {
	Body api.Page[labels.Label]
}

type labelOutput struct {
	Body labels.Label
}

type labelListInput struct {
	api.ListQueryInput

	LabelType      labels.LabelType             `query:"label_type,omitempty"`
	MembershipType []labels.LabelMembershipType `query:"label_membership_type,omitempty"`
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
		ListParams:           i.Params(),
		LabelType:            i.LabelType,
		LabelMembershipTypes: i.MembershipType,
	}
}

// RegisterAPI mounts label management endpoints.
func RegisterAPI(
	routes api.AppRoutes,
	labelStore *labels.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	registerAPI(routes, labelStore, authorizer, logger)
}

// RegisterOpenAPI documents label endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, nil, nil, nil)
}

func registerAPI(
	routes api.AppRoutes,
	labelStore *labels.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	humaAPI := authhuma.ResourceAPI(routes.Protected, authorizer, logger, rbac.ResourceLabels)
	registerListLabels(humaAPI, labelStore, logger)
	registerCreateLabel(humaAPI, labelStore, logger)
	registerGetLabel(humaAPI, labelStore, logger)
	registerUpdateLabel(humaAPI, labelStore, logger)
	registerDeleteLabel(humaAPI, labelStore, logger)
}

func registerListLabels(humaAPI huma.API, labelStore *labels.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-labels",
		Method:      http.MethodGet,
		Path:        "/api/labels",
		Tags:        []string{api.TagLabels},
		Summary:     "List labels",
	}, func(ctx context.Context, input *labelListInput) (*labelListOutput, error) {
		rows, count, err := labelStore.List(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-labels", labelResource, err)
		}
		return &labelListOutput{Body: api.Page[labels.Label]{Items: rows, Count: count}}, nil
	})
}

func registerCreateLabel(humaAPI huma.API, labelStore *labels.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-label",
		Method:        http.MethodPost,
		Path:          "/api/labels",
		Tags:          []string{api.TagLabels},
		Summary:       "Create a label",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *labelCreateInput) (*labelOutput, error) {
		label, err := labelStore.Create(ctx, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-label", labelResource, err)
		}
		return &labelOutput{Body: *label}, nil
	})
}

func registerGetLabel(humaAPI huma.API, labelStore *labels.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-label",
		Method:      http.MethodGet,
		Path:        labelIDPath,
		Tags:        []string{api.TagLabels},
		Summary:     "Get a label",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *labelGetInput) (*labelOutput, error) {
		label, err := labelStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-label", labelResource, err, "label_id", input.ID)
		}
		return &labelOutput{Body: *label}, nil
	})
}

func registerUpdateLabel(humaAPI huma.API, labelStore *labels.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-label",
		Method:      http.MethodPut,
		Path:        labelIDPath,
		Tags:        []string{api.TagLabels},
		Summary:     "Update a label",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *labelPutInput) (*labelOutput, error) {
		label, err := labelStore.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "update-label", labelResource, err, "label_id", input.ID)
		}
		return &labelOutput{Body: *label}, nil
	})
}

func registerDeleteLabel(humaAPI huma.API, labelStore *labels.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-label",
		Method:      http.MethodDelete,
		Path:        labelIDPath,
		Tags:        []string{api.TagLabels},
		Summary:     "Delete a label",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *labelDeleteInput) (*struct{}, error) {
		if err := labelStore.Delete(ctx, input.ID); err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-label", labelResource, err, "label_id", input.ID)
		}
		return &struct{}{}, nil
	})
}
