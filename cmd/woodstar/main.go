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
	"github.com/spf13/cobra"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/api"
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
	mdpprotocol "github.com/woodleighschool/woodstar/internal/munki/mdp/protocol"
	"github.com/woodleighschool/woodstar/internal/munki/mdp/worker"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
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
	logger := logging.New(os.Stderr, logging.ParseLevel(cfg.LogLevel))
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

	logger := logging.New(os.Stderr, logging.ParseLevel(cfg.LogLevel))

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
	server := api.NewServer(wiring.apiDependencies())

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", server.Addr())
	if err != nil {
		return fmt.Errorf("listen %s: %w", server.Addr(), err)
	}

	stopJobs := start(ctx, wiring.starters()...)
	defer stopJobs()

	return runHTTPServer(ctx, server, listener)
}

func runHTTPServer(
	ctx context.Context,
	server *api.Server,
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

	auth         *auth.Service
	users        *directory.UserService
	directory    *directory.Store
	hosts        *hosts.Store
	primaryUsers *hosts.PrimaryUserStore
	secrets      *agentauth.Store
	software     *inventory.Store
	labels       *labels.Store

	reports      *reports.Store
	checks       *checks.Store
	liveQueries  *livequery.Manager
	osqueryAgent *osquery.AgentService

	orbitAgent *orbit.EnrollmentService

	storageObjects    *storage.ObjectStore
	munkiPackages     *packages.Store
	munkiPackageSvc   *munki.PackageService
	munkiSoftware     *munkisoftware.Store
	munkiHostState    *munki.Store
	munkiRepo         *munki.RepositoryService
	munkiDistribution *mdp.Store
	munkiMDPProtocol  *mdpprotocol.Server

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
	w.primaryUsers = hosts.NewPrimaryUserStore(db)
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
	w.orbitAgent = orbit.NewEnrollmentService(w.hosts, w.secrets, w.primaryUsers)

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
		Software: w.munkiSoftware,
		Packages: w.munkiPackages,
		Objects:  w.storageObjects,
	})
	munkiDistributionLogger := logger.With("component", "munki_distribution")
	w.munkiDistribution = mdp.NewStore(db, munkiDistributionLogger)
	w.munkiMDPProtocol = mdpprotocol.NewServer(w.munkiDistribution, storageBackend, munkiDistributionLogger)
	w.munkiPackageSvc = munki.NewPackageService(munki.PackageServiceDependencies{
		Packages:               w.munkiPackages,
		DesiredPackagesChanged: w.munkiMDPProtocol.RefreshDesiredPackages,
	})

	w.santaSync = santa.NewSyncService(santa.Dependencies{
		HostStore:      santaHostStore,
		Configurations: w.configurations,
		Events:         w.events,
		Rules:          w.rules,
		Sync:           syncStore,
	})
	w.santaState = santa.NewHostStateService(santaHostStore, w.configurations)

	return w
}

// apiDependencies projects the constructed stores and services into the HTTP
// server. The API package owns route topology; main owns construction order.
func (w *wiring) apiDependencies() *api.Dependencies {
	return &api.Dependencies{
		Config:         w.cfg,
		DB:             w.db,
		Version:        buildinfo.Version,
		Logger:         w.logger,
		SessionManager: w.sessions,
		WebHandler: webui.NewHandler(webui.HandlerOptions{
			FS:        webdist.DistDirFS,
			Version:   buildinfo.Version,
			PublicURL: w.cfg.PublicURL,
			Logger:    w.logger.With("component", "web"),
		}),
		App: api.AppDependencies{
			AuthService: w.auth,
			Users:       w.users,
			Directory:   w.directory,
			Hosts:       w.hosts,
			PrimaryUser: w.primaryUsers,
			Secrets:     w.secrets,
			Software:    w.software,
			Labels:      w.labels,

			Reports:     w.reports,
			Checks:      w.checks,
			LiveQueries: w.liveQueries,

			StorageBackend: w.storageBackend,
			StorageKey:     slices.Clone(w.storageKey),
			StorageObjects: w.storageObjects,

			MunkiPackages:     w.munkiPackageSvc,
			MunkiSoftware:     w.munkiSoftware,
			MunkiHostState:    w.munkiHostState,
			MunkiDistribution: w.munkiDistribution,

			SantaConfigurations: w.configurations,
			SantaEvents:         w.events,
			SantaRules:          w.rules,
			SantaReferences:     w.references,
			SantaState:          w.santaState,
		},
		Protocols: api.ProtocolDependencies{
			AgentAuth: w.secrets,
			Orbit:     w.orbitAgent,
			Osquery:   w.osqueryAgent,
			Munki: api.MunkiProtocolDependencies{
				Repository:           w.munkiRepo,
				Distribution:         w.munkiDistribution,
				DistributionProtocol: w.munkiMDPProtocol,
				Storage:              w.storageBackend,
			},
			Santa: w.santaSync,
		},
	}
}

// starters returns the background lifecycle jobs the server runs alongside HTTP.
func (w *wiring) starters() []starter {
	return []starter{
		santaCleanupStarter(w.cfg, w.events, w.logger),
		entraSyncStarter(w.cfg, w.directory, w.labels, w.logger),
	}
}

func newAuth(
	ctx context.Context,
	cfg config.Config,
	users *directory.UserService,
	sessions *scs.SessionManager,
	logger *slog.Logger,
) *auth.Service {
	logger = logger.With("component", "auth")
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
			"operation", "oidc_discovery",
			"err", err,
		)
		return service
	}

	logger.InfoContext(ctx, "sso enabled",
		"operation", "oidc_discovery",
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
