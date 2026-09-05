package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/rbac"
)

type inventorySoftwareListInput struct {
	api.ListQueryInput

	Source []inventory.SoftwareSource `query:"source,omitempty"`
}

func (i inventorySoftwareListInput) params() inventory.SoftwareTitleListParams {
	return inventory.SoftwareTitleListParams{
		ListParams:      i.Params(),
		SoftwareSources: i.Source,
	}
}

type inventorySoftwareGetInput struct {
	ID int64 `path:"id"`
}

type inventorySoftwareListOutput struct {
	Body api.Page[inventory.SoftwareTitle]
}

type inventorySoftwareGetOutput struct {
	Body inventory.SoftwareTitle
}

// RegisterAPI mounts the observed software inventory endpoints.
func RegisterAPI(
	routes api.AppRoutes,
	softwareStore *inventory.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	registerAPI(routes, softwareStore, authorizer, logger)
}

// RegisterOpenAPI documents software inventory endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, nil, nil, nil)
}

func registerAPI(
	routes api.AppRoutes,
	softwareStore *inventory.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	humaAPI := authhuma.ResourceAPI(routes.Protected, authorizer, logger, rbac.ResourceSoftware)
	registerListInventorySoftware(humaAPI, softwareStore, logger)
	registerGetInventorySoftware(humaAPI, softwareStore, logger)
	registerHostSoftware(humaAPI, softwareStore, logger)
}

func registerListInventorySoftware(humaAPI huma.API, softwareStore *inventory.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-software",
		Method:      http.MethodGet,
		Path:        "/api/software",
		Tags:        []string{api.TagSoftware},
		Summary:     "List software titles",
	}, func(ctx context.Context, input *inventorySoftwareListInput) (*inventorySoftwareListOutput, error) {
		titles, count, err := softwareStore.ListTitles(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-software", "software", err)
		}
		return &inventorySoftwareListOutput{
			Body: api.Page[inventory.SoftwareTitle]{Items: titles, Count: count},
		}, nil
	})
}

func registerGetInventorySoftware(humaAPI huma.API, softwareStore *inventory.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-software",
		Method:      http.MethodGet,
		Path:        "/api/software/{id}",
		Tags:        []string{api.TagSoftware},
		Summary:     "Get a software title",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *inventorySoftwareGetInput) (*inventorySoftwareGetOutput, error) {
		title, err := softwareStore.GetTitle(ctx, input.ID)
		if errors.Is(err, fault.ErrNotFound) {
			return nil, huma.Error404NotFound("software title not found")
		}
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "get-software", err, "software_id", input.ID)
		}
		return &inventorySoftwareGetOutput{Body: *title}, nil
	})
}
