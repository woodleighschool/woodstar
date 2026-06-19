package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"

	"github.com/woodleighschool/woodstar/internal/adminapi"
	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/directory/entra"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/logging"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/munki/mdp/worker"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkiprotocol "github.com/woodleighschool/woodstar/internal/munki/protocol"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/orbit"
	orbitprotocol "github.com/woodleighschool/woodstar/internal/orbit/protocol"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	osqueryprotocol "github.com/woodleighschool/woodstar/internal/osquery/protocol"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	santaprotocol "github.com/woodleighschool/woodstar/internal/santa/protocol"
	"github.com/woodleighschool/woodstar/internal/santa/references"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/webui"
	webdist "github.com/woodleighschool/woodstar/web"
)

const gracefulShutdownTimeout = 15 * time.Second

func main() {
	root := &cobra.Command{
		Use:           "woodstar",
		Short:         "Woodstar macOS observability and admin server",
		Version:       buildinfo.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(serveCommand())
	root.AddCommand(mdpCommand())
	root.AddCommand(openAPICommand())

	if err := root.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serveCommand() *cobra.Command {
	var cfg config.Config

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Woodstar server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return serve(cmd.Context(), cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.Host, "host", "", "HTTP listen host")
	cmd.Flags().IntVar(&cfg.Port, "port", 0, "HTTP listen port")
	cmd.Flags().StringVar(&cfg.PublicURL, "public-url", "", "Public base URL")
	cmd.Flags().StringVar(&cfg.DatabaseURL, "database-url", "", "Postgres connection URL")
	cmd.Flags().StringVar(&cfg.LogLevel, "log-level", "", "Log level")
	cmd.Flags().StringVar(&cfg.SessionSecret, "session-secret", "", "Session signing secret")

	return cmd
}

func mdpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "mdp",
		Short: "Run a Munki distribution point that mirrors and serves package installers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMDP(cmd.Context())
		},
	}
}

func runMDP(parent context.Context) error {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := worker.LoadConfig()
	if err != nil {
		return fmt.Errorf("load mdp config: %w", err)
	}
	logger := logging.NewLogger(os.Stderr, logging.ParseLevel(cfg.LogLevel))
	mdp, err := worker.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("init mdp worker: %w", err)
	}
	return mdp.Run(ctx)
}

func serve(parent context.Context, cfg config.Config) error {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := config.ApplyEnvironment(&cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := logging.NewLogger(os.Stderr, logging.ParseLevel(cfg.LogLevel))
	adminapi.InstallHumaErrorHandler(logger)

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	sessions, sessionStore := newSessions(db, cfg)

	// StopCleanup must run while the DB pool still exists.
	defer sessionStore.StopCleanup()

	storageCapabilityKey := assetCapabilityKey(cfg.SessionSecret)
	storageBackend, err := storage.New(ctx, storageConfig(cfg, storageCapabilityKey))
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	wiring := buildWiring(ctx, cfg, db, sessions, logger, storageBackend, storageCapabilityKey)
	server := adminapi.NewServer(wiring.serverDependencies())

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", server.Addr())
	if err != nil {
		return fmt.Errorf("listen %s: %w", server.Addr(), err)
	}

	stopJobs := start(ctx, wiring.starters()...)
	defer stopJobs()

	if err := runHTTPServer(ctx, server, listener); err != nil {
		return err
	}

	logger.InfoContext(context.Background(), "server stopped")
	return nil
}

func runHTTPServer(
	ctx context.Context,
	server *adminapi.Server,
	listener net.Listener,
) error {
	errc := make(chan error, 1)

	go func() {
		errc <- server.Serve(listener)
	}()

	select {
	case err := <-errc:
		if err != nil {
			return fmt.Errorf("serve http: %w", err)
		}
		return nil

	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http: %w", err)
		}

		if err := <-errc; err != nil {
			return fmt.Errorf("serve http: %w", err)
		}

		return nil
	}
}

// wiring holds every constructed store and service. It is the dependency glass:
// buildWiring fills it from config and the database, while its zero value drives
// OpenAPI schema generation, which registers routes without touching a store.
type wiring struct {
	cfg            config.Config
	logger         *slog.Logger
	db             *database.DB
	sessions       *scs.SessionManager
	storageBackend storage.Backend
	storageKey     []byte

	auth           *auth.Service
	users          *directory.UserService
	directory      *directory.Store
	hosts          *hosts.Store
	userAffinities *hosts.UserAffinityStore
	secrets        *agentauth.Store
	software       *inventory.Store
	labels         *labels.Store

	reports      *reports.Store
	checks       *checks.Store
	liveQueries  *livequery.Manager
	osqueryAgent *osquery.AgentService

	orbitAgent *orbit.EnrollmentService

	storageObjects       *storage.ObjectStore
	munkiPackages        *packages.Store
	munkiSoftware        *munkisoftware.Store
	munkiHostState       *munki.Store
	munkiRepo            *munki.RepositoryService
	munkiDistribution    *mdp.Store
	munkiDistributionHub *mdp.Hub

	configurations *configurations.Store
	events         *events.Store
	rules          *rules.Store
	references     *references.Store
	santaSync      *santa.SyncService
	santaState     *santa.HostStateService
}

func buildWiring(
	ctx context.Context,
	cfg config.Config,
	db *database.DB,
	sessions *scs.SessionManager,
	logger *slog.Logger,
	storageBackend storage.Backend,
	storageCapabilityKey []byte,
) *wiring {
	w := &wiring{
		cfg:            cfg,
		logger:         logger,
		db:             db,
		sessions:       sessions,
		storageBackend: storageBackend,
		storageKey:     slices.Clone(storageCapabilityKey),
	}

	// Core stores.
	w.directory = directory.NewStore(db)
	w.hosts = hosts.NewStore(db)
	w.userAffinities = hosts.NewUserAffinityStore(db)
	w.secrets = agentauth.NewStore(db)
	w.software = inventory.NewStore(db)
	w.labels = labels.NewStore(db)

	// Osquery stores.
	w.reports = reports.NewStore(db)
	w.checks = checks.NewStore(db)
	w.liveQueries = livequery.NewManager()

	// Munki stores.
	w.storageObjects = storage.NewObjectStore(db, storageBackend)
	w.munkiPackages = packages.NewStore(db, w.storageObjects)
	w.munkiSoftware = munkisoftware.NewStore(db, w.storageObjects, w.munkiPackages)
	w.munkiHostState = munki.NewStore(db)

	// Santa stores.
	santaHostStore := santa.NewStore(db)
	w.configurations = configurations.NewStore(db)
	w.events = events.NewStore(db)
	w.rules = rules.NewStore(db)
	w.references = references.NewStore(db)
	syncStore := syncstate.NewStore(db)

	w.users = directory.NewUserService(w.directory)
	w.auth = newAuth(ctx, cfg, w.users, sessions, logger)
	w.orbitAgent = orbit.NewEnrollmentService(w.hosts, w.secrets, w.userAffinities)

	inventoryProjector := ingest.NewProjector(w.hosts, w.software, logger.With("component", "inventory"))
	munkiIngestor := munki.NewDetailIngestor(w.munkiHostState)
	inventoryProjector.RegisterDetailHandler(catalog.IngestMunkiInfo, munkiIngestor.IngestInfo)
	inventoryProjector.RegisterDetailHandler(catalog.IngestMunkiInstalls, munkiIngestor.IngestInstalls)
	labelEvaluator := ingest.NewLabelEvaluator(w.labels, logger.With("component", "labels"))
	w.osqueryAgent = osquery.NewAgentService(osquery.Dependencies{
		HostStore:          w.hosts,
		InventoryProjector: inventoryProjector,
		LabelEvaluator:     labelEvaluator,
		ReportStore:        w.reports,
		CheckStore:         w.checks,
		LiveQueries:        w.liveQueries,
		SecretStore:        w.secrets,
		Logger:             logger.With("component", "osquery"),
	})

	w.munkiRepo = munki.NewRepositoryService(munki.Dependencies{
		Hosts:    w.hosts,
		Packages: w.munkiSoftware,
		Objects:  w.storageObjects,
	})
	munkiPresence := mdp.NewPresence()
	munkiDistributionLogger := logger.With("component", "munki-distribution")
	w.munkiDistribution = mdp.NewStore(db, munkiPresence, munkiDistributionLogger)
	w.munkiDistributionHub = mdp.NewHub(
		w.munkiDistribution,
		munkiPresence,
		munkiDistributionLogger,
	)

	w.santaSync = santa.NewSyncService(santa.Dependencies{
		HostStore:      santaHostStore,
		Configurations: w.configurations,
		UserAffinities: w.userAffinities,
		Events:         w.events,
		Rules:          w.rules,
		Sync:           syncStore,
	})
	w.santaState = santa.NewHostStateService(santaHostStore, w.configurations)

	return w
}

// serverDependencies projects the wiring into the HTTP server dependencies:
// runtime concerns plus the protocol and admin route registrars.
func (w *wiring) serverDependencies() adminapi.Dependencies {
	return adminapi.Dependencies{
		Config:         w.cfg,
		DB:             w.db,
		Version:        buildinfo.Version,
		Logger:         w.logger,
		SessionManager: w.sessions,
		AuthService:    w.auth,
		WebHandler: webui.NewHandler(webui.HandlerOptions{
			FS:        webdist.DistDirFS,
			Version:   buildinfo.Version,
			PublicURL: w.cfg.PublicURL,
			Logger:    w.logger.With("component", "web"),
		}),
		Protocols: w.protocolRegistrars(),
		Admin:     w.adminRegistrars(),
	}
}

// adminRegistrars returns one registrar per admin-API capability. The wiring
// layer owns the mapping of capability to auth-posture group; adminapi just
// runs the list. Registrars capture stores in closures and never dereference
// them, so the zero-value wiring drives schema generation safely.
func (w *wiring) adminRegistrars() []adminapi.AdminRegistrar {
	return []adminapi.AdminRegistrar{
		func(g adminapi.AdminGroups) { auth.RegisterPublicAdminRoutes(g.Public, w.auth) },
		func(g adminapi.AdminGroups) { auth.RegisterSSO(g.Router, w.auth) },
		func(g adminapi.AdminGroups) { auth.RegisterAccountAdminRoutes(g.Protected, w.auth, w.users) },
		func(g adminapi.AdminGroups) { directory.RegisterUserAdminRoutes(g.Ordinary, w.users) },
		func(g adminapi.AdminGroups) { directory.RegisterGroupAdminRoutes(g.Ordinary, w.directory) },
		func(g adminapi.AdminGroups) {
			adminapi.RegisterHostAdminRoutes(g.Ordinary, adminapi.HostRoutesOptions{
				Store:          w.hosts,
				UserAffinities: w.userAffinities,
				CheckStore:     w.checks,
				MunkiState:     w.munkiHostState,
				SantaState:     w.santaState,
			})
		},
		func(g adminapi.AdminGroups) { inventory.RegisterAdminRoutes(g.Ordinary, w.software) },
		func(g adminapi.AdminGroups) { inventory.RegisterHostAdminRoutes(g.Ordinary, w.software, w.hosts) },
		func(g adminapi.AdminGroups) { references.RegisterSoftwareAdminRoutes(g.Ordinary, w.references) },
		func(g adminapi.AdminGroups) { labels.RegisterAdminRoutes(g.Ordinary, w.labels) },
		func(g adminapi.AdminGroups) { agentauth.RegisterAdminRoutes(g.Sensitive, w.secrets) },
		func(g adminapi.AdminGroups) { reports.RegisterAdminRoutes(g.Ordinary, w.reports) },
		func(g adminapi.AdminGroups) { reports.RegisterHostAdminRoutes(g.Ordinary, w.reports, w.hosts) },
		func(g adminapi.AdminGroups) { checks.RegisterAdminRoutes(g.Ordinary, w.checks) },
		func(g adminapi.AdminGroups) { checks.RegisterHostAdminRoutes(g.Ordinary, w.checks, w.hosts) },
		func(g adminapi.AdminGroups) { livequery.RegisterAdminRoutes(g.Sensitive, w.liveQueries, w.hosts) },
		func(g adminapi.AdminGroups) { configurations.RegisterAdminRoutes(g.Ordinary, w.configurations) },
		func(g adminapi.AdminGroups) { rules.RegisterAdminRoutes(g.Ordinary, w.rules) },
		func(g adminapi.AdminGroups) { events.RegisterAdminRoutes(g.Ordinary, w.events) },
		func(g adminapi.AdminGroups) { rules.RegisterHostAdminRoutes(g.Ordinary, w.rules, w.hosts) },
		func(g adminapi.AdminGroups) {
			munkisoftware.RegisterAdminRoutes(
				g.Ordinary, w.munkiSoftware, w.munkiPackages, w.storageObjects, w.storageBackend,
				w.munkiDistributionHub,
			)
			munkisoftware.RegisterIconContentRoute(
				g.Router.With(adminapi.RequireHTTPAuth(w.auth)),
				w.munkiSoftware,
				w.storageObjects,
				w.storageBackend,
			)
		},
		func(g adminapi.AdminGroups) {
			packages.RegisterAdminRoutes(
				g.Ordinary, w.munkiPackages, w.storageObjects, w.storageBackend, w.munkiDistributionHub,
			)
		},
		func(g adminapi.AdminGroups) {
			mdp.RegisterAdminRoutes(g.Sensitive, w.munkiDistribution)
		},
	}
}

// protocolRegistrars returns one registrar per agent-facing protocol.
func (w *wiring) protocolRegistrars() []adminapi.ProtocolRegistrar {
	return []adminapi.ProtocolRegistrar{
		func(r chi.Router) {
			storage.RegisterBlobRoutes(r, w.storageBackend, w.storageKey)
		},
		func(r chi.Router) {
			orbitprotocol.RegisterOrbitRoutes(r, w.orbitAgent, w.logger.With("component", "orbit"))
		},
		func(r chi.Router) {
			osqueryprotocol.RegisterOsqueryRoutes(r, w.osqueryAgent, w.logger.With("component", "osquery"))
		},
		func(r chi.Router) {
			munkiprotocol.RegisterMunkiRoutes(
				r, w.secrets, w.munkiRepo, w.munkiDistribution, w.storageBackend, w.logger.With("component", "munki"),
			)
		},
		func(r chi.Router) {
			mdp.RegisterProtocolRoutes(
				r, w.munkiDistributionHub, w.munkiDistribution, w.storageBackend,
				w.logger.With("component", "munki-distribution"),
			)
		},
		func(r chi.Router) {
			santaprotocol.RegisterSantaRoutes(r, w.secrets, w.santaSync, w.logger.With("component", "santa"))
		},
	}
}

// starters returns the background lifecycle jobs the server runs alongside HTTP.
func (w *wiring) starters() []starter {
	return []starter{
		santaCleanupStarter(w.cfg, w.events, w.logger),
		entraSyncStarter(w.cfg, w.directory, w.labels, w.logger),
		munkiDistributionStarter(w.munkiDistributionHub),
	}
}

func newAuth(
	ctx context.Context,
	cfg config.Config,
	users *directory.UserService,
	sessions *scs.SessionManager,
	logger *slog.Logger,
) *auth.Service {
	service := auth.NewService(users, sessions)
	if !cfg.OIDCEnabled() {
		return service
	}

	err := service.ConfigureOIDC(ctx, auth.OIDCConfig{
		IssuerURL:    cfg.OIDCIssuerURL,
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.PublicURL + "/api/auth/sso/callback",
		Scopes:       cfg.OIDCScopes,
		EmailClaim:   cfg.OIDCEmailClaim,
	})
	if err != nil {
		logger.WarnContext(ctx, "sso disabled",
			"component", "auth",
			"operation", "oidc-discovery",
			"err", err,
		)
		return service
	}

	logger.InfoContext(ctx, "sso enabled",
		"component", "auth",
		"issuer", cfg.OIDCIssuerURL,
	)

	return service
}

func storageConfig(cfg config.Config, capabilityKey []byte) storage.Config {
	return storage.Config{
		Kind:          storage.Kind(cfg.StorageKind),
		FileRoot:      cfg.StorageFileRoot,
		PublicURL:     cfg.PublicURL,
		CapabilityKey: slices.Clone(capabilityKey),
		PresignTTL:    cfg.StorageS3PresignTTL,
		S3: storage.S3Config{
			Bucket:         cfg.StorageS3Bucket,
			Region:         cfg.StorageS3Region,
			Endpoint:       cfg.StorageS3Endpoint,
			PublicEndpoint: cfg.StorageS3PublicEndpoint,
			AccessKey:      cfg.StorageS3AccessKey,
			SecretKey:      cfg.StorageS3SecretKey,
			PathStyle:      cfg.StorageS3PathStyle,
			PresignTTL:     cfg.StorageS3PresignTTL,
		},
	}
}

func assetCapabilityKey(sessionSecret string) []byte {
	mac := hmac.New(sha256.New, []byte(sessionSecret))
	_, _ = mac.Write([]byte("woodstar-storage-capability-v1"))
	return mac.Sum(nil)
}

func santaCleanupStarter(
	cfg config.Config,
	store *events.Store,
	logger *slog.Logger,
) starter {
	return func(ctx context.Context) func() {
		cleanup := events.StartCleanup(
			ctx,
			store,
			cfg.SantaEventRetentionDays,
			cfg.SantaEventSweepInterval,
			logger.With("component", "santa"),
		)

		return cleanup.Stop
	}
}

func entraSyncStarter(
	cfg config.Config,
	directoryStore *directory.Store,
	labelStore *labels.Store,
	logger *slog.Logger,
) starter {
	if !cfg.EntraEnabled() {
		return nil
	}

	client := entra.NewClient(entra.Config{
		TenantID:         cfg.EntraTenantID,
		ClientID:         cfg.EntraClientID,
		ClientSecret:     cfg.EntraClientSecret,
		TransitiveGroups: cfg.EntraTransitiveGroups,
	})

	service := entra.NewService(
		directoryStore,
		client,
		logger.With("component", "entra"),
		labelStore,
	)

	return func(ctx context.Context) func() {
		return service.StartScheduler(ctx, cfg.EntraSyncInterval)
	}
}

func munkiDistributionStarter(hub *mdp.Hub) starter {
	return func(context.Context) func() { return hub.Close }
}

// A nil starter means the capability is disabled by configuration.
type starter func(context.Context) func()

func start(ctx context.Context, starts ...starter) func() {
	var stops []func()

	for _, start := range starts {
		if start == nil {
			continue
		}

		stop := start(ctx)
		if stop != nil {
			stops = append(stops, stop)
		}
	}

	return func() {
		for _, stop := range slices.Backward(stops) {
			stop()
		}
	}
}

func newSessions(db *database.DB, cfg config.Config) (*scs.SessionManager, *pgxstore.PostgresStore) {
	store := pgxstore.New(db.Pool())

	sessions := scs.New()
	sessions.Store = store
	sessions.Lifetime = config.SessionLifetime
	sessions.Cookie.Name = "woodstar_session"
	sessions.Cookie.Path = "/"
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.Secure = cfg.IsHTTPS()
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	sessions.Cookie.Persist = true

	return sessions, store
}
