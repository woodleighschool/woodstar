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
	"github.com/woodleighschool/woodstar/internal/buildinfo"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			config.ApplyEnvironment(&cfg)
			config.ConfigureLogger(cfg.LogLevel)

			db, err := database.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			server := api.NewServer(api.Dependencies{
				Config:     cfg,
				DB:         db,
				Version:    buildinfo.Version,
				WebHandler: web.NewHandler(web.Options{FS: webfs.DistDirFS}),
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

	cmd.Flags().StringVar(&cfg.Host, "host", "0.0.0.0", "HTTP listen host")
	cmd.Flags().IntVar(&cfg.Port, "port", 8080, "HTTP listen port")
	cmd.Flags().StringVar(&cfg.DatabaseURL, "database-url", "", "Postgres connection URL")
	cmd.Flags().StringVar(&cfg.LogLevel, "log-level", "info", "log level")

	return cmd
}

func runVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version",
		Run: func(cmd *cobra.Command, args []string) {
			log.Debug().Msg("printing version")
			fmt.Print(buildinfo.String())
		},
	}
}
