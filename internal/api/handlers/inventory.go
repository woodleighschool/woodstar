package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/inventory"
)

type inventorySoftwareListInput struct {
	ListQueryInput

	Source []string `query:"source,omitempty"`
}

func (i inventorySoftwareListInput) params() inventory.SoftwareTitleListParams {
	return inventory.SoftwareTitleListParams{
		ListParams:      i.ListQueryInput.Params(),
		SoftwareSources: i.Source,
	}
}

type inventorySoftwareGetInput struct {
	SoftwareID int64 `path:"id"`
}

type inventorySoftwareListOutput struct {
	Body Page[inventory.SoftwareTitle]
}

type inventorySoftwareGetOutput struct {
	Body inventory.SoftwareTitle
}

func registerInventory(g Groups, deps Dependencies) {
	registerListInventorySoftware(g.Ordinary, deps.Software)
	registerGetInventorySoftware(g.Ordinary, deps.Software)
	registerHostSoftware(g.Ordinary, deps.Software, deps.Hosts)
}

func registerListInventorySoftware(api huma.API, softwareStore *inventory.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-software",
		Method:      http.MethodGet,
		Path:        "/api/software",
		Tags:        []string{softwareTag},
		Summary:     "List software titles",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *inventorySoftwareListInput) (*inventorySoftwareListOutput, error) {
		titles, count, err := softwareStore.ListTitles(ctx, input.params())
		if err != nil {
			return nil, ResourceMutationError("software", err)
		}
		return &inventorySoftwareListOutput{
			Body: Page[inventory.SoftwareTitle]{Items: titles, Count: int32(count)},
		}, nil
	})
}

func registerGetInventorySoftware(api huma.API, softwareStore *inventory.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-software",
		Method:      http.MethodGet,
		Path:        "/api/software/{id}",
		Tags:        []string{softwareTag},
		Summary:     "Get a software title",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *inventorySoftwareGetInput) (*inventorySoftwareGetOutput, error) {
		title, err := softwareStore.GetTitle(ctx, input.SoftwareID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("software title not found")
		}
		if err != nil {
			return nil, err
		}
		return &inventorySoftwareGetOutput{Body: *title}, nil
	})
}
