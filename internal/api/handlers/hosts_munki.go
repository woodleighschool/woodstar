package handlers

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki"
)

type hostMunkiStateLoader interface {
	LoadHostState(context.Context, int64) (*munki.HostState, error)
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
		"munki state not found",
		store.LoadHostState,
		logger,
	)
}
