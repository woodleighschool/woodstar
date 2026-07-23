package handlers

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

type hostMunkiStateLoader interface {
	LoadHostState(ctx context.Context, hostID int64) (*munki.HostState, error)
}

func registerHostMunkiState(
	api huma.API,
	store hostMunkiStateLoader,
	logger *slog.Logger,
) {
	registerHostState(
		api,
		"get-host-munki-state",
		"/api/hosts/{id}/munki",
		"Get Munki state for a host",
		store.LoadHostState,
		logger,
	)
}

func registerHostMunkiSoftware(
	api huma.API,
	store *munkisoftware.Store,
	logger *slog.Logger,
) {
	registerHostPage(
		api,
		"list-host-munki-software",
		"/api/hosts/{id}/munki/software",
		"List Munki software for a host",
		func(
			ctx context.Context,
			hostID int64,
			params dbutil.ListParams,
		) ([]munkisoftware.HostManifestSoftware, int, error) {
			return store.ListForHost(ctx, hostID, munkisoftware.HostManifestSoftwareListParams{
				ListParams: params,
			})
		},
		logger,
	)
}
