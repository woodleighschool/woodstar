package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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

			users := models.NewUserStore(db)
			sessions := models.NewSessionStore(db)
			hosts := models.NewHostStore(db)
			deviceMappings := models.NewDeviceMappingStore(db)
			secrets := models.NewSecretStore(db)
			software := models.NewSoftwareStore(db)

			authService := auth.NewService(users, sessions, api.SessionTTL, cfg.SessionSecret)
			orbitService := orbit.NewService(hosts, secrets, deviceMappings)
			osqueryService := osquery.NewService(hosts, software, secrets)

			server := api.NewServer(api.ServerDependencies{
				Config:         cfg,
				DB:             db,
				Version:        buildinfo.Version,
				AuthService:    authService,
				HostStore:      hosts,
				DeviceMappings: deviceMappings,
				SecretStore:    secrets,
				SoftwareStore:  software,
				OrbitService:   orbitService,
				OsqueryService: osqueryService,
				WebHandler: web.NewHandler(web.HandlerOptions{
					FS:      webfs.DistDirFS,
					BaseURL: cfg.BaseURL,
					Version: buildinfo.Version,
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
	cmd.Flags().StringVar(&cfg.BaseURL, "base-url", "", "Public base URL")
	cmd.Flags().StringVar(&cfg.DatabaseURL, "database-url", "", "Postgres connection URL")
	cmd.Flags().StringVar(&cfg.LogLevel, "log-level", "", "log level")
	cmd.Flags().StringVar(&cfg.SessionSecret, "session-secret", "", "Session signing secret")

	return cmd
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
