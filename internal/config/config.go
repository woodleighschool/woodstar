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

// ErrInvalidPublicURL means the public URL is bogus.
var ErrInvalidPublicURL = errors.New("invalid WOODSTAR_PUBLIC_URL")

// Config contains runtime settings.
type Config struct {
	Host          string `env:"HOST"                             envDefault:"0.0.0.0"`
	Port          int    `env:"PORT"                             envDefault:"8080"`
	PublicURL     string `env:"PUBLIC_URL,required,notEmpty"`
	SessionSecret string `env:"SESSION_SECRET,required,notEmpty"`
	DatabaseURL   string `env:"DATABASE_URL"`
	LogLevel      string `env:"LOG_LEVEL"                        envDefault:"info"`

	ShutdownTimeoutSeconds int `env:"SHUTDOWN_TIMEOUT_SECONDS" envDefault:"15"`

	SantaEventRetentionDays int           `env:"SANTA_EVENT_RETENTION_DAYS" envDefault:"90"`
	SantaEventSweepInterval time.Duration `env:"SANTA_EVENT_SWEEP_INTERVAL" envDefault:"1h"`

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

	MunkiS3Bucket         string        `env:"MUNKI_S3_BUCKET"`
	MunkiS3Region         string        `env:"MUNKI_S3_REGION"`
	MunkiS3Endpoint       string        `env:"MUNKI_S3_ENDPOINT"`
	MunkiS3PublicEndpoint string        `env:"MUNKI_S3_PUBLIC_ENDPOINT"`
	MunkiS3AccessKey      string        `env:"MUNKI_S3_ACCESS_KEY"`
	MunkiS3SecretKey      string        `env:"MUNKI_S3_SECRET_KEY"`
	MunkiS3PathStyle      bool          `env:"MUNKI_S3_PATH_STYLE"`
	MunkiS3PresignTTL     time.Duration `env:"MUNKI_S3_PRESIGN_TTL"     envDefault:"15m"`

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

// MunkiS3Enabled reports whether Munki artifact redirects can use S3.
func (cfg *Config) MunkiS3Enabled() bool {
	return cfg.MunkiS3Bucket != "" &&
		cfg.MunkiS3Region != "" &&
		cfg.MunkiS3AccessKey != "" &&
		cfg.MunkiS3SecretKey != ""
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
	if err := cfg.normalizeMunkiS3(); err != nil {
		return err
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

func (cfg *Config) normalizeMunkiS3() error {
	cfg.MunkiS3Bucket = strings.TrimSpace(cfg.MunkiS3Bucket)
	cfg.MunkiS3Region = strings.TrimSpace(cfg.MunkiS3Region)
	cfg.MunkiS3Endpoint = strings.TrimSpace(cfg.MunkiS3Endpoint)
	cfg.MunkiS3PublicEndpoint = strings.TrimSpace(cfg.MunkiS3PublicEndpoint)
	cfg.MunkiS3AccessKey = strings.TrimSpace(cfg.MunkiS3AccessKey)
	cfg.MunkiS3SecretKey = strings.TrimSpace(cfg.MunkiS3SecretKey)
	if !cfg.munkiS3Configured() {
		return nil
	}
	if !cfg.MunkiS3Enabled() {
		return errors.New("incomplete WOODSTAR_MUNKI_S3 configuration")
	}
	if cfg.MunkiS3PresignTTL <= 0 {
		return errors.New("WOODSTAR_MUNKI_S3_PRESIGN_TTL must be positive")
	}
	return nil
}

func (cfg *Config) munkiS3Configured() bool {
	return cfg.MunkiS3Bucket != "" ||
		cfg.MunkiS3Region != "" ||
		cfg.MunkiS3Endpoint != "" ||
		cfg.MunkiS3PublicEndpoint != "" ||
		cfg.MunkiS3AccessKey != "" ||
		cfg.MunkiS3SecretKey != ""
}
