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
	"github.com/woodleighschool/goodies/auth/authn"
	"github.com/woodleighschool/goodies/auth/authz"
	"github.com/woodleighschool/goodies/bloby"
	blobydb "github.com/woodleighschool/goodies/bloby/pgxstore"
	"github.com/woodleighschool/goodies/pglock"

	"github.com/woodleighschool/woodstar/internal/account"
	"github.com/woodleighschool/woodstar/internal/activity"
	activityapi "github.com/woodleighschool/woodstar/internal/activity/httpapi"
	"github.com/woodleighschool/woodstar/internal/agentauth"
	agentauthapi "github.com/woodleighschool/woodstar/internal/agentauth/httpapi"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/backgroundjobs"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/directory/entra"
	directoryapi "github.com/woodleighschool/woodstar/internal/directory/httpapi"
	"github.com/woodleighschool/woodstar/internal/geoip"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	hostsapi "github.com/woodleighschool/woodstar/internal/hosts/httpapi"
	"github.com/woodleighschool/woodstar/internal/inventory"
	inventoryapi "github.com/woodleighschool/woodstar/internal/inventory/httpapi"
	"github.com/woodleighschool/woodstar/internal/labels"
	labelsapi "github.com/woodleighschool/woodstar/internal/labels/httpapi"
	"github.com/woodleighschool/woodstar/internal/logging"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	munkiapi "github.com/woodleighschool/woodstar/internal/munki/httpapi"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	mdpprotocol "github.com/woodleighschool/woodstar/internal/munki/mdp/protocol"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkiprotocol "github.com/woodleighschool/woodstar/internal/munki/protocol"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/orbit"
	orbitprotocol "github.com/woodleighschool/woodstar/internal/orbit/protocol"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/osquery/history"
	osqueryapi "github.com/woodleighschool/woodstar/internal/osquery/httpapi"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
	osqueryprotocol "github.com/woodleighschool/woodstar/internal/osquery/protocol"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/postgres"
	"github.com/woodleighschool/woodstar/internal/rbac"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	santaapi "github.com/woodleighschool/woodstar/internal/santa/httpapi"
	santaprotocol "github.com/woodleighschool/woodstar/internal/santa/protocol"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
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
	root := &cobra.Command{
		Use:           "woodstar",
		Short:         "macOS observability and admin server",
		Version:       buildinfo.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context())
		},
	}

	root.AddCommand(userCommand())
	root.AddCommand(mdpCommand())
	root.AddCommand(openAPICommand())

	return root
}

func run(parent context.Context) error {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logLevel, err := logging.ParseLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("parse log level: %w", err)
	}
	logger := logging.New(os.Stdout, logLevel)
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

	sessions, sessionStore := newSessions(pool, cfg, logger)

	// StopCleanup must run while the DB pool still exists.
	defer sessionStore.StopCleanup()

	storageLogger := logger.With("component", "storage")
	storage, err := bloby.New(ctx, blobydb.New(pool), storageConfig(cfg), storageLogger)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	app, err := buildApplication(
		ctx,
		cfg,
		pool,
		sessions,
		logger,
		storage,
		geoLookup,
	)
	if err != nil {
		return fmt.Errorf("build services: %w", err)
	}
	defer app.close()

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", app.server.Addr())
	if err != nil {
		return fmt.Errorf("listen %s: %w", app.server.Addr(), err)
	}

	stopJobs, err := start(ctx, app.starters...)
	if err != nil {
		return fmt.Errorf("start background services: %w", err)
	}
	defer stopJobs()

	return runServer(ctx, app, listener)
}

func runServer(
	ctx context.Context,
	app *application,
	listener net.Listener,
) error {
	errc := make(chan error, 1)

	go func() {
		errc <- app.server.Serve(listener)
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

		if err := app.shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		if err := <-errc; err != nil {
			return fmt.Errorf("serve: %w", err)
		}

		return nil
	}
}

type application struct {
	server                    *api.Server
	starters                  []starter
	munkiDistributionProtocol *mdpprotocol.Server
}

func (app *application) close() {
	app.munkiDistributionProtocol.Close()
}

func (app *application) shutdown(ctx context.Context) error {
	app.close()
	return app.server.Shutdown(ctx)
}

func buildApplication(
	ctx context.Context,
	cfg config.Config,
	pool *pgxpool.Pool,
	sessions *scs.SessionManager,
	logger *slog.Logger,
	storage *bloby.Service,
	geoLookup func(netip.Addr) (*geoip.Result, error),
) (*application, error) {
	// Core stores.
	labelStore := labels.NewStore(pool)
	directoryStore := directory.NewStore(pool, labelStore)
	hostStore := hosts.NewStore(pool, labelStore)
	heartbeatStore := heartbeats.NewStore(pool)
	secretStore := agentauth.NewStore(pool)
	inventoryStore := inventory.NewStore(pool)
	primaryUsers := hosts.NewPrimaryUserStore(pool, labelStore)
	activityStore := activity.NewStore(pool)

	// Osquery stores.
	reportStore := reports.NewStore(pool)
	policyStore := policies.NewStore(pool)
	liveQueries := livequery.NewStore(pool)
	historyStore := history.NewStore(pool)

	// Munki stores.
	clientResourceStore := clientresources.NewStore(pool, storage)
	clientResourceService := clientresources.NewService(clientResourceStore, storage)
	packageStore := packages.NewStore(pool, storage)
	munkiSoftwareStore := munkisoftware.NewStore(pool, storage, packageStore)
	munkiHostState := munki.NewStore(pool)

	// Santa stores.
	santaHostStore := santa.NewStore(pool)
	configurationStore := configurations.NewStore(pool)
	eventStore := events.NewStore(pool)
	ruleStore := rules.NewStore(pool)
	syncStore := syncstate.NewStore(pool)

	userService := directory.NewUserService(directoryStore)
	authzService, err := authz.NewService(rbac.NewStore(pool), rbac.Resources())
	if err != nil {
		return nil, fmt.Errorf("create authorization service: %w", err)
	}
	authnService, err := newAuth(ctx, cfg, directoryStore, sessions, authzService, logger.With("component", "auth"))
	if err != nil {
		return nil, err
	}
	orbitAgent := orbit.NewEnrollmentService(orbit.Dependencies{
		Hosts:                  hostStore,
		Secrets:                secretStore,
		PrimaryUsers:           primaryUsers,
		Heartbeats:             heartbeatStore,
		Remediations:           policyStore,
		ScriptExecutionTimeout: cfg.OrbitScriptTimeout,
		Activity:               activityStore,
		Logger:                 logger.With("component", "orbit"),
	})

	osqueryAgent := osquery.NewAgentService(osquery.Dependencies{
		HostStore:          hostStore,
		InventoryProjector: ingest.NewProjector(hostStore, inventoryStore, logger.With("component", "inventory")),
		MunkiCollector:     munki.NewDetailIngestor(munkiHostState),
		LabelEvaluator:     ingest.NewLabelEvaluator(labelStore, logger.With("component", "labels")),
		ReportStore:        reportStore,
		PolicyStore:        policyStore,
		LiveQueries:        liveQueries,
		SecretStore:        secretStore,
		Heartbeats:         heartbeatStore,
		Activity:           activityStore,
		Logger:             logger.With("component", "osquery"),
	})

	munkiRepository := munki.NewRepositoryService(munki.Dependencies{
		Software:        munkiSoftwareStore,
		Packages:        packageStore,
		Objects:         storage,
		ClientResources: clientResourceStore,
	})
	munkiDistributionLogger := logger.With("component", "munki_distribution")
	munkiDistribution := mdp.NewStore(pool, storage, munkiDistributionLogger)
	munkiDistributionProtocol, err := mdpprotocol.NewServer(
		ctx,
		munkiDistribution,
		storage,
		buildinfo.Version,
		munkiDistributionLogger,
	)
	if err != nil {
		return nil, fmt.Errorf("configure MDP protocol: %w", err)
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
		activityStore,
		historyStore,
		logger,
	)
	if err != nil {
		munkiDistributionProtocol.Close()
		return nil, err
	}

	apiLogger := logger.With("component", "api")
	server := api.NewServer(api.ServerOptions{
		Config:         cfg,
		Ready:          pool.Ping,
		Version:        buildinfo.Version,
		Logger:         logger,
		SessionManager: sessions,
		Authn:          authnService,
		TransferOrigin: storage.TransferOrigin(),
		WebHandler: webui.NewHandler(webui.HandlerOptions{
			FS:        webdist.DistDirFS,
			Version:   buildinfo.Version,
			ServerURL: cfg.ServerURL,
			Logger:    logger.With("component", "web"),
		}),
		RegisterRoutes: func(routes api.Routes) {
			routes.StorageTransfers.Handle("/storage/*", storage.TransferHandler())

			activityapi.RegisterAPI(routes.App, activityStore, authzService, apiLogger)
			account.RegisterAPI(routes.App, account.Dependencies{Users: userService, Authn: authnService, Authz: authzService, Logger: apiLogger})
			directoryapi.RegisterAPI(
				routes.App,
				userService,
				directoryStore,
				directorySync,
				authzService,
				apiLogger,
			)
			hostsapi.RegisterAPI(
				routes.App,
				hostStore,
				primaryUsers,
				munkiHostState,
				santaState,
				munkiDistribution,
				geoLookup,
				activityStore,
				authzService,
				apiLogger,
			)
			inventoryapi.RegisterAPI(routes.App, inventoryStore, authzService, apiLogger)
			labelsapi.RegisterAPI(routes.App, labelStore, authzService, apiLogger)
			agentauthapi.RegisterAPI(routes.App, secretStore, authzService, apiLogger)
			osqueryapi.RegisterAPI(routes.App, osqueryapi.Dependencies{
				Reports:     reportStore,
				Policies:    policyStore,
				LiveQueries: liveQueries,
				Hosts:       hostStore,
				History:     historyStore,
				Activity:    activityStore,
				Authorizer:  authzService,
				Logger:      apiLogger,
			})
			munkiapi.RegisterAPI(routes.App, munkiapi.Dependencies{
				Authenticator:   authnService,
				Authorizer:      authzService,
				HostState:       munkiHostState,
				Software:        munkiSoftwareStore,
				DeleteSoftware:  munkiSoftwareDeletions,
				Packages:        munkiPackageService,
				ClientResources: clientResourceService,
				Objects:         storage,
				Distribution:    munkiDistribution,
				Connections:     munkiDistributionProtocol,
				Logger:          apiLogger,
			})
			santaapi.RegisterAPI(
				routes.App,
				santaState,
				configurationStore,
				ruleStore,
				eventStore,
				authzService,
				apiLogger,
			)

			orbitprotocol.NewServer(
				orbitAgent,
				logger.With("component", "orbit"),
			).RegisterRoutes(routes.Protocols.Ordinary)
			osqueryprotocol.NewServer(
				osqueryAgent,
				logger.With("component", "osquery"),
			).RegisterRoutes(routes.Protocols.Ordinary)
			munkiprotocol.NewServer(
				secretStore,
				hostStore,
				munkiRepository,
				heartbeatStore,
				munkiDistribution,
				storage,
				logger.With("component", "munki"),
			).RegisterRoutes(routes.Protocols.Ordinary, routes.Protocols.Transfers)
			munkiDistributionProtocol.RegisterRoutes(
				routes.Protocols.Ordinary,
				routes.Protocols.WebSockets,
			)
			santaprotocol.NewServer(
				secretStore,
				santaSync,
				logger.With("component", "santa"),
			).RegisterRoutes(routes.Protocols.Ordinary)
		},
	})
	starters := []starter{
		storageUploadCleanupStarter(storage),
		backgroundJobsStarter(jobs, logger.With("component", "background_jobs")),
	}

	return &application{
		server:                    server,
		starters:                  starters,
		munkiDistributionProtocol: munkiDistributionProtocol,
	}, nil
}

func newBackgroundJobs(
	cfg config.Config,
	pool *pgxpool.Pool,
	directoryStore *directory.Store,
	inventoryStore *inventory.Store,
	eventStore *events.Store,
	activityStore *activity.Store,
	historyStore *history.Store,
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
	if err := river.AddWorkerSafely(
		jobWorkers,
		activity.NewCleanupWorker(activityStore, logger.With("component", "activity")),
	); err != nil {
		return nil, nil, fmt.Errorf("register activity cleanup worker: %w", err)
	}
	if err := river.AddWorkerSafely(jobWorkers, history.NewSnapshotWorker(historyStore)); err != nil {
		return nil, nil, fmt.Errorf("register osquery history snapshot worker: %w", err)
	}
	if err := river.AddWorkerSafely(
		jobWorkers,
		history.NewCleanupWorker(historyStore, logger.With("component", "osquery")),
	); err != nil {
		return nil, nil, fmt.Errorf("register osquery history cleanup worker: %w", err)
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
		periodicJob(
			activity.CleanupJobKind,
			activity.CleanupJobInterval,
			func() river.JobArgs {
				return activity.CleanupJobArgs{
					Trigger:       backgroundjobs.TriggerScheduled,
					RetentionDays: cfg.ActivityRetentionDays,
				}
			},
		),
		periodicJob(
			history.SnapshotJobKind,
			history.SnapshotJobInterval,
			func() river.JobArgs {
				return history.SnapshotJobArgs{Trigger: backgroundjobs.TriggerScheduled}
			},
		),
		periodicJob(
			history.CleanupJobKind,
			history.CleanupJobInterval,
			func() river.JobArgs {
				return history.CleanupJobArgs{
					Trigger:       backgroundjobs.TriggerScheduled,
					RetentionDays: cfg.OsqueryHistoryRetentionDays,
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
				pglock.New(pool, entra.SyncAdvisoryLockID),
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

func storageUploadCleanupStarter(storage *bloby.Service) starter {
	return func(ctx context.Context) (func(), error) {
		cleanupCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			defer close(done)
			storage.RunCleanup(cleanupCtx)
		}()
		return func() {
			cancel()
			<-done
		}, nil
	}
}

func newAuth(
	ctx context.Context,
	cfg config.Config,
	store *directory.Store,
	sessions *scs.SessionManager,
	authorization *authz.Service,
	logger *slog.Logger,
) (*authn.Service, error) {
	authConfig := authn.Config{
		Admit: authorization.HasAccess, SuccessRedirect: "/hosts", FailureRedirect: "/login", Logger: logger,
	}
	if cfg.OIDCEnabled() {
		authConfig.OIDC = &authn.OIDCConfig{
			IssuerURL:    cfg.OIDCIssuerURL,
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			RedirectURL:  cfg.OIDCRedirectURL,
			Scopes:       cfg.OIDCScopes,
			EmailClaim:   cfg.OIDCEmailClaim,
		}
	}
	service, err := authn.New(ctx, directory.NewAuthnStore(store), sessions, authConfig)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	return service, nil
}

func storageConfig(cfg config.Config) bloby.Config {
	return bloby.Config{
		Kind:        bloby.Kind(cfg.StorageKind),
		TransferTTL: cfg.StorageTransferTTL,
		File: bloby.FileConfig{
			Root:             cfg.StorageFileRoot,
			BaseURL:          cfg.ServerURL,
			CapabilityKeyHex: cfg.StorageCapabilityKey,
		},
		S3: bloby.S3Config{
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

func newSessions(pool *pgxpool.Pool, cfg config.Config, logger *slog.Logger) (*scs.SessionManager, *pgxstore.PostgresStore) {
	store := pgxstore.New(pool)

	sessions := scs.New()
	sessions.ErrorFunc = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.ErrorContext(r.Context(), "session persistence failed", "operation", "session", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
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
