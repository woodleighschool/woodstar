package adminapi

import (
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/adminapi/middleware"
)

// protocolRoutes mounts every agent-facing protocol endpoint. These are not
// admin API routes; they speak the wire protocol that each agent client
// hardcodes (orbit uses /api/fleet/orbit, osquery uses /api/v1/osquery and
// /api/osquery, Munki uses /munki, Santa uses /santa/sync).
func protocolRoutes(r chi.Router, deps Dependencies) {
	r.Use(middleware.RequestLogger(deps.Logger))
	for _, register := range deps.Protocols {
		register(r)
	}
}
