package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa"
)

const (
	santaConfigurationsTag     = "Santa"
	santaConfigurationResource = "Santa configuration"
	santaConfigurationIDPath   = "/api/santa/configurations/{id}"
)

type santaConfigurationListInput struct {
	Q              string `query:"q,omitempty"`
	Page           int    `query:"page,omitempty"`
	PerPage        int    `query:"per_page,omitempty"`
	OrderKey       string `query:"order_key,omitempty"`
	OrderDirection string `query:"order_direction,omitempty"`
}

type santaConfigurationGetInput struct {
	ID string `path:"id"`
}

type santaConfigurationCreateInput struct {
	Body santa.ConfigurationCreate
}

type santaConfigurationPatchInput struct {
	ID   string `path:"id"`
	Body santa.ConfigurationUpdate
}

type santaConfigurationDeleteInput struct {
	ID string `path:"id"`
}

type santaConfigurationReorderInput struct {
	Body santaConfigurationReorderBody
}

type santaConfigurationReorderBody struct {
	OrderedIDs []int64 `json:"ordered_ids"`
}

type santaConfigurationListOutput struct {
	Body paginatedBody[santa.Configuration]
}

type santaConfigurationOutput struct {
	Body santa.Configuration
}

type santaConfigurationConflictError struct {
	Code              string `json:"code"`
	LabelID           int64  `json:"label_id"`
	ConfigurationID   int64  `json:"configuration_id"`
	ConfigurationName string `json:"configuration_name"`
}

func (e santaConfigurationConflictError) Error() string {
	return "configuration label already belongs to another configuration"
}

func (e santaConfigurationConflictError) GetStatus() int {
	return http.StatusConflict
}

func (input santaConfigurationListInput) params() santa.ConfigurationListParams {
	return santa.ConfigurationListParams{
		ListParams: dbutil.ListParams{
			Q:              input.Q,
			Page:           input.Page,
			PerPage:        input.PerPage,
			OrderKey:       input.OrderKey,
			OrderDirection: input.OrderDirection,
		},
	}
}

func RegisterSantaConfigurations(api huma.API, store *santa.Store) {
	registerListSantaConfigurations(api, store)
	registerCreateSantaConfiguration(api, store)
	registerGetSantaConfiguration(api, store)
	registerPatchSantaConfiguration(api, store)
	registerDeleteSantaConfiguration(api, store)
	registerReorderSantaConfigurations(api, store)
}

func registerListSantaConfigurations(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-configurations",
		Method:      http.MethodGet,
		Path:        "/api/santa/configurations",
		Tags:        []string{santaConfigurationsTag},
		Summary:     "List Santa configurations",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaConfigurationListInput) (*santaConfigurationListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		configurations, count, err := store.ListConfigurations(ctx, input.params())
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationListOutput{
			Body: paginatedBody[santa.Configuration]{Items: configurations, Count: count},
		}, nil
	})
}

func registerCreateSantaConfiguration(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-santa-configuration",
		Method:        http.MethodPost,
		Path:          "/api/santa/configurations",
		Tags:          []string{santaConfigurationsTag},
		Summary:       "Create a Santa configuration",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusConflict},
	}, func(ctx context.Context, input *santaConfigurationCreateInput) (*santaConfigurationOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		configuration, err := store.CreateConfiguration(ctx, input.Body)
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerGetSantaConfiguration(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-configuration",
		Method:      http.MethodGet,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaConfigurationsTag},
		Summary:     "Get a Santa configuration",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaConfigurationGetInput) (*santaConfigurationOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, santaConfigurationResource)
		if err != nil {
			return nil, err
		}
		configuration, err := store.GetConfigurationByID(ctx, id)
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerPatchSantaConfiguration(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-santa-configuration",
		Method:      http.MethodPatch,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaConfigurationsTag},
		Summary:     "Update a Santa configuration",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *santaConfigurationPatchInput) (*santaConfigurationOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, santaConfigurationResource)
		if err != nil {
			return nil, err
		}
		configuration, err := store.UpdateConfiguration(ctx, id, input.Body)
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerDeleteSantaConfiguration(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-configuration",
		Method:      http.MethodDelete,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaConfigurationsTag},
		Summary:     "Delete a Santa configuration",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaConfigurationDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, santaConfigurationResource)
		if err != nil {
			return nil, err
		}
		if err := store.DeleteConfiguration(ctx, id); err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func registerReorderSantaConfigurations(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-santa-configurations",
		Method:      http.MethodPut,
		Path:        "/api/santa/configurations/order",
		Tags:        []string{santaConfigurationsTag},
		Summary:     "Reorder Santa configurations",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaConfigurationReorderInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if err := store.ReorderConfigurations(ctx, input.Body.OrderedIDs); err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func santaConfigurationMutationError(err error) error {
	var conflict *santa.ConfigurationLabelConflictError
	if errors.As(err, &conflict) {
		return santaConfigurationConflictError{
			Code:              conflict.Code,
			LabelID:           conflict.LabelID,
			ConfigurationID:   conflict.ConfigurationID,
			ConfigurationName: conflict.ConfigurationName,
		}
	}
	return resourceMutationError(santaConfigurationResource, err)
}
