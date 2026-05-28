package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
)

func RegisterHostChecks(api huma.API, checkStore *checks.Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-osquery-checks",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/checks",
		Tags:        []string{checksTag, hostsTag},
		Summary:     "List checks for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*checkHostsOutput, error) {
		host, err := hostStore.GetByID(ctx, input.ID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		rows, err := checkStore.HostChecks(ctx, host)
		if err != nil {
			return nil, err
		}
		return &checkHostsOutput{Body: itemsBody[checks.CheckHostStatus]{Items: rows}}, nil
	})
}
