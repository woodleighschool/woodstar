package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/spf13/cobra"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/backgroundjobs"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/directory/entra"
	"github.com/woodleighschool/woodstar/internal/geoip"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/logging"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	mdpprotocol "github.com/woodleighschool/woodstar/internal/munki/mdp/protocol"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/postgres"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/webui"
	webdist "github.com/woodleighschool/woodstar/web"
)

const gracefulShutdownTimeout = 15 * time.Second

func main() {
	if err := rootCommand().ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCommand() *cobra.Command {
	var cfg config.Config

	root := &cobra.Command{
		Use:           "woodstar",
		Short:         "Woodstar macOS observability and admin server",
		Version:       buildinfo.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), cfg)
		},
	}

	root.AddCommand(userCommand())
	root.AddCommand(mdpCommand())
	root.AddCommand(openAPICommand())

	root.Flags().StringVar(&cfg.Host, "host", "", "Listen host")
	root.Flags().IntVar(&cfg.Port, "port", 0, "Listen port")
	root.Flags().StringVar(&cfg.ServerURL, "url", "", "Canonical HTTPS server origin")
	root.Flags().StringVar(&cfg.TLSCertFile, "tls-cert-file", "", "TLS certificate file")
	root.Flags().StringVar(&cfg.TLSKeyFile, "tls-key-file", "", "TLS private key file")
	root.Flags().StringVar(&cfg.DatabaseURL, "database-url", "", "Postgres connection URL")
	root.Flags().StringVar(&cfg.LogLevel, "log-level", "", "Log level")

	return root
}

func run(parent context.Context, cfg config.Config) error {
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
	var geoLookup func(netip.Addr) (*geoip.Result, error)
	if cfg.GeoIPEnabled() {
		geoReader, geoErr := geoip.Open(cfg.GeoIPCityFile, cfg.GeoIPASNFile)
		if geoErr != nil {
			logger.WarnContext(parent, "load GeoIP databases", "err", geoErr)
		} else {
			geoLookup = geoReader.Lookup
			defer func() {
				if err := geoReader.Close(); err != nil {
					logger.WarnContext(parent, "close GeoIP databases", "err", err)
				}
			}()
		}
	}

	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	sessions, sessionStore := newSessions(pool, cfg)

	// StopCleanup must run while the DB pool still exists.
	defer sessionStore.StopCleanup()

	storageBackend, err := storage.New(ctx, storageConfig(cfg))
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	deps, starters, err := buildDependencies(
		ctx,
		cfg,
		pool,
		sessions,
		logger,
		storageBackend,
		geoLookup,
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

	stopJobs, err := start(ctx, starters...)
	if err != nil {
		return fmt.Errorf("start background services: %w", err)
	}
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
	pool *pgxpool.Pool,
	sessions *scs.SessionManager,
	logger *slog.Logger,
	storageBackend storage.Backend,
	geoLookup func(netip.Addr) (*geoip.Result, error),
) (*api.Dependencies, []starter, error) {
	storageDelivery := storage.NewDelivery(storageBackend)

	// Core stores.
	labelStore := labels.NewStore(pool)
	directoryStore := directory.NewStore(pool)
	hostStore, heartbeatStore := newHostStores(pool)
	secretStore := agentauth.NewStore(pool)
	inventoryStore := inventory.NewStore(pool)
	primaryUsers := hosts.NewPrimaryUserStore(pool)

	// Osquery stores.
	reportStore := reports.NewStore(pool)
	checkStore := checks.NewStore(pool)
	liveQueries := livequery.NewStore(pool)

	// Munki stores.
	storageLogger := logger.With("component", "storage")
	objectStore := storage.NewObjectStore(pool, storageBackend, storageLogger)
	storageIngestor := storage.NewIngestor(objectStore, storageBackend)
	clientResourceStore := clientresources.NewStore(pool, objectStore)
	clientResourceService := clientresources.NewService(
		clientResourceStore,
		objectStore,
		storageIngestor,
		storageBackend,
	)
	packageStore := packages.NewStore(
		pool,
		objectStore,
	)
	munkiSoftwareStore := munkisoftware.NewStore(pool, objectStore, packageStore)
	munkiHostState := munki.NewStore(pool)

	// Santa stores.
	santaHostStore := santa.NewStore(pool)
	configurationStore := configurations.NewStore(pool)
	eventStore := events.NewStore(pool)
	ruleStore := rules.NewStore(pool)
	syncStore := syncstate.NewStore(pool)

	userService := directory.NewUserService(directoryStore)
	authService, err := newAuth(ctx, cfg, userService, sessions)
	if err != nil {
		return nil, nil, err
	}
	orbitAgent := orbit.NewEnrollmentService(hostStore, secretStore, primaryUsers, heartbeatStore)

	inventoryProjector := ingest.NewProjector(
		hostStore,
		inventoryStore,
		logger.With("component", "inventory"),
	)
	munkiIngestor := munki.NewDetailIngestor(munkiHostState)
	labelEvaluator := ingest.NewLabelEvaluator(labelStore, logger.With("component", "labels"))
	osqueryAgent := osquery.NewAgentService(osquery.Dependencies{
		HostStore:          hostStore,
		InventoryProjector: inventoryProjector,
		MunkiCollector:     munkiIngestor,
		LabelEvaluator:     labelEvaluator,
		ReportStore:        reportStore,
		CheckStore:         checkStore,
		LiveQueries:        liveQueries,
		SecretStore:        secretStore,
		Heartbeats:         heartbeatStore,
		Logger:             logger.With("component", "osquery"),
	})

	munkiRepository := munki.NewRepositoryService(munki.Dependencies{
		Software:        munkiSoftwareStore,
		Packages:        packageStore,
		Objects:         objectStore,
		ClientResources: clientResourceStore,
	})
	munkiDistributionLogger := logger.With("component", "munki_distribution")
	munkiDistribution := mdp.NewStore(pool, objectStore, munkiDistributionLogger)
	munkiDistributionProtocol, err := mdpprotocol.NewServer(
		ctx,
		munkiDistribution,
		storageDelivery,
		buildinfo.Version,
		munkiDistributionLogger,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("configure MDP protocol: %w", err)
	}
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
		Heartbeats:     heartbeatStore,
	})
	santaState := santa.NewHostStateService(santaHostStore, configurationStore)

	jobs, directorySync, err := newBackgroundJobs(
		cfg,
		pool,
		directoryStore,
		inventoryStore,
		eventStore,
		logger,
	)
	if err != nil {
		return nil, nil, err
	}

	deps := &api.Dependencies{
		Config:         cfg,
		Ready:          pool.Ping,
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
			AuthService:   authService,
			Users:         userService,
			Directory:     directoryStore,
			DirectorySync: directorySync,
			Hosts:         hostStore,
			PrimaryUser:   primaryUsers,
			Secrets:       secretStore,
			Software:      inventoryStore,
			Labels:        labelStore,
			GeoIP:         geoLookup,

			Reports:     reportStore,
			Checks:      checkStore,
			LiveQueries: liveQueries,

			StorageBackend:  storageBackend,
			StorageDelivery: storageDelivery,
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
			SantaState:          santaState,
		},
		Protocols: api.ProtocolDependencies{
			AgentAuth: secretStore,
			Orbit:     orbitAgent,
			Osquery:   osqueryAgent,
			Munki: api.MunkiProtocolDependencies{
				Hosts:                hostStore,
				Heartbeats:           heartbeatStore,
				Repository:           munkiRepository,
				Distribution:         munkiDistribution,
				DistributionProtocol: munkiDistributionProtocol,
				Delivery:             storageDelivery,
			},
			Santa: santaSync,
		},
	}
	starters := []starter{
		storageUploadCleanupStarter(storageIngestor, cfg.StorageTransferTTL, storageLogger),
		backgroundJobsStarter(jobs, logger.With("component", "background_jobs")),
	}

	return deps, starters, nil
}

func newBackgroundJobs(
	cfg config.Config,
	pool *pgxpool.Pool,
	directoryStore *directory.Store,
	inventoryStore *inventory.Store,
	eventStore *events.Store,
	logger *slog.Logger,
) (*backgroundjobs.Runtime, *entra.SyncJobs, error) {
	jobWorkers := river.NewWorkers()
	if err := river.AddWorkerSafely(
		jobWorkers,
		inventory.NewCleanupWorker(inventoryStore, logger.With("component", "inventory")),
	); err != nil {
		return nil, nil, fmt.Errorf("register inventory cleanup worker: %w", err)
	}
	if err := river.AddWorkerSafely(
		jobWorkers,
		events.NewCleanupWorker(eventStore, logger.With("component", "santa")),
	); err != nil {
		return nil, nil, fmt.Errorf("register Santa cleanup worker: %w", err)
	}
	periodicJobs := []*river.PeriodicJob{
		periodicJob(
			inventory.CleanupJobKind,
			inventory.CleanupJobInterval,
			func() river.JobArgs {
				return inventory.CleanupJobArgs{Trigger: backgroundjobs.TriggerScheduled}
			},
		),
		periodicJob(
			events.CleanupJobKind,
			cfg.SantaEventSweepInterval,
			func() river.JobArgs {
				return events.CleanupJobArgs{
					Trigger:       backgroundjobs.TriggerScheduled,
					RetentionDays: cfg.SantaEventRetentionDays,
				}
			},
		),
	}

	entraService, err := newEntraSyncService(cfg, directoryStore, logger)
	if err != nil {
		return nil, nil, err
	}
	if entraService != nil {
		if err := river.AddWorkerSafely(
			jobWorkers,
			entra.NewSyncWorker(
				entraService,
				postgres.NewSessionLocker(pool, entra.SyncAdvisoryLockID),
			),
		); err != nil {
			return nil, nil, fmt.Errorf("register Entra sync worker: %w", err)
		}
		periodicJobs = append(periodicJobs, periodicJob(
			entra.SyncJobKind,
			cfg.EntraSyncInterval,
			func() river.JobArgs {
				return entra.SyncJobArgs{Trigger: backgroundjobs.TriggerScheduled}
			},
		))
	}

	jobs, err := backgroundjobs.New(
		pool,
		jobWorkers,
		periodicJobs,
		logger.With("component", "background_jobs"),
	)
	if err != nil {
		return nil, nil, err
	}
	directorySync := entra.NewSyncJobs(entraService != nil, jobs)
	return jobs, directorySync, nil
}

func newHostStores(pool *pgxpool.Pool) (*hosts.Store, *heartbeats.Store) {
	return hosts.NewStore(pool), heartbeats.NewStore(pool)
}

func storageUploadCleanupStarter(
	ingestor *storage.Ingestor,
	transferTTL time.Duration,
	logger *slog.Logger,
) starter {
	return func(ctx context.Context) (func(), error) {
		cleanup := storage.StartUploadCleanup(ctx, ingestor, transferTTL, logger)
		return cleanup.Stop, nil
	}
}

func newAuth(
	ctx context.Context,
	cfg config.Config,
	users *directory.UserService,
	sessions *scs.SessionManager,
) (*auth.Service, error) {
	service, err := auth.NewService(users, sessions)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	if !cfg.OIDCEnabled() {
		return service, nil
	}

	err = service.ConfigureOIDC(ctx, auth.OIDCConfig{
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

func storageConfig(cfg config.Config) storage.Config {
	return storage.Config{
		Kind:        storage.Kind(cfg.StorageKind),
		TransferTTL: cfg.StorageTransferTTL,
		File: storage.FileConfig{
			Root:             cfg.StorageFileRoot,
			BaseURL:          cfg.ServerURL,
			CapabilityKeyHex: cfg.StorageCapabilityKey,
		},
		S3: storage.S3Config{
			Bucket:    cfg.StorageS3Bucket,
			Region:    cfg.StorageS3Region,
			Endpoint:  cfg.StorageS3Endpoint,
			AccessKey: cfg.StorageS3AccessKey,
			SecretKey: cfg.StorageS3SecretKey,
			PathStyle: cfg.StorageS3PathStyle,
		},
	}
}

func newEntraSyncService(
	cfg config.Config,
	directoryStore *directory.Store,
	logger *slog.Logger,
) (*entra.Service, error) {
	if !cfg.EntraEnabled() {
		return nil, nil
	}

	client, err := entra.NewClient(entra.Config{
		TenantID:         cfg.EntraTenantID,
		ClientID:         cfg.EntraClientID,
		ClientSecret:     cfg.EntraClientSecret,
		TransitiveGroups: cfg.EntraTransitiveGroups,
	})
	if err != nil {
		return nil, fmt.Errorf("configure Entra sync: %w", err)
	}

	return entra.NewService(
		directoryStore,
		client,
		logger.With("component", "entra"),
	), nil
}

// A nil starter means the capability is disabled by configuration.
type starter func(context.Context) (func(), error)

func start(ctx context.Context, starts ...starter) (func(), error) {
	var stops []func()

	for _, start := range starts {
		if start == nil {
			continue
		}

		stop, err := start(ctx)
		if err != nil {
			for _, stop := range slices.Backward(stops) {
				stop()
			}
			return nil, err
		}
		if stop != nil {
			stops = append(stops, stop)
		}
	}

	return func() {
		for _, stop := range slices.Backward(stops) {
			stop()
		}
	}, nil
}

func periodicJob(
	id string,
	interval time.Duration,
	args func() river.JobArgs,
) *river.PeriodicJob {
	return river.NewPeriodicJob(
		river.PeriodicInterval(interval),
		func() (river.JobArgs, *river.InsertOpts) { return args(), nil },
		&river.PeriodicJobOpts{ID: id, RunOnStart: true},
	)
}

func backgroundJobsStarter(jobs *backgroundjobs.Runtime, logger *slog.Logger) starter {
	return func(ctx context.Context) (func(), error) {
		if err := jobs.Start(ctx); err != nil {
			return nil, err
		}
		return func() {
			stopCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
			defer cancel()
			if err := jobs.Stop(stopCtx); err != nil {
				logger.WarnContext(stopCtx, "stop background jobs", "err", err)
			}
		}, nil
	}
}

func newSessions(pool *pgxpool.Pool, cfg config.Config) (*scs.SessionManager, *pgxstore.PostgresStore) {
	store := pgxstore.New(pool)

	sessions := scs.New()
	sessions.Store = store
	sessions.HashTokenInStore = true
	sessions.Lifetime = config.SessionLifetime
	sessions.Cookie.Name = "woodstar_session"
	sessions.Cookie.Path = "/"
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.Secure = cfg.SessionCookieSecure
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	sessions.Cookie.Persist = true

	return sessions, store
}
