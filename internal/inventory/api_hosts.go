package inventory

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

const hostsTag = "Hosts"

type hostSoftwareInput struct {
	apitypes.ListQueryInput

	HostID int64    `path:"id"`
	Source []string `query:"source,omitempty"`
}

func (i hostSoftwareInput) params() (int64, HostSoftwareListParams) {
	return i.HostID, HostSoftwareListParams{
		ListParams:      i.ListQueryInput.Params(),
		SoftwareSources: i.Source,
	}
}

type hostSoftwareOutput struct {
	Body apitypes.Page[HostSoftwareRow]
}

// RegisterHostAdminRoutes registers software inventory endpoints for a host.
func RegisterHostAdminRoutes(api huma.API, softwareStore *Store, hostStore *hosts.Store) {
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
			return nil, err
		}
		rows, count, err := softwareStore.ListForHost(ctx, id, params)
		if err != nil {
			return nil, apitypes.ResourceMutationError("software", err)
		}
		return &hostSoftwareOutput{Body: apitypes.Page[HostSoftwareRow]{Items: rows, Count: int32(count)}}, nil
	})
}
