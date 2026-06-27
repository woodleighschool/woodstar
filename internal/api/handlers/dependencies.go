package handlers

import (
	"log/slog"

	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
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

// Dependencies is everything the HTTP server needs. Package main constructs
// stores and services; this package owns how those dependencies become Huma
// handlers and raw app routes.
type Dependencies struct {
	Config         config.Config
	DB             *database.DB
	Version        string
	Logger         *slog.Logger
	WebHandler     *webui.Handler
	SessionManager *scs.SessionManager

	AuthService *auth.Service
	Users       *directory.UserService
	Directory   *directory.Store
	Hosts       *hosts.Store
	PrimaryUser *hosts.PrimaryUserStore
	Secrets     *agentauth.Store
	Software    *inventory.Store
	Labels      *labels.Store

	Reports      *reports.Store
	Checks       *checks.Store
	LiveQueries  *livequery.Manager
	OsqueryAgent *osquery.AgentService

	OrbitAgent *orbit.EnrollmentService

	StorageBackend storage.Backend
	StorageKey     []byte
	StorageObjects *storage.ObjectStore

	MunkiPackages        *packages.Store
	MunkiSoftware        *munkisoftware.Store
	MunkiHostState       *munki.Store
	MunkiRepo            *munki.RepositoryService
	MunkiDistribution    *mdp.Store
	MunkiDistributionHub *mdp.Hub

	SantaConfigurations *configurations.Store
	SantaEvents         *events.Store
	SantaRules          *rules.Store
	SantaReferences     *references.Store
	SantaSync           *santa.SyncService
	SantaState          *santa.HostStateService
}

// Groups are the Huma route groups distinguished by auth posture, plus the raw
// router for app endpoints that bypass Huma response handling.
type Groups struct {
	Public    huma.API   // unauthenticated
	Session   huma.API   // public, but resolves a user when credentials are valid
	Protected huma.API   // authenticated user
	Ordinary  huma.API   // admin required for mutations, open reads
	Sensitive huma.API   // admin required for every operation
	Router    chi.Router // raw router: SSO redirect, storage streaming
}
