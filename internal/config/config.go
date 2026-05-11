package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

const minSessionSecretLength = 32

// SessionLifetime is the browser session lifetime.
const SessionLifetime = 14 * 24 * time.Hour

// ErrInvalidPublicURL is returned when WOODSTAR_PUBLIC_URL is not a valid public origin.
var ErrInvalidPublicURL = errors.New("invalid WOODSTAR_PUBLIC_URL")

// Config contains runtime settings.
type Config struct {
	Host          string `env:"HOST"                             envDefault:"0.0.0.0"`
	Port          int    `env:"PORT"                             envDefault:"8080"`
	PublicURL     string `env:"PUBLIC_URL"                       envDefault:"http://localhost:8080"`
	SessionSecret string `env:"SESSION_SECRET,required,notEmpty"`
	DatabaseURL   string `env:"DATABASE_URL"`
	LogLevel      string `env:"LOG_LEVEL"                        envDefault:"info"`

	MaxReportRows           int `env:"MAX_REPORT_ROWS"            envDefault:"1000"`
	LiveQueryTimeoutSeconds int `env:"LIVE_QUERY_TIMEOUT_SECONDS" envDefault:"60"`
	ShutdownTimeoutSeconds  int `env:"SHUTDOWN_TIMEOUT_SECONDS"   envDefault:"15"`

	publicURLScheme string
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
	publicURL, scheme, err := normalizePublicURL(cfg.PublicURL)
	if err != nil {
		return err
	}
	cfg.PublicURL = publicURL
	cfg.publicURLScheme = scheme
	if len(cfg.SessionSecret) < minSessionSecretLength {
		return fmt.Errorf("WOODSTAR_SESSION_SECRET must be at least %d characters", minSessionSecretLength)
	}

	return nil
}

func normalizePublicURL(value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "http://localhost:8080"
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", "", fmt.Errorf("%w: parse URL", ErrInvalidPublicURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", fmt.Errorf("%w: scheme must be http or https", ErrInvalidPublicURL)
	}
	if parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", fmt.Errorf("%w: must include host and omit query or fragment", ErrInvalidPublicURL)
	}
	if path := strings.Trim(parsed.Path, "/"); path != "" {
		return "", "", fmt.Errorf(
			"%w: must not include a path; use a reverse proxy if you need a sub-path",
			ErrInvalidPublicURL,
		)
	}
	parsed.Path = ""
	return strings.TrimRight(parsed.String(), "/"), parsed.Scheme, nil
}

// IsHTTPS reports whether PublicURL uses the https scheme.
func (cfg *Config) IsHTTPS() bool {
	return cfg.publicURLScheme == "https"
}
