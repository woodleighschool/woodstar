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

type hostSoftwareInput struct {
	ListQueryInput

	HostID int64    `path:"id"`
	Source []string `          query:"source,omitempty"`
}

func (i hostSoftwareInput) params() (int64, inventory.HostSoftwareListParams) {
	return i.HostID, inventory.HostSoftwareListParams{
		ListParams:      i.ListQueryInput.Params(),
		SoftwareSources: i.Source,
	}
}

type hostSoftwareOutput struct {
	Body Page[inventory.HostSoftwareRow]
}

func registerHostSoftware(
	api huma.API,
	softwareStore *inventory.Store,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-software",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/software",
		Tags:        []string{hostsTag},
		Summary:     "List software installed on a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostSoftwareInput) (*hostSoftwareOutput, error) {
		id, params := input.params()
		if _, err := hostStore.GetByID(ctx, id); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, handlerError(ctx, logger, "list-host-software", err, "host_id", id)
		}
		rows, count, err := softwareStore.ListForHost(ctx, id, params)
		if err != nil {
			return nil, resourceError(ctx, logger, "list-host-software", "software", err, "host_id", id)
		}
		return &hostSoftwareOutput{
			Body: Page[inventory.HostSoftwareRow]{Items: rows, Count: int32(count)},
		}, nil
	})
}
