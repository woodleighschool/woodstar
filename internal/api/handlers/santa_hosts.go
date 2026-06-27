package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa"
)

type santaHostStateLoader interface {
	LoadHostState(context.Context, int64) (*santa.HostState, error)
}

func registerSantaHostState(api huma.API, store santaHostStateLoader, hostStore *hosts.Store) {
	registerHostState(
		api,
		"get-host-santa-state",
		"/api/hosts/{id}/santa",
		"Get Santa state for a host",
		"santa state not found",
		hostStore,
		store.LoadHostState,
	)
}
