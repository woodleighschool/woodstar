package main

import (
	"context"
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
	admin "github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/directory/entra"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/logging"
	"github.com/woodleighschool/woodstar/internal/munki"
	munkistorage "github.com/woodleighschool/woodstar/internal/munki/storage"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
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
	"github.com/woodleighschool/woodstar/internal/software"
	"github.com/woodleighschool/woodstar/internal/web"
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

func serve(parent context.Context, cfg config.Config) error {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := config.ApplyEnvironment(&cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := logging.NewLogger(os.Stderr, logging.ParseLevel(cfg.LogLevel))
	admin.InstallHumaErrorHandler(logger)

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	sessions, sessionStore := newSessions(db, cfg)

	// StopCleanup must run while the DB pool still exists.
	defer sessionStore.StopCleanup()

	server, starts := newServer(ctx, cfg, db, sessions, logger)

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", server.Addr())
	if err != nil {
		return fmt.Errorf("listen %s: %w", server.Addr(), err)
	}

	stopJobs := start(ctx, starts...)
	defer stopJobs()

	if err := runHTTPServer(ctx, server, listener); err != nil {
		return err
	}

	logger.InfoContext(context.Background(), "server stopped")
	return nil
}

func runHTTPServer(
	ctx context.Context,
	server *admin.Server,
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

func newServer(
	ctx context.Context,
	cfg config.Config,
	db *database.DB,
	sessions *scs.SessionManager,
	logger *slog.Logger,
) (*admin.Server, []starter) {
	// Core stores.
	directoryStore := directory.NewStore(db)
	hostStore := hosts.NewStore(db)
	userAffinities := hosts.NewUserAffinityStore(db)
	secretStore := agentauth.NewStore(db)
	softwareStore := software.NewStore(db)
	labelStore := labels.NewStore(db)

	// Osquery stores.
	reportStore := reports.NewStore(db)
	checkStore := checks.NewStore(db)

	// Munki stores.
	munkiStore := munki.NewStore(db)

	// Santa stores.
	santaHostStore := santa.NewStore(db)
	configurationStore := configurations.NewStore(db)
	eventStore := events.NewStore(db)
	ruleStore := rules.NewStore(db)
	referenceStore := references.NewStore(db)
	syncStore := syncstate.NewStore(db)

	users := directory.NewUserService(directoryStore)
	authn := newAuth(ctx, cfg, users, sessions, logger)

	orbitAgent := orbit.NewService(hostStore, secretStore, userAffinities)

	liveQueries := livequery.NewManager()

	inventoryProjector := ingest.NewProjector(
		hostStore,
		softwareStore,
		logger.With("component", "inventory"),
	).WithMunkiStore(munkiStore)

	labelEvaluator := ingest.NewLabelEvaluator(
		labelStore,
		logger.With("component", "labels"),
	)

	osqueryAgent := osquery.NewService(osquery.Dependencies{
		HostStore:          hostStore,
		InventoryProjector: inventoryProjector,
		LabelEvaluator:     labelEvaluator,
		ReportStore:        reportStore,
		CheckStore:         checkStore,
		LiveQueries:        liveQueries,
		SecretStore:        secretStore,
		Logger:             logger.With("component", "osquery"),
	})

	munkiRepo, munkiArtifacts := newMunki(ctx, cfg, hostStore, munkiStore, logger)

	santaSync := santa.NewService(santa.Dependencies{
		HostStore:      santaHostStore,
		Configurations: configurationStore,
		UserAffinities: userAffinities,
		Events:         eventStore,
		Rules:          ruleStore,
		Sync:           syncStore,
	})

	santaState := santa.NewHostStateService(santaHostStore, configurationStore)

	server := admin.NewServer(admin.Dependencies{
		Runtime: admin.RuntimeDependencies{
			Config:         cfg,
			DB:             db,
			Version:        buildinfo.Version,
			Logger:         logger,
			SessionManager: sessions,
			WebHandler: web.NewHandler(web.HandlerOptions{
				FS:        webdist.DistDirFS,
				Version:   buildinfo.Version,
				PublicURL: cfg.PublicURL,
				Logger:    logger.With("component", "web"),
			}),
		},

		Auth: admin.AuthDependencies{
			AuthService: authn,
			UserService: users,
		},

		Directory: admin.DirectoryDependencies{
			Store: directoryStore,
		},

		Inventory: admin.InventoryDependencies{
			Hosts:          hostStore,
			UserAffinities: userAffinities,
			Software:       softwareStore,
			Labels:         labelStore,
		},

		AgentAuth: admin.AgentAuthDependencies{
			Store: secretStore,
		},

		Orbit: admin.OrbitDependencies{
			Agent: orbitAgent,
		},

		Osquery: admin.OsqueryDependencies{
			Agent:       osqueryAgent,
			LiveQueries: liveQueries,
			Reports:     reportStore,
			Checks:      checkStore,
		},

		Munki: admin.MunkiDependencies{
			Repository:      munkiRepo,
			State:           munkiStore,
			ArtifactStorage: munkiArtifacts,
		},

		Santa: admin.SantaDependencies{
			Sync:           santaSync,
			HostState:      santaState,
			Configurations: configurationStore,
			Rules:          ruleStore,
			Events:         eventStore,
			References:     referenceStore,
		},
	})

	starts := []starter{
		santaCleanup(cfg, eventStore, logger),
		entraSync(cfg, directoryStore, labelStore, logger),
	}

	return server, starts
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

func newMunki(
	ctx context.Context,
	cfg config.Config,
	hosts *hosts.Store,
	state *munki.Store,
	logger *slog.Logger,
) (*munki.Service, munkistorage.ArtifactStorage) {
	artifacts, err := munkistorage.NewArtifactStorage(ctx, munkistorage.Config{
		Enabled: cfg.MunkiS3Enabled(),
		S3: munkistorage.S3Config{
			Bucket:         cfg.MunkiS3Bucket,
			Region:         cfg.MunkiS3Region,
			Endpoint:       cfg.MunkiS3Endpoint,
			PublicEndpoint: cfg.MunkiS3PublicEndpoint,
			AccessKey:      cfg.MunkiS3AccessKey,
			SecretKey:      cfg.MunkiS3SecretKey,
			PathStyle:      cfg.MunkiS3PathStyle,
			TTL:            cfg.MunkiS3PresignTTL,
		},
	})
	if err != nil {
		logger.WarnContext(ctx, "munki artifact storage disabled",
			"component", "munki",
			"operation", "artifact-storage",
			"err", err,
		)
	}

	options := []munki.ServiceOption{
		munki.WithArtifactStore(state),
		munki.WithArtifactPresigner(artifacts),
		munki.WithPublicURL(cfg.PublicURL),
	}

	return munki.NewService(hosts, state, options...), artifacts
}

func santaCleanup(
	cfg config.Config,
	store *events.Store,
	logger *slog.Logger,
) starter {
	return func(ctx context.Context) func() {
		cleanup := events.StartCleanup(ctx, store, events.CleanupOptions{
			RetentionDays: cfg.SantaEventRetentionDays,
			SweepInterval: cfg.SantaEventSweepInterval,
		}, logger.With("component", "santa"))

		if cleanup == nil {
			return nil
		}

		return cleanup.Stop
	}
}

func entraSync(
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
