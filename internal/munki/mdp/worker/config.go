// Package worker runs the `woodstar mdp` face: it mirrors the desired Munki
// package installers from Woodstar, verifies them, reports its state, and serves
// them under per-DP grants. It never touches the database or the server domain;
// it shares only the grant leaf and talks to Woodstar over a WebSocket plus HTTP
// downloads.
package worker

import (
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config is the worker's runtime settings, all under the WOODSTAR_MDP_ prefix.
// It carries no database, storage, or session configuration: the worker has
// none of those concerns.
type Config struct {
	ServerURL           string `env:"SERVER_URL,required,notEmpty"`
	Key                 string `env:"KEY,required,notEmpty"`
	DataDir             string `env:"DATA_DIR,required,notEmpty"`
	ListenAddr          string `env:"LISTEN_ADDR"                  envDefault:":8080"`
	LogLevel            string `env:"LOG_LEVEL"                    envDefault:"info"`
	LogFormat           string `env:"LOG_FORMAT"                   envDefault:"text"`
	DownloadConcurrency int    `env:"DOWNLOAD_CONCURRENCY"         envDefault:"4"`
}

// LoadConfig reads the worker configuration from the environment and normalizes it.
func LoadConfig() (Config, error) {
	var cfg Config
	if err := env.ParseWithOptions(&cfg, env.Options{Prefix: "WOODSTAR_MDP_"}); err != nil {
		return Config{}, err
	}
	cfg.ServerURL = strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/")
	cfg.DataDir = strings.TrimSpace(cfg.DataDir)
	if cfg.DownloadConcurrency < 1 {
		cfg.DownloadConcurrency = 1
	}
	return cfg, nil
}
