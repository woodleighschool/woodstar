package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
)

type munkiHostStateLoader interface {
	LoadHostState(context.Context, int64) (*munki.HostState, error)
}

func registerMunkiHostState(api huma.API, store munkiHostStateLoader, hostStore *hosts.Store) {
	registerHostState(
		api,
		"get-host-munki-state",
		"/api/hosts/{id}/munki",
		"Get Munki state for a host",
		"munki state not found",
		hostStore,
		store.LoadHostState,
	)
}
