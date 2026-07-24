package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/woodleighschool/woodstar/internal/buildinfo"
	"github.com/woodleighschool/woodstar/internal/logging"
	"github.com/woodleighschool/woodstar/internal/munki/mdp/worker"
)

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
	mdp, err := worker.New(cfg, buildinfo.Version, logger)
	if err != nil {
		return fmt.Errorf("init mdp worker: %w", err)
	}
	return mdp.Run(ctx)
}
