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
	PublicURL     string `env:"PUBLIC_URL,required,notEmpty"`
	SessionSecret string `env:"SESSION_SECRET,required,notEmpty"`
	DatabaseURL   string `env:"DATABASE_URL"`
	LogLevel      string `env:"LOG_LEVEL"                        envDefault:"info"`

	MaxReportRows           int `env:"MAX_REPORT_ROWS"            envDefault:"1000"`
	LiveQueryTimeoutSeconds int `env:"LIVE_QUERY_TIMEOUT_SECONDS" envDefault:"60"`
	ShutdownTimeoutSeconds  int `env:"SHUTDOWN_TIMEOUT_SECONDS"   envDefault:"15"`

	// OIDC is capability-gated: SSO endpoints only mount when IssuerURL,
	// ClientID, and ClientSecret are all set.
	OIDCIssuerURL    string   `env:"OIDC_ISSUER_URL"`
	OIDCClientID     string   `env:"OIDC_CLIENT_ID"`
	OIDCClientSecret string   `env:"OIDC_CLIENT_SECRET"`
	OIDCScopes       []string `env:"OIDC_SCOPES"        envDefault:"openid,email,profile"`
	OIDCEmailClaim   string   `env:"OIDC_EMAIL_CLAIM"   envDefault:"email"`

	// Entra directory sync is capability-gated: TenantID, ClientID, and
	// ClientSecret must all be set for the sync loop to start.
	EntraTenantID         string        `env:"ENTRA_TENANT_ID"`
	EntraClientID         string        `env:"ENTRA_CLIENT_ID"`
	EntraClientSecret     string        `env:"ENTRA_CLIENT_SECRET"`
	EntraTransitiveGroups bool          `env:"ENTRA_TRANSITIVE_GROUPS"`
	EntraSyncInterval     time.Duration `env:"ENTRA_SYNC_INTERVAL"     envDefault:"1h"`

	publicURLScheme string
}

// OIDCEnabled reports whether the required OIDC settings are present.
func (cfg *Config) OIDCEnabled() bool {
	return cfg.OIDCIssuerURL != "" && cfg.OIDCClientID != "" && cfg.OIDCClientSecret != ""
}

// EntraEnabled reports whether the required Entra directory settings are present.
func (cfg *Config) EntraEnabled() bool {
	return cfg.EntraTenantID != "" && cfg.EntraClientID != "" && cfg.EntraClientSecret != ""
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
