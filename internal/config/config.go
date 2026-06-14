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

	SantaEventRetentionDays int           `env:"SANTA_EVENT_RETENTION_DAYS" envDefault:"90"`
	SantaEventSweepInterval time.Duration `env:"SANTA_EVENT_SWEEP_INTERVAL" envDefault:"1h"`

	// OIDC is capability-gated: SSO endpoints only mount when IssuerURL,
	// ClientID, and ClientSecret are all set.
	OIDCIssuerURL    string   `env:"OIDC_ISSUER_URL"`
	OIDCClientID     string   `env:"OIDC_CLIENT_ID"`
	OIDCClientSecret string   `env:"OIDC_CLIENT_SECRET"`
	OIDCScopes       []string `env:"OIDC_SCOPES"        envDefault:"openid,email,profile"`
	OIDCEmailClaim   string   `env:"OIDC_EMAIL_CLAIM"   envDefault:"email"`

	// Entra sync is capability-gated: TenantID, ClientID, and ClientSecret
	// must all be set for the sync loop to start.
	EntraTenantID         string        `env:"ENTRA_TENANT_ID"`
	EntraClientID         string        `env:"ENTRA_CLIENT_ID"`
	EntraClientSecret     string        `env:"ENTRA_CLIENT_SECRET"`
	EntraTransitiveGroups bool          `env:"ENTRA_TRANSITIVE_GROUPS"`
	EntraSyncInterval     time.Duration `env:"ENTRA_SYNC_INTERVAL"     envDefault:"1h"`

	StorageKind             string        `env:"STORAGE_KIND"               envDefault:"file"`
	StorageFileRoot         string        `env:"STORAGE_FILE_ROOT"          envDefault:"data/storage"`
	StorageS3Bucket         string        `env:"STORAGE_S3_BUCKET"`
	StorageS3Region         string        `env:"STORAGE_S3_REGION"`
	StorageS3Endpoint       string        `env:"STORAGE_S3_ENDPOINT"`
	StorageS3PublicEndpoint string        `env:"STORAGE_S3_PUBLIC_ENDPOINT"`
	StorageS3AccessKey      string        `env:"STORAGE_S3_ACCESS_KEY"`
	StorageS3SecretKey      string        `env:"STORAGE_S3_SECRET_KEY"`
	StorageS3PathStyle      bool          `env:"STORAGE_S3_PATH_STYLE"`
	StorageS3PresignTTL     time.Duration `env:"STORAGE_S3_PRESIGN_TTL"     envDefault:"15m"`

	publicURLScheme string
}

// OIDCEnabled reports whether the required OIDC settings are present.
func (cfg *Config) OIDCEnabled() bool {
	return cfg.OIDCIssuerURL != "" && cfg.OIDCClientID != "" && cfg.OIDCClientSecret != ""
}

// EntraEnabled reports whether the required Entra sync settings are present.
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
	if err := cfg.normalizeStorage(); err != nil {
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

func (cfg *Config) normalizeStorage() error {
	cfg.StorageKind = strings.TrimSpace(cfg.StorageKind)
	cfg.StorageFileRoot = strings.TrimSpace(cfg.StorageFileRoot)
	cfg.StorageS3Bucket = strings.TrimSpace(cfg.StorageS3Bucket)
	cfg.StorageS3Region = strings.TrimSpace(cfg.StorageS3Region)
	cfg.StorageS3Endpoint = strings.TrimSpace(cfg.StorageS3Endpoint)
	cfg.StorageS3PublicEndpoint = strings.TrimSpace(cfg.StorageS3PublicEndpoint)
	cfg.StorageS3AccessKey = strings.TrimSpace(cfg.StorageS3AccessKey)
	cfg.StorageS3SecretKey = strings.TrimSpace(cfg.StorageS3SecretKey)
	switch cfg.StorageKind {
	case "file":
		if cfg.StorageFileRoot == "" {
			return errors.New("WOODSTAR_STORAGE_FILE_ROOT is required when WOODSTAR_STORAGE_KIND=file")
		}
	case "s3":
		if cfg.StorageS3Bucket == "" || cfg.StorageS3Region == "" ||
			cfg.StorageS3AccessKey == "" || cfg.StorageS3SecretKey == "" {
			return errors.New("incomplete WOODSTAR_STORAGE_S3 configuration")
		}
		if cfg.StorageS3PresignTTL <= 0 {
			return errors.New("WOODSTAR_STORAGE_S3_PRESIGN_TTL must be positive")
		}
	default:
		return fmt.Errorf("WOODSTAR_STORAGE_KIND must be file or s3, got %q", cfg.StorageKind)
	}
	return nil
}
