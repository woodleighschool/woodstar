package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/logging"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/software"
	"github.com/woodleighschool/woodstar/internal/users"
	"github.com/woodleighschool/woodstar/internal/web"
	webfs "github.com/woodleighschool/woodstar/web"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "woodstar",
		Short: "Woodstar macOS observability and admin server",
	}
	rootCmd.Version = buildinfo.Version
	rootCmd.AddCommand(serveCommand())
	rootCmd.AddCommand(versionCommand())
	rootCmd.AddCommand(openAPICommand())

	if err := rootCmd.Execute(); err != nil {
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
	cmd.Flags().StringVar(&cfg.LogLevel, "log-level", "", "log level")
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
	api.InstallHumaErrorHandler(logger)
	logger.InfoContext(parent, "woodstar configuration loaded", "component", "config", "operation", "load")

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	logger.InfoContext(parent, "database ready", "component", "database", "operation", "open")

	sessionManager, sessionStore := newSessionManager(db, cfg)
	// StopCleanup must be deferred after db.Close so it runs first (LIFO); the
	// cleanup goroutine talks to the pool, which must still be open when it stops.
	defer sessionStore.StopCleanup()

	server, stopBackground := newServer(ctx, cfg, db, sessionManager, logger)
	defer stopBackground()
	return runServer(ctx, server, time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second)
}

func runServer(ctx context.Context, server *api.Server, shutdownTimeout time.Duration) error {
	group, ctx := errgroup.WithContext(ctx)

	group.Go(server.ListenAndServe)
	group.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	})

	return group.Wait()
}

func newServer(
	ctx context.Context,
	cfg config.Config,
	db *database.DB,
	sessionManager *scs.SessionManager,
	logger *slog.Logger,
) (*api.Server, func()) {
	stores := newStores(db)
	userService := users.NewService(stores.users)
	authService := newAuthService(ctx, cfg, userService, sessionManager, logger)
	orbitRuntime := newOrbitRuntime(ctx, cfg, stores, logger)
	stopBackground := append([]func(){orbitRuntime.stop}, startIntegrations(ctx, cfg, db, logger)...)

	server := api.NewServer(api.Dependencies{
		Runtime: api.RuntimeDependencies{
			Config:         cfg,
			DB:             db,
			Version:        buildinfo.Version,
			Logger:         logger,
			SessionManager: sessionManager,
			WebHandler: web.NewHandler(web.HandlerOptions{
				FS:      webfs.DistDirFS,
				Version: buildinfo.Version,
				Logger:  logger.With("component", "web"),
			}),
		},
		Auth: api.AuthDependencies{
			AuthService: authService,
			UserService: userService,
		},
		Inventory: api.InventoryDependencies{
			HostStore:     stores.hosts,
			SecretStore:   stores.secrets,
			SoftwareStore: stores.software,
			LabelStore:    stores.labels,
			ReportStore:   stores.reports,
			CheckStore:    stores.checks,
		},
		Orbit: api.OrbitDependencies{
			LiveQueryManager: orbitRuntime.liveQueries,
			Service:          orbitRuntime.orbit,
			OsqueryService:   orbitRuntime.osquery,
		},
		Santa: api.SantaDependencies{
			Store:   stores.santa,
			Service: santa.NewService(stores.santa),
		},
	})
	return server, func() {
		for _, v := range slices.Backward(stopBackground) {
			v()
		}
	}
}

type appStores struct {
	users          *users.Store
	hosts          *hosts.Store
	deviceMappings *hosts.DeviceMappingStore
	secrets        *enrollment.Store
	software       *software.Store
	labels         *labels.Store
	reports        *reports.Store
	checks         *checks.Store
	santa          *santa.Store
}

func newStores(db *database.DB) appStores {
	return appStores{
		users:          users.NewStore(db),
		hosts:          hosts.NewStore(db),
		deviceMappings: hosts.NewDeviceMappingStore(db),
		secrets:        enrollment.NewStore(db),
		software:       software.NewStore(db),
		labels:         labels.NewStore(db),
		reports:        reports.NewStore(db),
		checks:         checks.NewStore(db),
		santa:          santa.NewStore(db),
	}
}

func newAuthService(
	ctx context.Context,
	cfg config.Config,
	userService *users.Service,
	sessionManager *scs.SessionManager,
	logger *slog.Logger,
) *auth.Service {
	authService := auth.NewService(userService, sessionManager)
	if !cfg.OIDCEnabled() {
		return authService
	}
	oidcErr := authService.ConfigureOIDC(ctx, auth.OIDCConfig{
		IssuerURL:    cfg.OIDCIssuerURL,
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.PublicURL + "/api/auth/sso/callback",
		Scopes:       cfg.OIDCScopes,
		EmailClaim:   cfg.OIDCEmailClaim,
	})
	if oidcErr != nil {
		logger.WarnContext(ctx, "sso disabled: oidc discovery failed",
			"component", "auth", "operation", "oidc-configure", "err", oidcErr)
	} else {
		logger.InfoContext(ctx, "sso enabled",
			"component", "auth", "operation", "oidc-configure", "issuer", cfg.OIDCIssuerURL)
	}
	return authService
}

type orbitRuntime struct {
	orbit       *orbit.Service
	osquery     *osquery.Service
	liveQueries *livequery.Manager
	stop        func()
}

func newOrbitRuntime(
	ctx context.Context,
	cfg config.Config,
	stores appStores,
	logger *slog.Logger,
) orbitRuntime {
	liveQueries := livequery.NewManager()
	inventoryProjector := ingest.NewProjector(
		stores.hosts,
		stores.software,
		logger.With("component", "inventory"),
	)
	labelEvaluator := ingest.NewLabelEvaluator(stores.labels, logger.With("component", "labels"))
	osqueryService := osquery.NewService(osquery.Dependencies{
		HostStore:          stores.hosts,
		InventoryProjector: inventoryProjector,
		LabelEvaluator:     labelEvaluator,
		ReportStore:        stores.reports,
		CheckStore:         stores.checks,
		LiveQueries:        liveQueries,
		SecretStore:        stores.secrets,
		Logger:             logger.With("component", "osquery"),
	})
	reportCleanup := reports.StartCleanup(ctx, stores.reports, reports.CleanupOptions{
		MaxReportRows: cfg.MaxReportRows,
	}, logger.With("component", "reports"))
	return orbitRuntime{
		orbit:       orbit.NewService(stores.hosts, stores.secrets, stores.deviceMappings),
		osquery:     osqueryService,
		liveQueries: liveQueries,
		stop:        reportCleanup.Stop,
	}
}

func startIntegrations(ctx context.Context, cfg config.Config, db *database.DB, logger *slog.Logger) []func() {
	if !cfg.EntraEnabled() {
		return nil
	}
	directoryStore := directory.NewStore(db)
	entraClient := directory.NewEntraClient(directory.EntraConfig{
		TenantID:         cfg.EntraTenantID,
		ClientID:         cfg.EntraClientID,
		ClientSecret:     cfg.EntraClientSecret,
		TransitiveGroups: cfg.EntraTransitiveGroups,
	})
	directorySvc := directory.NewService(directoryStore, entraClient, logger.With("component", "directory"))
	return []func(){directorySvc.StartScheduler(ctx, cfg.EntraSyncInterval)}
}

func newSessionManager(db *database.DB, cfg config.Config) (*scs.SessionManager, *pgxstore.PostgresStore) {
	store := pgxstore.New(db.Pool())
	sm := scs.New()
	sm.Store = store
	sm.Lifetime = config.SessionLifetime
	sm.Cookie.Name = "woodstar_session"
	sm.Cookie.Path = "/"
	sm.Cookie.HttpOnly = true
	sm.Cookie.Secure = cfg.IsHTTPS()
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Persist = true
	return sm, store
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version",
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = cmd.OutOrStdout().Write([]byte(buildinfo.String()))
		},
	}
}
