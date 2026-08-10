package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/inventory"
)

const hostResource = "host"

type hostSoftwareInput struct {
	api.ListQueryInput

	ID     int64                      `path:"id"`
	Source []inventory.SoftwareSource `          query:"source,omitempty"`
}

func (i hostSoftwareInput) params() (int64, inventory.HostSoftwareListParams) {
	return i.ID, inventory.HostSoftwareListParams{
		ListParams:      i.Params(),
		SoftwareSources: i.Source,
	}
}

type hostSoftwareOutput struct {
	Body api.Page[inventory.HostSoftware]
}

func registerHostSoftware(
	humaAPI huma.API,
	softwareStore *inventory.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-host-software",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/software",
		Tags:        []string{api.TagHosts},
		Summary:     "List software for a host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostSoftwareInput) (*hostSoftwareOutput, error) {
		id, params := input.params()
		rows, count, err := softwareStore.ListForHost(ctx, id, params)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-host-software", hostResource, err, "host_id", id)
		}
		return &hostSoftwareOutput{
			Body: api.Page[inventory.HostSoftware]{Items: rows, Count: count},
		}, nil
	})
}
