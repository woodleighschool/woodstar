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
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
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

	cmd.Flags().StringVar(&cfg.Host, "host", "", "Listen host")
	cmd.Flags().IntVar(&cfg.Port, "port", 0, "Listen port")
	cmd.Flags().StringVar(&cfg.ServerURL, "url", "", "Canonical HTTPS server origin")
	cmd.Flags().StringVar(&cfg.TLSCertFile, "tls-cert-file", "", "TLS certificate file")
	cmd.Flags().StringVar(&cfg.TLSKeyFile, "tls-key-file", "", "TLS private key file")
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
	logLevel, err := logging.ParseLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("parse mdp log level: %w", err)
	}
	logger := logging.New(os.Stderr, logLevel)
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
		return fmt.Errorf("parse environment: %w", err)
	}
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	logLevel, err := logging.ParseLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("parse log level: %w", err)
	}
	logger := logging.New(os.Stderr, logLevel)

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

	deps, starters, err := buildDependencies(
		ctx,
		cfg,
		db,
		sessions,
		logger,
		storageBackend,
		storageCapabilityKey,
	)
	if err != nil {
		return fmt.Errorf("build services: %w", err)
	}
	server, err := api.NewServer(deps)
	if err != nil {
		return fmt.Errorf("build HTTP server: %w", err)
	}

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", server.Addr())
	if err != nil {
		return fmt.Errorf("listen %s: %w", server.Addr(), err)
	}

	stopJobs := start(ctx, starters...)
	defer stopJobs()

	return runServer(ctx, server, listener)
}

func runServer(
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
			return fmt.Errorf("serve: %w", err)
		}
		return nil

	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), gracefulShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		if err := <-errc; err != nil {
			return fmt.Errorf("serve: %w", err)
		}

		return nil
	}
}

func buildDependencies(
	ctx context.Context,
	cfg config.Config,
	db *database.DB,
	sessions *scs.SessionManager,
	logger *slog.Logger,
	storageBackend storage.Backend,
	storageCapabilityKey []byte,
) (*api.Dependencies, []starter, error) {
	storageDelivery := storage.NewDelivery(storageBackend)

	// Core stores.
	directoryStore := directory.NewStore(db)
	hostStore := hosts.NewStore(db)
	secretStore := agentauth.NewStore(db)
	inventoryStore := inventory.NewStore(db)
	labelStore := labels.NewStore(db)
	primaryUsers := hosts.NewPrimaryUserService(hosts.NewPrimaryUserStore(db), labelStore)

	// Osquery stores.
	reportStore := reports.NewStore(db)
	checkStore := checks.NewStore(db)
	liveQueries := livequery.NewManager()

	// Munki stores.
	objectStore := storage.NewObjectStore(db, storageBackend)
	storageIngestor := storage.NewIngestor(objectStore, storageBackend)
	clientResourceStore := clientresources.NewStore(db, objectStore)
	clientResourceService := clientresources.NewService(
		clientResourceStore,
		objectStore,
		storageIngestor,
		storageBackend,
	)
	packageStore := packages.NewStore(
		db,
		objectStore,
		logger.With("component", "munki_packages"),
	)
	munkiSoftwareStore := munkisoftware.NewStore(db, objectStore, packageStore)
	munkiHostState := munki.NewStore(db)

	// Santa stores.
	santaHostStore := santa.NewStore(db)
	configurationStore := configurations.NewStore(db)
	eventStore := events.NewStore(db)
	ruleStore := rules.NewStore(db)
	referenceStore := references.NewStore(db)
	syncStore := syncstate.NewStore(db)

	userService := directory.NewUserService(directoryStore, labelStore)
	authService, err := newAuth(ctx, cfg, userService, sessions)
	if err != nil {
		return nil, nil, err
	}
	orbitAgent := orbit.NewEnrollmentService(hostStore, secretStore, primaryUsers)

	inventoryProjector := ingest.NewProjector(
		hostStore,
		inventoryStore,
		logger.With("component", "inventory"),
	)
	munkiIngestor := munki.NewDetailIngestor(munkiHostState)
	inventoryProjector.RegisterDetailHandler(catalog.IngestMunkiInfo, munkiIngestor.IngestInfo)
	inventoryProjector.RegisterDetailHandler(catalog.IngestMunkiInstalls, munkiIngestor.IngestInstalls)
	labelEvaluator := ingest.NewLabelEvaluator(labelStore, logger.With("component", "labels"))
	osqueryAgent := osquery.NewAgentService(osquery.Dependencies{
		HostStore:          hostStore,
		InventoryProjector: inventoryProjector,
		LabelEvaluator:     labelEvaluator,
		ReportStore:        reportStore,
		CheckStore:         checkStore,
		LiveQueries:        liveQueries,
		SecretStore:        secretStore,
		Logger:             logger.With("component", "osquery"),
	})

	munkiRepository := munki.NewRepositoryService(munki.Dependencies{
		Hosts:           hostStore,
		Software:        munkiSoftwareStore,
		Packages:        packageStore,
		Objects:         objectStore,
		ClientResources: clientResourceStore,
	})
	munkiDistributionLogger := logger.With("component", "munki_distribution")
	munkiDistribution := mdp.NewStore(db, munkiDistributionLogger)
	munkiDistributionProtocol := mdpprotocol.NewServer(
		ctx,
		munkiDistribution,
		storageDelivery,
		munkiDistributionLogger,
	)
	munkiPackageService := munki.NewPackageService(munki.PackageServiceDependencies{
		Packages:               packageStore,
		DesiredPackagesChanged: munkiDistributionProtocol.RefreshDesiredPackages,
	})
	munkiSoftwareDeletions := munki.NewSoftwareDeletionService(
		munkiSoftwareStore,
		munkiDistributionProtocol.RefreshDesiredPackages,
	)

	santaSync := santa.NewSyncService(santa.Dependencies{
		HostStore:      santaHostStore,
		Configurations: configurationStore,
		Events:         eventStore,
		Rules:          ruleStore,
		Sync:           syncStore,
	})
	santaState := santa.NewHostStateService(santaHostStore, configurationStore)

	deps := &api.Dependencies{
		Config:         cfg,
		DB:             db,
		Version:        buildinfo.Version,
		Logger:         logger,
		SessionManager: sessions,
		WebHandler: webui.NewHandler(webui.HandlerOptions{
			FS:        webdist.DistDirFS,
			Version:   buildinfo.Version,
			ServerURL: cfg.ServerURL,
			Logger:    logger.With("component", "web"),
		}),
		App: api.AppDependencies{
			AuthService: authService,
			Users:       userService,
			Directory:   directoryStore,
			Hosts:       hostStore,
			PrimaryUser: primaryUsers,
			Secrets:     secretStore,
			Software:    inventoryStore,
			Labels:      labelStore,

			Reports:     reportStore,
			Checks:      checkStore,
			LiveQueries: liveQueries,

			StorageBackend:  storageBackend,
			StorageDelivery: storageDelivery,
			StorageKey:      slices.Clone(storageCapabilityKey),
			StorageObjects:  objectStore,
			StorageIngestor: storageIngestor,

			MunkiPackages:          munkiPackageService,
			MunkiClientResources:   clientResourceService,
			MunkiSoftware:          munkiSoftwareStore,
			MunkiSoftwareDeletions: munkiSoftwareDeletions,
			MunkiHostState:         munkiHostState,
			MunkiDistribution:      munkiDistribution,

			SantaConfigurations: configurationStore,
			SantaEvents:         eventStore,
			SantaRules:          ruleStore,
			SantaReferences:     referenceStore,
			SantaState:          santaState,
		},
		Protocols: api.ProtocolDependencies{
			AgentAuth: secretStore,
			Orbit:     orbitAgent,
			Osquery:   osqueryAgent,
			Munki: api.MunkiProtocolDependencies{
				Repository:           munkiRepository,
				Distribution:         munkiDistribution,
				DistributionProtocol: munkiDistributionProtocol,
				Delivery:             storageDelivery,
			},
			Santa: santaSync,
		},
	}
	starters := []starter{
		santaCleanupStarter(cfg, eventStore, logger),
		entraSyncStarter(cfg, directoryStore, labelStore, logger),
	}

	return deps, starters, nil
}

func newAuth(
	ctx context.Context,
	cfg config.Config,
	users *directory.UserService,
	sessions *scs.SessionManager,
) (*auth.Service, error) {
	service := auth.NewService(users, sessions)
	if !cfg.OIDCEnabled() {
		return service, nil
	}

	err := service.ConfigureOIDC(ctx, auth.OIDCConfig{
		IssuerURL:    cfg.OIDCIssuerURL,
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURL,
		Scopes:       cfg.OIDCScopes,
		EmailClaim:   cfg.OIDCEmailClaim,
	})
	if err != nil {
		return nil, fmt.Errorf("configure OIDC: %w", err)
	}
	return service, nil
}

func storageConfig(cfg config.Config, capabilityKey []byte) storage.Config {
	return storage.Config{
		Kind:          storage.Kind(cfg.StorageKind),
		FileRoot:      cfg.StorageFileRoot,
		BaseURL:       cfg.ServerURL,
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
		directory.NewProviderService(directoryStore, labelStore),
		client,
		logger.With("component", "entra"),
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
	sessions.Cookie.Secure = cfg.SessionCookieSecure
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	sessions.Cookie.Persist = true

	return sessions, store
}
