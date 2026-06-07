package adminapi

import (
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/adminapi/middleware"
	munkiprotocol "github.com/woodleighschool/woodstar/internal/munki/protocol"
	orbitprotocol "github.com/woodleighschool/woodstar/internal/orbit/protocol"
	osqueryprotocol "github.com/woodleighschool/woodstar/internal/osquery/protocol"
	santaprotocol "github.com/woodleighschool/woodstar/internal/santa/protocol"
)

// protocolRoutes mounts every agent-facing protocol endpoint. These are not
// admin API routes; they speak the wire protocol that each agent client
// hardcodes (orbit uses /api/fleet/orbit, osquery uses /api/v1/osquery and
// /api/osquery, Munki uses /munki, Santa uses /santa/sync).
func protocolRoutes(r chi.Router, deps Dependencies) {
	r.Use(middleware.RequestLogger(deps.Runtime.Logger))
	orbitprotocol.RegisterOrbitRoutes(r, deps.Orbit.Agent, deps.Runtime.Logger.With("component", "orbit"))
	osqueryprotocol.RegisterOsqueryRoutes(
		r,
		deps.Osquery.Agent,
		deps.Runtime.Logger.With("component", "osquery"),
	)
	munkiprotocol.RegisterMunkiRoutes(
		r,
		deps.AgentAuth.Store,
		deps.Munki.Repository,
		deps.Runtime.Logger.With("component", "munki"),
	)
	santaprotocol.RegisterSantaRoutes(
		r,
		deps.AgentAuth.Store,
		deps.Santa.Sync,
		deps.Runtime.Logger.With("component", "santa"),
	)
}
