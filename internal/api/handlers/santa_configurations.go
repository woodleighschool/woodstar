package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

const (
	santaConfigurationResource = "Santa configuration"
	santaConfigurationIDPath   = "/api/santa/configurations/{id}"
)

type santaConfigurationListInput struct {
	ListQueryInput
}

type santaConfigurationGetInput struct {
	ConfigurationID int64 `path:"id"`
}

type santaConfigurationCreateInput struct {
	Body configurations.ConfigurationMutation
}

type santaConfigurationUpdateInput struct {
	ConfigurationID int64 `path:"id"`
	Body            configurations.ConfigurationMutation
}

type santaConfigurationDeleteInput struct {
	ConfigurationID int64 `path:"id"`
}

type santaConfigurationBulkDeleteInput struct {
	Body BulkIDsBody
}

type santaConfigurationReorderInput struct {
	Body santaConfigurationReorderBody
}

type santaConfigurationReorderBody struct {
	OrderedIDs []int64 `json:"ordered_ids"`
}

type santaConfigurationListOutput struct {
	Body Page[configurations.Configuration]
}

type santaConfigurationOutput struct {
	Body configurations.Configuration
}

func (input santaConfigurationListInput) params() configurations.ConfigurationListParams {
	return configurations.ConfigurationListParams{
		ListParams: input.ListQueryInput.Params(),
	}
}

func registerSantaConfigurations(api huma.API, store *configurations.Store, logger *slog.Logger) {
	registerListSantaConfigurations(api, store, logger)
	registerCreateSantaConfiguration(api, store, logger)
	registerGetSantaConfiguration(api, store, logger)
	registerUpdateSantaConfiguration(api, store, logger)
	registerDeleteSantaConfiguration(api, store, logger)
	registerBulkDeleteSantaConfigurations(api, store, logger)
	registerReorderSantaConfigurations(api, store, logger)
}

func registerListSantaConfigurations(api huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-configurations",
		Method:      http.MethodGet,
		Path:        "/api/santa/configurations",
		Tags:        []string{santaTag},
		Summary:     "List Santa configurations",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *santaConfigurationListInput) (*santaConfigurationListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, handlerError(ctx, logger, "list-santa-configurations", santaConfigurationMutationError(err))
		}
		return &santaConfigurationListOutput{
			Body: Page[configurations.Configuration]{Items: rows, Count: int32(count)},
		}, nil
	})
}

func registerCreateSantaConfiguration(api huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-santa-configuration",
		Method:        http.MethodPost,
		Path:          "/api/santa/configurations",
		Tags:          []string{santaTag},
		Summary:       "Create a Santa configuration",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *santaConfigurationCreateInput) (*santaConfigurationOutput, error) {
		configuration, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, handlerError(ctx, logger, "create-santa-configuration", santaConfigurationMutationError(err))
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerGetSantaConfiguration(api huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-configuration",
		Method:      http.MethodGet,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaTag},
		Summary:     "Get a Santa configuration",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *santaConfigurationGetInput) (*santaConfigurationOutput, error) {
		configuration, err := store.GetByID(ctx, input.ConfigurationID)
		if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"get-santa-configuration",
				santaConfigurationMutationError(err),
				"configuration_id", input.ConfigurationID,
			)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerUpdateSantaConfiguration(api huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "update-santa-configuration",
		Method:      http.MethodPut,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaTag},
		Summary:     "Update a Santa configuration",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *santaConfigurationUpdateInput) (*santaConfigurationOutput, error) {
		configuration, err := store.Update(ctx, input.ConfigurationID, input.Body)
		if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"update-santa-configuration",
				santaConfigurationMutationError(err),
				"configuration_id", input.ConfigurationID,
			)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerDeleteSantaConfiguration(api huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-configuration",
		Method:      http.MethodDelete,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaTag},
		Summary:     "Delete a Santa configuration",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaConfigurationDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ConfigurationID); err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"delete-santa-configuration",
				santaConfigurationMutationError(err),
				"configuration_id", input.ConfigurationID,
			)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteSantaConfigurations(api huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-santa-configurations",
		Method:      http.MethodPost,
		Path:        "/api/santa/configurations/bulk-delete",
		Tags:        []string{santaTag},
		Summary:     "Delete Santa configurations",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaConfigurationBulkDeleteInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"bulk-delete-santa-configurations",
				santaConfigurationMutationError(err),
			)
		}
		return &struct{}{}, nil
	})
}

func registerReorderSantaConfigurations(api huma.API, store *configurations.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-santa-configurations",
		Method:      http.MethodPut,
		Path:        "/api/santa/configurations/order",
		Tags:        []string{santaTag},
		Summary:     "Reorder Santa configurations",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaConfigurationReorderInput) (*struct{}, error) {
		if err := store.ReorderConfigurations(ctx, input.Body.OrderedIDs); err != nil {
			return nil, handlerError(ctx, logger, "reorder-santa-configurations", santaConfigurationMutationError(err))
		}
		return &struct{}{}, nil
	})
}

func santaConfigurationMutationError(err error) error {
	return ResourceMutationError(santaConfigurationResource, err)
}
