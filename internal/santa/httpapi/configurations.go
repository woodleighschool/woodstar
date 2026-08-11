package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

const (
	santaConfigurationResource = "Santa configuration"
	santaConfigurationIDPath   = "/api/santa/configurations/{id}"
)

type santaConfigurationListInput struct {
	api.ListQueryInput
}

type santaConfigurationGetInput struct {
	ID int64 `path:"id"`
}

type santaConfigurationCreateInput struct {
	Body configurations.ConfigurationMutation
}

type santaConfigurationUpdateInput struct {
	ID   int64 `path:"id"`
	Body configurations.ConfigurationMutation
}

type santaConfigurationDeleteInput struct {
	ID int64 `path:"id"`
}

type santaConfigurationReorderInput struct {
	Body santaConfigurationReorderBody
}

type santaConfigurationReorderBody struct {
	OrderedIDs []int64 `json:"ordered_ids"`
}

type santaConfigurationListOutput struct {
	Body api.Page[configurations.Configuration]
}

type santaConfigurationOutput struct {
	Body configurations.Configuration
}

func (input santaConfigurationListInput) params() configurations.ConfigurationListParams {
	return configurations.ConfigurationListParams{
		ListParams: input.Params(),
	}
}

func registerSantaConfigurations(humaAPI huma.API, store *configurations.Store, logger *slog.Logger) {
	registerListSantaConfigurations(humaAPI, store, logger)
	registerCreateSantaConfiguration(humaAPI, store, logger)
	registerGetSantaConfiguration(humaAPI, store, logger)
	registerUpdateSantaConfiguration(humaAPI, store, logger)
	registerDeleteSantaConfiguration(humaAPI, store, logger)
	registerBulkDeleteSantaConfigurations(humaAPI, store, logger)
	registerReorderSantaConfigurations(humaAPI, store, logger)
}

func registerListSantaConfigurations(humaAPI huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-santa-configurations",
		Method:      http.MethodGet,
		Path:        "/api/santa/configurations",
		Tags:        []string{api.TagSantaConfigurations},
		Summary:     "List configurations",
	}, func(ctx context.Context, input *santaConfigurationListInput) (*santaConfigurationListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"list-santa-configurations",
				api.ResourceMutationError(santaConfigurationResource, err),
			)
		}
		return &santaConfigurationListOutput{
			Body: api.Page[configurations.Configuration]{Items: rows, Count: count},
		}, nil
	})
}

func registerCreateSantaConfiguration(humaAPI huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-santa-configuration",
		Method:        http.MethodPost,
		Path:          "/api/santa/configurations",
		Tags:          []string{api.TagSantaConfigurations},
		Summary:       "Create a configuration",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *santaConfigurationCreateInput) (*santaConfigurationOutput, error) {
		configuration, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"create-santa-configuration",
				api.ResourceMutationError(santaConfigurationResource, err),
			)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerGetSantaConfiguration(humaAPI huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-santa-configuration",
		Method:      http.MethodGet,
		Path:        santaConfigurationIDPath,
		Tags:        []string{api.TagSantaConfigurations},
		Summary:     "Get a configuration",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *santaConfigurationGetInput) (*santaConfigurationOutput, error) {
		configuration, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"get-santa-configuration",
				api.ResourceMutationError(santaConfigurationResource, err),
				"id", input.ID,
			)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerUpdateSantaConfiguration(humaAPI huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-santa-configuration",
		Method:      http.MethodPut,
		Path:        santaConfigurationIDPath,
		Tags:        []string{api.TagSantaConfigurations},
		Summary:     "Update a configuration",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *santaConfigurationUpdateInput) (*santaConfigurationOutput, error) {
		configuration, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"update-santa-configuration",
				api.ResourceMutationError(santaConfigurationResource, err),
				"id", input.ID,
			)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerDeleteSantaConfiguration(humaAPI huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-santa-configuration",
		Method:      http.MethodDelete,
		Path:        santaConfigurationIDPath,
		Tags:        []string{api.TagSantaConfigurations},
		Summary:     "Delete a configuration",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *santaConfigurationDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ID); err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"delete-santa-configuration",
				api.ResourceMutationError(santaConfigurationResource, err),
				"id", input.ID,
			)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteSantaConfigurations(humaAPI huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "bulk-delete-santa-configurations",
		Method:        http.MethodDelete,
		Path:          "/api/santa/configurations",
		Tags:          []string{api.TagSantaConfigurations},
		Summary:       "Delete configurations",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *api.DeleteManyInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.IDs); err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"bulk-delete-santa-configurations",
				api.ResourceMutationError(santaConfigurationResource, err),
			)
		}
		return &struct{}{}, nil
	})
}

func registerReorderSantaConfigurations(humaAPI huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "reorder-santa-configurations",
		Method:      http.MethodPut,
		Path:        "/api/santa/configurations/order",
		Tags:        []string{api.TagSantaConfigurations},
		Summary:     "Reorder configurations",
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *santaConfigurationReorderInput) (*struct{}, error) {
		if err := store.ReorderConfigurations(ctx, input.Body.OrderedIDs); err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"reorder-santa-configurations",
				api.ResourceMutationError(santaConfigurationResource, err),
			)
		}
		return &struct{}{}, nil
	})
}
