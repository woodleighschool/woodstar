package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/inventory"
)

type hostSoftwareInput struct {
	ListQueryInput

	ID     int64    `path:"id"`
	Source []string `          query:"source,omitempty"`
}

func (i hostSoftwareInput) params() (int64, inventory.HostSoftwareListParams) {
	return i.ID, inventory.HostSoftwareListParams{
		ListParams:      i.ListQueryInput.params(),
		SoftwareSources: i.Source,
	}
}

type hostSoftwareOutput struct {
	Body Page[inventory.HostSoftware]
}

func registerHostSoftware(
	api huma.API,
	softwareStore *inventory.Store,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-software",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/software",
		Tags:        []string{hostsTag},
		Summary:     "List software installed on a host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostSoftwareInput) (*hostSoftwareOutput, error) {
		id, params := input.params()
		rows, count, err := softwareStore.ListForHost(ctx, id, params)
		if err != nil {
			return nil, resourceError(ctx, logger, "list-host-software", hostResource, err, "host_id", id)
		}
		return &hostSoftwareOutput{
			Body: Page[inventory.HostSoftware]{Items: rows, Count: count},
		}, nil
	})
}
