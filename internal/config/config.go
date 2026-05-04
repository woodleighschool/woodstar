package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

const minSessionSecretLength = 32

// Config contains runtime settings.
type Config struct {
	Host          string `env:"HOST"                             envDefault:"0.0.0.0"`
	Port          int    `env:"PORT"                             envDefault:"8080"`
	BaseURL       string `env:"BASE_URL"                         envDefault:"http://localhost:8080"`
	SessionSecret string `env:"SESSION_SECRET,required,notEmpty"`
	DatabaseURL   string `env:"DATABASE_URL"`
	LogLevel      string `env:"LOG_LEVEL"                        envDefault:"info"`
}

// ApplyEnvironment fills cfg from environment variables and normalizes derived values.
func ApplyEnvironment(cfg *Config) error {
	if err := env.ParseWithOptions(cfg, env.Options{
		Prefix:                       "WOODSTAR_",
		SetDefaultsForZeroValuesOnly: true,
	}); err != nil {
		return err
	}
	return cfg.normalize()
}

func (cfg *Config) normalize() error {
	baseURL, err := normalizeBaseURL(cfg.BaseURL)
	if err != nil {
		return err
	}
	cfg.BaseURL = baseURL
	if len(cfg.SessionSecret) < minSessionSecretLength {
		return fmt.Errorf("WOODSTAR_SESSION_SECRET must be at least %d characters", minSessionSecretLength)
	}

	return nil
}

func normalizeBaseURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "http://localhost:8080"
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", errors.New("invalid WOODSTAR_BASE_URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("invalid WOODSTAR_BASE_URL")
	}
	if parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid WOODSTAR_BASE_URL")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

// BasePath returns the mount path from BaseURL.
func (cfg *Config) BasePath() string {
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Path == "" {
		return "/"
	}
	path := strings.TrimRight(parsed.Path, "/")
	if path == "" {
		return "/"
	}
	return path
}
