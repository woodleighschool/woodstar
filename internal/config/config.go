package config

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/woodleighschool/woodstar/internal/validation"
)

const oidcCallbackPath = "/api/auth/sso/callback"

// SessionLifetime is the browser session lifetime.
const SessionLifetime = 14 * 24 * time.Hour

// ErrInvalidServerURL is returned when ServerURL is not a valid HTTPS origin.
var ErrInvalidServerURL = errors.New("invalid server URL")

// ErrInvalidOIDCRedirectURL is returned when OIDCRedirectURL cannot reach Woodstar's callback.
var ErrInvalidOIDCRedirectURL = errors.New("invalid OIDC redirect URL")

// Config contains runtime settings.
type Config struct {
	Host                string   `env:"HOST"                  envDefault:"0.0.0.0" validate:"required"`
	Port                int      `env:"PORT"                  envDefault:"8080"    validate:"gte=1,lte=65535"`
	ServerURL           string   `env:"URL"                                        validate:"required"`
	TLSCertFile         string   `env:"TLS_CERT_FILE"                              validate:"required_with=TLSKeyFile"`
	TLSKeyFile          string   `env:"TLS_KEY_FILE"                               validate:"required_with=TLSCertFile"`
	SessionCookieSecure bool     `env:"SESSION_COOKIE_SECURE" envDefault:"true"`
	DatabaseURL         string   `env:"DATABASE_URL"                               validate:"required"`
	LogLevel            string   `env:"LOG_LEVEL"             envDefault:"info"    validate:"required,oneof=debug info warn error"`
	CORSAllowedOrigins  []string `env:"CORS_ALLOWED_ORIGINS"                       validate:"dive,web_origin"`

	SantaEventRetentionDays int           `env:"SANTA_EVENT_RETENTION_DAYS" envDefault:"90" validate:"gte=1"`
	SantaEventSweepInterval time.Duration `env:"SANTA_EVENT_SWEEP_INTERVAL" envDefault:"1h" validate:"gt=0"`

	// OIDC is capability-gated: SSO endpoints only mount when IssuerURL,
	// ClientID, and ClientSecret are all set.
	OIDCIssuerURL    string   `env:"OIDC_ISSUER_URL"    validate:"required_with=OIDCClientID OIDCClientSecret,omitempty,https_url"`
	OIDCClientID     string   `env:"OIDC_CLIENT_ID"     validate:"required_with=OIDCIssuerURL OIDCClientSecret"`
	OIDCClientSecret string   `env:"OIDC_CLIENT_SECRET" validate:"required_with=OIDCIssuerURL OIDCClientID"`
	OIDCRedirectURL  string   `env:"OIDC_REDIRECT_URL"`
	OIDCScopes       []string `env:"OIDC_SCOPES"        validate:"min=1,dive,required"                                             envDefault:"openid,email,profile"`
	OIDCEmailClaim   string   `env:"OIDC_EMAIL_CLAIM"   validate:"required"                                                        envDefault:"email"`

	// Entra sync is capability-gated: TenantID, ClientID, and ClientSecret
	// must all be set for the sync loop to start.
	EntraTenantID         string        `env:"ENTRA_TENANT_ID"         validate:"required_with=EntraClientID EntraClientSecret"`
	EntraClientID         string        `env:"ENTRA_CLIENT_ID"         validate:"required_with=EntraTenantID EntraClientSecret"`
	EntraClientSecret     string        `env:"ENTRA_CLIENT_SECRET"     validate:"required_with=EntraTenantID EntraClientID"`
	EntraTransitiveGroups bool          `env:"ENTRA_TRANSITIVE_GROUPS"`
	EntraSyncInterval     time.Duration `env:"ENTRA_SYNC_INTERVAL"     validate:"gt=0"                                          envDefault:"1h"`

	StorageKind             string        `env:"STORAGE_KIND"               envDefault:"file"         validate:"required,oneof=file s3"`
	StorageFileRoot         string        `env:"STORAGE_FILE_ROOT"          envDefault:"data/storage" validate:"required_if=StorageKind file"`
	StorageCapabilityKey    string        `env:"STORAGE_CAPABILITY_KEY"                               validate:"required_if=StorageKind file"`
	StorageTransferTTL      time.Duration `env:"STORAGE_TRANSFER_TTL"       envDefault:"15m"          validate:"gt=0"`
	StorageS3Bucket         string        `env:"STORAGE_S3_BUCKET"                                    validate:"required_if=StorageKind s3"`
	StorageS3Region         string        `env:"STORAGE_S3_REGION"                                    validate:"required_if=StorageKind s3"`
	StorageS3Endpoint       string        `env:"STORAGE_S3_ENDPOINT"                                  validate:"omitempty,url"`
	StorageS3PublicEndpoint string        `env:"STORAGE_S3_PUBLIC_ENDPOINT"                           validate:"omitempty,url"`
	StorageS3AccessKey      string        `env:"STORAGE_S3_ACCESS_KEY"                                validate:"required_if=StorageKind s3"`
	StorageS3SecretKey      string        `env:"STORAGE_S3_SECRET_KEY"                                validate:"required_if=StorageKind s3"`
	StorageS3PathStyle      bool          `env:"STORAGE_S3_PATH_STYLE"`

	// ClientIPSource selects how the real client IP is derived behind proxies.
	// The companion fields are required only for the matching source.
	ClientIPSource         ClientIPSource `env:"HTTP_CLIENT_IP_SOURCE"              envDefault:"remote_addr" validate:"required,oneof=remote_addr xff_trusted_cidrs xff_trusted_proxies header"`
	ClientIPTrustedCIDRs   []string       `env:"HTTP_CLIENT_IP_TRUSTED_CIDRS"                                validate:"excluded_unless=ClientIPSource xff_trusted_cidrs,required_if=ClientIPSource xff_trusted_cidrs,dive,cidr"`
	ClientIPTrustedProxies int            `env:"HTTP_CLIENT_IP_TRUSTED_PROXY_COUNT"                          validate:"excluded_unless=ClientIPSource xff_trusted_proxies,required_if=ClientIPSource xff_trusted_proxies,omitempty,gte=1"`
	ClientIPHeader         string         `env:"HTTP_CLIENT_IP_HEADER"                                       validate:"excluded_unless=ClientIPSource header,required_if=ClientIPSource header"`
}

// ClientIPSource is how the server derives the real client IP behind proxies.
type ClientIPSource string

const (
	// ClientIPSourceRemoteAddr trusts the connection's remote address.
	ClientIPSourceRemoteAddr ClientIPSource = "remote_addr"
	// ClientIPSourceXFFTrustedCIDRs walks X-Forwarded-For, skipping trusted prefixes.
	ClientIPSourceXFFTrustedCIDRs ClientIPSource = "xff_trusted_cidrs"
	// ClientIPSourceXFFTrustedProxies takes the IP a fixed proxy count from the right of X-Forwarded-For.
	ClientIPSourceXFFTrustedProxies ClientIPSource = "xff_trusted_proxies"
	// ClientIPSourceHeader reads a single trusted header set by the proxy.
	ClientIPSourceHeader ClientIPSource = "header"
)

// OIDCEnabled reports whether the required OIDC settings are present.
func (cfg *Config) OIDCEnabled() bool {
	return cfg.OIDCIssuerURL != "" && cfg.OIDCClientID != "" && cfg.OIDCClientSecret != ""
}

// EntraEnabled reports whether the required Entra sync settings are present.
func (cfg *Config) EntraEnabled() bool {
	return cfg.EntraTenantID != "" && cfg.EntraClientID != "" && cfg.EntraClientSecret != ""
}

// ApplyEnvironment fills unset config fields from environment variables and defaults.
func ApplyEnvironment(cfg *Config) error {
	return env.ParseWithOptions(cfg, env.Options{
		Prefix:                       "WOODSTAR_",
		SetDefaultsForZeroValuesOnly: true,
	})
}

// Normalize canonicalizes the resolved configuration without validating it.
func (cfg *Config) Normalize() {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.ServerURL = normalizeOrigin(cfg.ServerURL)
	cfg.OIDCRedirectURL = strings.TrimSpace(cfg.OIDCRedirectURL)
	if cfg.OIDCRedirectURL == "" && cfg.ServerURL != "" {
		cfg.OIDCRedirectURL = cfg.ServerURL + oidcCallbackPath
	}
	cfg.TLSCertFile = strings.TrimSpace(cfg.TLSCertFile)
	cfg.TLSKeyFile = strings.TrimSpace(cfg.TLSKeyFile)
	cfg.LogLevel = strings.ToLower(strings.TrimSpace(cfg.LogLevel))
	cfg.OIDCIssuerURL = strings.TrimSpace(cfg.OIDCIssuerURL)
	cfg.OIDCClientID = strings.TrimSpace(cfg.OIDCClientID)
	cfg.OIDCScopes = normalizeStrings(cfg.OIDCScopes)
	cfg.OIDCEmailClaim = strings.TrimSpace(cfg.OIDCEmailClaim)
	cfg.EntraTenantID = strings.TrimSpace(cfg.EntraTenantID)
	cfg.EntraClientID = strings.TrimSpace(cfg.EntraClientID)
	cfg.normalizeStorage()
	cfg.normalizeCORSAllowedOrigins()
	cfg.normalizeClientIP()
}

// Validate checks the resolved configuration independently of its input sources.
func (cfg *Config) Validate() error {
	if !validation.IsHTTPSOrigin(cfg.ServerURL) {
		return fmt.Errorf("%w: must be an HTTPS origin", ErrInvalidServerURL)
	}
	if !validOIDCRedirectURL(cfg.OIDCRedirectURL) {
		return fmt.Errorf(
			"%w: must be an HTTPS URL or loopback HTTP URL ending in %s",
			ErrInvalidOIDCRedirectURL,
			oidcCallbackPath,
		)
	}
	if err := validation.Struct(cfg); err != nil {
		return err
	}
	publicStorageEndpoint := cfg.StorageS3PublicEndpoint
	if publicStorageEndpoint == "" {
		publicStorageEndpoint = cfg.StorageS3Endpoint
	}
	if cfg.StorageKind == "s3" && publicStorageEndpoint != "" &&
		!validation.IsHTTPSOrigin(publicStorageEndpoint) {
		return errors.New("StorageS3PublicEndpoint must resolve to an HTTPS origin")
	}
	return nil
}

func validOIDCRedirectURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" ||
		parsed.ForceQuery || parsed.Fragment != "" || parsed.Path != oidcCallbackPath {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	if parsed.Scheme != "http" {
		return false
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	addr, err := netip.ParseAddr(host)
	return err == nil && addr.IsLoopback()
}

func (cfg *Config) normalizeCORSAllowedOrigins() {
	if len(cfg.CORSAllowedOrigins) == 0 {
		return
	}
	normalized := make([]string, 0, len(cfg.CORSAllowedOrigins))
	seen := make(map[string]struct{}, len(cfg.CORSAllowedOrigins))
	for _, raw := range cfg.CORSAllowedOrigins {
		origin := normalizeOrigin(raw)
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		normalized = append(normalized, origin)
	}
	cfg.CORSAllowedOrigins = normalized
}

func (cfg *Config) normalizeClientIP() {
	cfg.ClientIPSource = ClientIPSource(strings.TrimSpace(string(cfg.ClientIPSource)))
	cfg.ClientIPHeader = strings.TrimSpace(cfg.ClientIPHeader)
	for i := range cfg.ClientIPTrustedCIDRs {
		cfg.ClientIPTrustedCIDRs[i] = strings.TrimSpace(cfg.ClientIPTrustedCIDRs[i])
	}
}

func normalizeOrigin(value string) string {
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

// TLSConfigured reports whether Woodstar should terminate TLS itself.
func (cfg *Config) TLSConfigured() bool {
	return cfg.TLSCertFile != ""
}

func (cfg *Config) normalizeStorage() {
	cfg.StorageKind = strings.ToLower(strings.TrimSpace(cfg.StorageKind))
	cfg.StorageFileRoot = strings.TrimSpace(cfg.StorageFileRoot)
	cfg.StorageS3Bucket = strings.TrimSpace(cfg.StorageS3Bucket)
	cfg.StorageS3Region = strings.TrimSpace(cfg.StorageS3Region)
	cfg.StorageS3Endpoint = normalizeOrigin(cfg.StorageS3Endpoint)
	cfg.StorageS3PublicEndpoint = normalizeOrigin(cfg.StorageS3PublicEndpoint)
	cfg.StorageS3AccessKey = strings.TrimSpace(cfg.StorageS3AccessKey)
}

func normalizeStrings(values []string) []string {
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values
}
