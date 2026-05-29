package api

import (
	"log/slog"

	"github.com/alexedwards/scs/v2"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	"github.com/woodleighschool/woodstar/internal/santa/references"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/software"
	"github.com/woodleighschool/woodstar/internal/users"
	"github.com/woodleighschool/woodstar/internal/web"
)

// Dependencies is the set of runtime services and resource stores the HTTP
// server needs. Capability fields are grouped by product ownership, while
// Runtime/Auth hold cross-cutting server concerns.
type Dependencies struct {
	Runtime   RuntimeDependencies
	Auth      AuthDependencies
	Inventory InventoryDependencies
	Directory DirectoryDependencies
	AgentAuth AgentAuthDependencies
	Orbit     OrbitDependencies
	Osquery   OsqueryDependencies
	Santa     SantaDependencies
}

type RuntimeDependencies struct {
	Config         config.Config
	DB             *database.DB
	Version        string
	Logger         *slog.Logger
	WebHandler     *web.Handler
	SessionManager *scs.SessionManager
}

type AuthDependencies struct {
	AuthService *auth.Service
	UserService *users.Service
}

type InventoryDependencies struct {
	Hosts          *hosts.Store
	DeviceMappings *hosts.DeviceMappingStore
	Software       *software.Store
	Labels         *labels.Store
}

type DirectoryDependencies struct {
	Store *directory.Store
}

type AgentAuthDependencies struct {
	Store *agentauth.Store
}

type OrbitDependencies struct {
	Agent *orbit.Service
}

type OsqueryDependencies struct {
	Agent       *osquery.Service
	LiveQueries *livequery.Manager
	Reports     *reports.Store
	Checks      *checks.Store
}

type SantaDependencies struct {
	Sync           *santa.Service
	HostState      *santa.HostStateService
	Configurations *configurations.Store
	Rules          *rules.Store
	Events         *events.Store
	References     *references.Store
}
