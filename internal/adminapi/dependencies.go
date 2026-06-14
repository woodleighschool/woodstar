package adminapi

import (
	"log/slog"

	"github.com/alexedwards/scs/v2"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
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
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/webui"
)

// Dependencies is the set of runtime services and resource stores the HTTP
// server needs. Capability fields are grouped by product ownership, while
// Runtime/Auth hold cross-cutting server concerns.
type Dependencies struct {
	Runtime   RuntimeDependencies
	Auth      AuthDependencies
	Directory DirectoryDependencies
	Inventory InventoryDependencies
	AgentAuth AgentAuthDependencies
	Orbit     OrbitDependencies
	Osquery   OsqueryDependencies
	Munki     MunkiDependencies
	Santa     SantaDependencies
}

type RuntimeDependencies struct {
	Config         config.Config
	DB             *database.DB
	Version        string
	Logger         *slog.Logger
	WebHandler     *webui.Handler
	SessionManager *scs.SessionManager
}

type AuthDependencies struct {
	AuthService *auth.Service
	UserService *directory.UserService
}

type DirectoryDependencies struct {
	Store *directory.Store
}

type InventoryDependencies struct {
	Hosts          *hosts.Store
	UserAffinities *hosts.UserAffinityStore
	Software       *inventory.Store
	Labels         *labels.Store
}

type AgentAuthDependencies struct {
	Store *agentauth.Store
}

type OrbitDependencies struct {
	Agent *orbit.EnrollmentService
}

type OsqueryDependencies struct {
	Agent       *osquery.AgentService
	LiveQueries *livequery.Manager
	Reports     *reports.Store
	Checks      *checks.Store
}

type MunkiDependencies struct {
	Repository *munki.RepositoryService
	Store      storage.Store
	HostState  *munki.Store
	Packages   *packages.Store
	Software   *munkisoftware.Store
}

type SantaDependencies struct {
	Sync           *santa.SyncService
	HostState      *santa.HostStateService
	Configurations *configurations.Store
	Rules          *rules.Store
	Events         *events.Store
	References     *references.Store
}
