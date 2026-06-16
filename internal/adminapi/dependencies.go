package adminapi

import (
	"log/slog"

	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/webui"
)

// Dependencies is everything the HTTP server needs: its own runtime concerns
// plus the agent-protocol and admin-API route registrars supplied by the wiring
// layer. The wiring layer (package main) owns store construction and decides
// which capability registers against which group; adminapi just hosts them.
type Dependencies struct {
	Config         config.Config
	DB             *database.DB
	Version        string
	Logger         *slog.Logger
	WebHandler     *webui.Handler
	SessionManager *scs.SessionManager
	// AuthService gates every protected admin route. It is the one cross-cutting
	// service adminapi needs directly; capability auth lives in the registrars.
	AuthService *auth.Service

	Protocols []ProtocolRegistrar
	Admin     []AdminRegistrar
}

// AdminGroups are the Huma route groups distinguished by auth posture, plus the
// raw router for the few endpoints that bypass Huma. A registrar attaches its
// routes to the group matching its required posture.
type AdminGroups struct {
	Public    huma.API   // unauthenticated
	Protected huma.API   // authenticated user
	Ordinary  huma.API   // admin required for mutations, open reads
	Sensitive huma.API   // admin required for every operation
	Router    chi.Router // raw router: SSO redirect, storage streaming
}

// AdminRegistrar attaches one capability's admin routes to the shared groups.
// Registration must only define operations and bind handler closures; it must
// never dereference a dependency. That invariant lets BuildSchemaAPI emit the
// OpenAPI document by running the same registrars with nil dependencies and no
// database.
type AdminRegistrar func(AdminGroups)

// ProtocolRegistrar attaches one agent-facing protocol's routes to the router.
// The same no-dereference rule as AdminRegistrar applies.
type ProtocolRegistrar func(chi.Router)
