package configurations

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
)

const (
	santaTag                   = "Santa"
	santaConfigurationResource = "Santa configuration"
	santaConfigurationIDPath   = "/api/santa/configurations/{id}"
)

type santaConfigurationListInput struct {
	apitypes.ListQueryInput
}

type santaConfigurationGetInput struct {
	ID int64 `path:"id"`
}

type santaConfigurationCreateInput struct {
	Body ConfigurationMutation
}

type santaConfigurationUpdateInput struct {
	ID   int64 `path:"id"`
	Body ConfigurationMutation
}

type santaConfigurationDeleteInput struct {
	ID int64 `path:"id"`
}

type santaConfigurationBulkDeleteInput struct {
	Body apitypes.BulkIDsBody
}

type santaConfigurationReorderInput struct {
	Body santaConfigurationReorderBody
}

type santaConfigurationReorderBody struct {
	OrderedIDs []int64 `json:"ordered_ids"`
}

type santaConfigurationListOutput struct {
	Body apitypes.Page[Configuration]
}

type santaConfigurationOutput struct {
	Body Configuration
}

func (input santaConfigurationListInput) params() ConfigurationListParams {
	return ConfigurationListParams{
		ListParams: input.ListQueryInput.Params(),
	}
}

func RegisterAdminRoutes(api huma.API, store *Store) {
	registerListSantaConfigurations(api, store)
	registerCreateSantaConfiguration(api, store)
	registerGetSantaConfiguration(api, store)
	registerUpdateSantaConfiguration(api, store)
	registerDeleteSantaConfiguration(api, store)
	registerBulkDeleteSantaConfigurations(api, store)
	registerReorderSantaConfigurations(api, store)
}

func registerListSantaConfigurations(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-configurations",
		Method:      http.MethodGet,
		Path:        "/api/santa/configurations",
		Tags:        []string{santaTag},
		Summary:     "List Santa configurations",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaConfigurationListInput) (*santaConfigurationListOutput, error) {
		rows, count, err := store.ListConfigurations(ctx, input.params())
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationListOutput{
			Body: apitypes.Page[Configuration]{Items: rows, Count: count},
		}, nil
	})
}

func registerCreateSantaConfiguration(api huma.API, store *Store) {
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
		configuration, err := store.CreateConfiguration(ctx, input.Body)
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerGetSantaConfiguration(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-configuration",
		Method:      http.MethodGet,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaTag},
		Summary:     "Get a Santa configuration",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaConfigurationGetInput) (*santaConfigurationOutput, error) {
		configuration, err := store.GetConfigurationByID(ctx, input.ID)
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerUpdateSantaConfiguration(api huma.API, store *Store) {
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
		configuration, err := store.UpdateConfiguration(ctx, input.ID, input.Body)
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerDeleteSantaConfiguration(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-configuration",
		Method:      http.MethodDelete,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaTag},
		Summary:     "Delete a Santa configuration",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaConfigurationDeleteInput) (*struct{}, error) {
		if err := store.DeleteConfiguration(ctx, input.ID); err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteSantaConfigurations(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-santa-configurations",
		Method:      http.MethodPost,
		Path:        "/api/santa/configurations/bulk-delete",
		Tags:        []string{santaTag},
		Summary:     "Delete Santa configurations",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaConfigurationBulkDeleteInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func registerReorderSantaConfigurations(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-santa-configurations",
		Method:      http.MethodPut,
		Path:        "/api/santa/configurations/order",
		Tags:        []string{santaTag},
		Summary:     "Reorder Santa configurations",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaConfigurationReorderInput) (*struct{}, error) {
		if err := store.ReorderConfigurations(ctx, input.Body.OrderedIDs); err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func santaConfigurationMutationError(err error) error {
	return apitypes.ResourceMutationError(santaConfigurationResource, err)
}
