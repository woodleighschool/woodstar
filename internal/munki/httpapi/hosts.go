package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/munki"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

type hostMunkiStateInput struct {
	ID int64 `path:"id"`
}

type hostMunkiStateOutput struct {
	Body munki.HostState
}

func registerHostMunkiState(
	humaAPI huma.API,
	store *munki.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-host-munki-state",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/munki",
		Tags:        []string{api.TagHosts},
		Summary:     "Get Munki state for a host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostMunkiStateInput) (*hostMunkiStateOutput, error) {
		state, err := store.LoadHostState(ctx, input.ID)
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"get-host-munki-state",
				err,
				"host_id", input.ID,
			)
		}
		if state == nil {
			return nil, huma.Error404NotFound("")
		}
		return &hostMunkiStateOutput{Body: *state}, nil
	})
}

type hostMunkiSoftwareInput struct {
	api.ListQueryInput

	ID int64 `path:"id"`
}

type hostMunkiSoftwareOutput struct {
	Body api.Page[munkisoftware.HostManifestSoftware]
}

func registerHostMunkiSoftware(
	humaAPI huma.API,
	store *munkisoftware.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-host-munki-software",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/munki/software",
		Tags:        []string{api.TagHosts},
		Summary:     "List Munki software for a host",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *hostMunkiSoftwareInput) (*hostMunkiSoftwareOutput, error) {
		rows, count, err := store.ListForHost(ctx, input.ID, munkisoftware.HostManifestSoftwareListParams{
			ListParams: input.Params(),
		})
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"list-host-munki-software",
				"host",
				err,
				"host_id", input.ID,
			)
		}
		return &hostMunkiSoftwareOutput{
			Body: api.Page[munkisoftware.HostManifestSoftware]{Items: rows, Count: count},
		}, nil
	})
}
