// Package worker runs the `woodstar mdp` face: it mirrors the desired Munki
// package installers from Woodstar, verifies them, reports its state, and serves
// them under per-DP grants. It never touches the database or the server domain;
// it shares only the grant leaf and talks to Woodstar over a WebSocket plus HTTPS
// downloads.
package worker

import (
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"

	"github.com/woodleighschool/woodstar/internal/validation"
)

// Config is the worker's runtime settings, all under the WOODSTAR_MDP_ prefix.
// It carries no database, storage, or session configuration: the worker has
// none of those concerns.
type Config struct {
	ServerURL           string `env:"SERVER_URL,required,notEmpty" validate:"https_origin"`
	ServerCAFile        string `env:"SERVER_CA_FILE"`
	Key                 string `env:"KEY,required,notEmpty"        validate:"notblank"`
	DataDir             string `env:"DATA_DIR,required,notEmpty"   validate:"required"`
	ListenAddr          string `env:"LISTEN_ADDR"                  validate:"required"                             envDefault:":8080"`
	TLSCertFile         string `env:"TLS_CERT_FILE"                validate:"required_with=TLSKeyFile"`
	TLSKeyFile          string `env:"TLS_KEY_FILE"                 validate:"required_with=TLSCertFile"`
	LogLevel            string `env:"LOG_LEVEL"                    validate:"required,oneof=debug info warn error" envDefault:"info"`
	DownloadConcurrency int    `env:"DOWNLOAD_CONCURRENCY"         validate:"gte=1"                                envDefault:"4"`
}

// LoadConfig reads, normalizes, and validates the worker configuration.
func LoadConfig() (Config, error) {
	var cfg Config
	if err := env.ParseWithOptions(&cfg, env.Options{Prefix: "WOODSTAR_MDP_"}); err != nil {
		return Config{}, err
	}
	cfg.normalize()
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg *Config) validate() error {
	return validation.Struct(cfg)
}

func (cfg *Config) normalize() {
	cfg.ServerURL = normalizeServerURL(cfg.ServerURL)
	cfg.ServerCAFile = strings.TrimSpace(cfg.ServerCAFile)
	cfg.DataDir = strings.TrimSpace(cfg.DataDir)
	cfg.ListenAddr = strings.TrimSpace(cfg.ListenAddr)
	cfg.TLSCertFile = strings.TrimSpace(cfg.TLSCertFile)
	cfg.TLSKeyFile = strings.TrimSpace(cfg.TLSKeyFile)
	cfg.LogLevel = strings.ToLower(strings.TrimSpace(cfg.LogLevel))
}

func normalizeServerURL(value string) string {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil {
		return value
	}
	if parsed.Path == "/" {
		parsed.Path = ""
	}
	return parsed.String()
}

// TLSConfigured reports whether the worker should terminate TLS itself.
func (cfg *Config) TLSConfigured() bool {
	return cfg.TLSCertFile != ""
}
