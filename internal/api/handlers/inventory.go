package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
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

// RegisterInventory mounts the observed software inventory endpoints.
func RegisterInventory(
	api huma.API,
	softwareStore *inventory.Store,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	registerListInventorySoftware(api, softwareStore, logger)
	registerGetInventorySoftware(api, softwareStore, logger)
	registerHostSoftware(api, softwareStore, hostStore, logger)
}

func registerListInventorySoftware(api huma.API, softwareStore *inventory.Store, logger *slog.Logger) {
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
			return nil, resourceError(ctx, logger, "list-software", "software", err)
		}
		return &inventorySoftwareListOutput{
			Body: Page[inventory.SoftwareTitle]{Items: titles, Count: int32(count)},
		}, nil
	})
}

func registerGetInventorySoftware(api huma.API, softwareStore *inventory.Store, logger *slog.Logger) {
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
			return nil, handlerError(ctx, logger, "get-software", err, "software_id", input.SoftwareID)
		}
		return &inventorySoftwareGetOutput{Body: *title}, nil
	})
}
