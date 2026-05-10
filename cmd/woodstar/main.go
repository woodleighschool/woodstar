package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/csrf"
	"github.com/spf13/cobra"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/db"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/logging"
	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/queries"
	"github.com/woodleighschool/woodstar/internal/software"
	"github.com/woodleighschool/woodstar/internal/transport"
	"github.com/woodleighschool/woodstar/internal/web"
	webfs "github.com/woodleighschool/woodstar/web"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "woodstar",
		Short: "Woodstar macOS observability and admin server",
	}
	rootCmd.Version = buildinfo.Version
	rootCmd.AddCommand(runServeCommand())
	rootCmd.AddCommand(runVersionCommand())
	rootCmd.AddCommand(runOpenAPICommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServeCommand() *cobra.Command {
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
	logger.InfoContext(parent, "woodstar configuration loaded", "component", "config", "operation", "load")

	database, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()
	logger.InfoContext(parent, "database ready", "component", "database", "operation", "open")

	sessionManager, sessionStore := newSessionManager(database, cfg)
	defer sessionStore.StopCleanup()

	return runServer(ctx, newServer(ctx, cfg, database, sessionManager, logger))
}

func runServer(ctx context.Context, server *transport.Server) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func newServer(
	ctx context.Context,
	cfg config.Config,
	db *db.DB,
	sessionManager *scs.SessionManager,
	logger *slog.Logger,
) *transport.Server {
	stores := newModelStores(db)
	authService := auth.NewService(stores.userStore, sessionManager)
	orbitService := orbit.NewService(stores.hostStore, stores.secretStore, stores.deviceMappingStore)
	hub := queries.NewHub()
	liveQueries := queries.NewLiveQueryManager(hub, time.Duration(cfg.LiveQueryTimeoutSeconds)*time.Second)
	inventoryProjector := inventory.NewProjector(
		stores.hostStore,
		stores.softwareStore,
		logger.With("component", "inventory"),
	)
	osqueryService := osquery.NewService(
		stores.hostStore,
		inventoryProjector,
		stores.labelStore,
		stores.queryStore,
		stores.checkStore,
		liveQueries,
		stores.secretStore,
		logger.With("component", "osquery"),
	)
	queries.StartCleanup(ctx, stores.queryStore, queries.CleanupOptions{
		MaxReportRows: cfg.MaxReportRows,
	}, logger.With("component", "queries"))

	return transport.NewServer(transport.Dependencies{
		Config:           cfg,
		DB:               db,
		Version:          buildinfo.Version,
		Logger:           logger,
		AuthService:      authService,
		SessionManager:   sessionManager,
		HostStore:        stores.hostStore,
		DeviceMappings:   stores.deviceMappingStore,
		SecretStore:      stores.secretStore,
		SoftwareStore:    stores.softwareStore,
		LabelStore:       stores.labelStore,
		QueryStore:       stores.queryStore,
		CheckStore:       stores.checkStore,
		LiveQueryManager: liveQueries,
		OrbitService:     orbitService,
		OsqueryService:   osqueryService,
		WebHandler: web.NewHandler(web.HandlerOptions{
			FS:        webfs.DistDirFS,
			Version:   buildinfo.Version,
			CSRFToken: csrf.Token,
			Logger:    logger.With("component", "web"),
		}),
	})
}

type modelStores struct {
	userStore          *models.UserStore
	hostStore          *hosts.HostStore
	deviceMappingStore *hosts.DeviceMappingStore
	secretStore        *models.SecretStore
	softwareStore      *software.SoftwareStore
	labelStore         *labels.LabelStore
	queryStore         *queries.QueryStore
	checkStore         *queries.CheckStore
}

func newModelStores(db *db.DB) modelStores {
	return modelStores{
		userStore:          models.NewUserStore(db),
		hostStore:          hosts.NewHostStore(db),
		deviceMappingStore: hosts.NewDeviceMappingStore(db),
		secretStore:        models.NewSecretStore(db),
		softwareStore:      software.NewSoftwareStore(db),
		labelStore:         labels.NewLabelStore(db),
		queryStore:         queries.NewQueryStore(db),
		checkStore:         queries.NewCheckStore(db),
	}
}

func newSessionManager(db *db.DB, cfg config.Config) (*scs.SessionManager, *pgxstore.PostgresStore) {
	store := pgxstore.New(db.Pool())
	sm := scs.New()
	sm.Store = store
	sm.Lifetime = transport.SessionLifetime
	sm.Cookie.Name = "woodstar_session"
	sm.Cookie.Path = "/"
	sm.Cookie.HttpOnly = true
	sm.Cookie.Secure = cfg.IsHTTPS()
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Persist = true
	return sm, store
}

func runVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version",
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = cmd.OutOrStdout().Write([]byte(buildinfo.String()))
		},
	}
}
