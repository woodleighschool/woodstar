package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/csrf"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/web"
	webfs "github.com/woodleighschool/woodstar/web"
)

func main() {
	config.InitLogger(buildinfo.Version)

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
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			if err := config.ApplyEnvironment(&cfg); err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			config.ConfigureLogger(cfg.LogLevel)

			db, err := database.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			sessionManager, sessionStore := newSessionManager(db, cfg)
			defer sessionStore.StopCleanup()

			users := models.NewUserStore(db)
			hosts := models.NewHostStore(db)
			deviceMappings := models.NewDeviceMappingStore(db)
			secrets := models.NewSecretStore(db)
			software := models.NewSoftwareStore(db)

			authService := auth.NewService(users, sessionManager)
			orbitService := orbit.NewService(hosts, secrets, deviceMappings)
			osqueryService := osquery.NewService(hosts, software, secrets)

			server := api.NewServer(api.ServerDependencies{
				Config:         cfg,
				DB:             db,
				Version:        buildinfo.Version,
				AuthService:    authService,
				SessionManager: sessionManager,
				HostStore:      hosts,
				DeviceMappings: deviceMappings,
				SecretStore:    secrets,
				SoftwareStore:  software,
				OrbitService:   orbitService,
				OsqueryService: osqueryService,
				WebHandler: web.NewHandler(web.HandlerOptions{
					FS:        webfs.DistDirFS,
					PublicURL: cfg.PublicURL,
					Version:   buildinfo.Version,
					CSRFToken: csrf.Token,
				}),
			})

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

func newSessionManager(db *database.DB, cfg config.Config) (*scs.SessionManager, *pgxstore.PostgresStore) {
	store := pgxstore.New(db.Pool())
	sm := scs.New()
	sm.Store = store
	sm.Lifetime = api.SessionLifetime
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
			log.Debug().Msg("printing version")
			_, _ = cmd.OutOrStdout().Write([]byte(buildinfo.String()))
		},
	}
}
