package config

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/caarlos0/env/v11"
)

const storageCapabilityKeyHexLength = 64

func setValidEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("WOODSTAR_URL", "https://localhost:8080")
	t.Setenv("WOODSTAR_DATABASE_URL", "postgres://woodstar:woodstar@localhost:5432/woodstar")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))
}

func resolveConfig(cfg *Config) error {
	if err := ApplyEnvironment(cfg); err != nil {
		return err
	}
	cfg.Normalize()
	return cfg.Validate()
}

func TestApplyEnvironmentPreservesResolvedValues(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_HOST", "0.0.0.0")
	t.Setenv("WOODSTAR_PORT", "8080")
	t.Setenv("WOODSTAR_URL", "https://environment.example")
	t.Setenv("WOODSTAR_DATABASE_URL", "postgres://environment")

	cfg := Config{
		Host:                 "127.0.0.1",
		Port:                 9443,
		ServerURL:            "https://cli.example",
		DatabaseURL:          "postgres://cli",
		StorageCapabilityKey: strings.Repeat("c", storageCapabilityKeyHexLength),
	}
	if err := ApplyEnvironment(&cfg); err != nil {
		t.Fatalf("ApplyEnvironment: %v", err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != 9443 || cfg.ServerURL != "https://cli.example" ||
		cfg.DatabaseURL != "postgres://cli" {
		t.Fatalf("environment replaced resolved values: %#v", cfg)
	}
}

func TestApplyEnvironmentReturnsAggregateParseErrors(t *testing.T) {
	t.Setenv("WOODSTAR_PORT", "not-a-port")
	t.Setenv("WOODSTAR_SANTA_EVENT_RETENTION_DAYS", "not-a-day-count")

	err := ApplyEnvironment(&Config{})
	var aggregate env.AggregateError
	if !errors.As(err, &aggregate) {
		t.Fatalf("ApplyEnvironment error = %T, want env.AggregateError", err)
	}
	if len(aggregate.Errors) != 2 {
		t.Fatalf("aggregate errors = %d, want 2", len(aggregate.Errors))
	}
}

func TestConfigValidateUsesGoFieldNames(t *testing.T) {
	cfg := validConfig()
	cfg.StorageCapabilityKey = ""

	err := cfg.Validate()
	if err == nil || err.Error() != "StorageCapabilityKey is required" {
		t.Fatalf("Validate error = %v, want Go field name", err)
	}
}

func TestConfigRequiresStorageCapabilityKey(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_HOST", "")
	t.Setenv("WOODSTAR_PORT", "")
	t.Setenv("WOODSTAR_URL", "https://localhost:8080")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", "")
	t.Setenv("WOODSTAR_DATABASE_URL", "")
	t.Setenv("WOODSTAR_LOG_LEVEL", "")

	cfg := Config{}

	err := resolveConfig(&cfg)
	if err == nil {
		t.Fatal("resolveConfig returned nil error, want required storage capability key error")
	}
}

func TestConfigRequiresServerURL(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_URL", "")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))

	err := resolveConfig(&Config{})
	if err == nil {
		t.Fatal("resolveConfig returned nil error, want required server URL error")
	}
}

func TestConfigDefaults(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_HOST", "")
	t.Setenv("WOODSTAR_PORT", "")
	t.Setenv("WOODSTAR_URL", "https://localhost:8080")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))
	t.Setenv("WOODSTAR_LOG_LEVEL", "")

	cfg := Config{}

	if err := resolveConfig(&cfg); err != nil {
		t.Fatalf("resolveConfig returned error: %v", err)
	}

	if cfg.Host != "0.0.0.0" {
		t.Fatalf("Host = %q, want 0.0.0.0", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Fatalf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.TLSConfigured() {
		t.Fatal("TLSConfigured = true without certificate and key files")
	}
	if !cfg.SessionCookieSecure {
		t.Fatal("SessionCookieSecure = false, want secure cookies by default")
	}
	if cfg.OIDCRedirectURL != "https://localhost:8080/api/auth/sso/callback" {
		t.Fatalf("OIDCRedirectURL = %q, want server callback URL", cfg.OIDCRedirectURL)
	}
	if len(cfg.StorageCapabilityKey) != storageCapabilityKeyHexLength {
		t.Fatalf(
			"storage capability key length = %d, want %d",
			len(cfg.StorageCapabilityKey),
			storageCapabilityKeyHexLength,
		)
	}
	if cfg.StorageTransferTTL != 15*time.Minute {
		t.Fatalf("StorageTransferTTL = %s, want 15m", cfg.StorageTransferTTL)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want info", cfg.LogLevel)
	}
}

func TestConfigReadsAndNormalizesEnvironment(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_HOST", "127.0.0.1")
	t.Setenv("WOODSTAR_PORT", "9090")
	t.Setenv("WOODSTAR_URL", "https://example.com/")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))
	t.Setenv("WOODSTAR_DATABASE_URL", "postgres://example")
	t.Setenv("WOODSTAR_LOG_LEVEL", "debug")
	t.Setenv("WOODSTAR_TLS_CERT_FILE", "/etc/woodstar/tls.crt")
	t.Setenv("WOODSTAR_TLS_KEY_FILE", "/etc/woodstar/tls.key")

	cfg := Config{}

	err := resolveConfig(&cfg)
	if err != nil {
		t.Fatalf("resolveConfig returned error: %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("Host = %q", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port = %d", cfg.Port)
	}
	if cfg.ServerURL != "https://example.com" {
		t.Fatalf("ServerURL = %q, want https://example.com", cfg.ServerURL)
	}
	if cfg.OIDCRedirectURL != "https://example.com/api/auth/sso/callback" {
		t.Fatalf("OIDCRedirectURL = %q, want derived server callback", cfg.OIDCRedirectURL)
	}
	if !cfg.TLSConfigured() {
		t.Fatal("TLSConfigured = false with certificate and key files")
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestConfigNormalizesCORSAllowedOrigins(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_URL", "https://example.com/")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))
	t.Setenv(
		"WOODSTAR_CORS_ALLOWED_ORIGINS",
		" https://panel.example.com/,https://panel.example.com, http://localhost:5173 ",
	)

	cfg := Config{}
	if err := resolveConfig(&cfg); err != nil {
		t.Fatalf("resolveConfig returned error: %v", err)
	}

	want := []string{"https://panel.example.com", "http://localhost:5173"}
	if strings.Join(cfg.CORSAllowedOrigins, ",") != strings.Join(want, ",") {
		t.Fatalf("CORSAllowedOrigins = %#v, want %#v", cfg.CORSAllowedOrigins, want)
	}
}

func TestConfigRejectsCORSAllowedOriginWithPath(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_URL", "https://example.com/")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))
	t.Setenv("WOODSTAR_CORS_ALLOWED_ORIGINS", "https://panel.example.com/woodstar")

	err := resolveConfig(&Config{})
	if err == nil {
		t.Fatal("resolveConfig returned nil error, want CORS origin path rejection")
	}
}

func TestConfigReadsStorageS3Environment(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_URL", "https://example.com/")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", "")
	t.Setenv("WOODSTAR_STORAGE_KIND", "s3")
	t.Setenv("WOODSTAR_STORAGE_S3_BUCKET", "woodstar")
	t.Setenv("WOODSTAR_STORAGE_S3_REGION", "ap-southeast-2")
	t.Setenv("WOODSTAR_STORAGE_S3_ENDPOINT", " https://storage.example ")
	t.Setenv("WOODSTAR_STORAGE_S3_ACCESS_KEY", "access")
	t.Setenv("WOODSTAR_STORAGE_S3_SECRET_KEY", "secret")
	t.Setenv("WOODSTAR_STORAGE_S3_PATH_STYLE", "true")
	t.Setenv("WOODSTAR_STORAGE_TRANSFER_TTL", "10m")

	cfg := Config{}
	if err := resolveConfig(&cfg); err != nil {
		t.Fatalf("resolveConfig returned error: %v", err)
	}

	if cfg.StorageKind != "s3" {
		t.Fatalf("StorageKind = %q, want s3", cfg.StorageKind)
	}
	if cfg.StorageS3Endpoint != "https://storage.example" {
		t.Fatalf("StorageS3Endpoint = %q", cfg.StorageS3Endpoint)
	}
	if cfg.StorageTransferTTL.String() != "10m0s" {
		t.Fatalf("StorageTransferTTL = %s", cfg.StorageTransferTTL)
	}
}

func TestConfigRejectsPartialStorageS3Config(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_URL", "https://example.com/")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))
	t.Setenv("WOODSTAR_STORAGE_KIND", "s3")
	t.Setenv("WOODSTAR_STORAGE_S3_BUCKET", "woodstar")

	err := resolveConfig(&Config{})
	if err == nil {
		t.Fatal("resolveConfig returned nil error, want partial storage S3 rejection")
	}
}

func TestConfigRequiresHTTPSForS3Endpoint(t *testing.T) {
	for _, tc := range []struct {
		name      string
		endpoint  string
		wantError bool
	}{
		{name: "HTTP endpoint", endpoint: "http://garage:3900", wantError: true},
		{name: "HTTPS endpoint", endpoint: "https://garage.example"},
		{name: "AWS default endpoint"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.StorageKind = "s3"
			cfg.StorageFileRoot = ""
			cfg.StorageS3Bucket = "woodstar"
			cfg.StorageS3Region = "ap-southeast-2"
			cfg.StorageS3AccessKey = "access"
			cfg.StorageS3SecretKey = "secret"
			cfg.StorageS3Endpoint = tc.endpoint
			err := cfg.Validate()
			if (err != nil) != tc.wantError {
				t.Fatalf("Validate error = %v, wantError %t", err, tc.wantError)
			}
		})
	}
}

func TestConfigRejectsNonPositiveStorageTransferTTL(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_URL", "https://example.com/")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))
	t.Setenv("WOODSTAR_STORAGE_TRANSFER_TTL", "0s")

	err := resolveConfig(&Config{})
	if err == nil {
		t.Fatal("resolveConfig returned nil error, want transfer TTL rejection")
	}
}

func TestConfigRejectsServerURLWithPath(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_URL", "https://example.com/woodstar")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))

	err := resolveConfig(&Config{})
	if err == nil {
		t.Fatal("resolveConfig returned nil error, want path rejection")
	}
	if !errors.Is(err, ErrInvalidServerURL) {
		t.Fatalf("err = %v, want ErrInvalidServerURL", err)
	}
}

func TestConfigRejectsHTTPServerURL(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_URL", "http://localhost:8443")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))

	err := resolveConfig(&Config{})
	if err == nil {
		t.Fatal("resolveConfig returned nil error, want HTTP server URL rejection")
	}
	if !errors.Is(err, ErrInvalidServerURL) {
		t.Fatalf("err = %v, want ErrInvalidServerURL", err)
	}
}

func TestConfigAllowsHTTPViteOIDCRedirect(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_URL", "https://localhost:8443/")
	t.Setenv("WOODSTAR_OIDC_REDIRECT_URL", " http://localhost:5173/api/auth/sso/callback ")
	t.Setenv("WOODSTAR_SESSION_COOKIE_SECURE", "false")

	cfg := Config{}
	if err := resolveConfig(&cfg); err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.ServerURL != "https://localhost:8443" {
		t.Fatalf("ServerURL = %q", cfg.ServerURL)
	}
	if cfg.OIDCRedirectURL != "http://localhost:5173/api/auth/sso/callback" {
		t.Fatalf("OIDCRedirectURL = %q", cfg.OIDCRedirectURL)
	}
	if cfg.SessionCookieSecure {
		t.Fatal("SessionCookieSecure = true for HTTP Vite origin")
	}
}

func TestConfigRejectsInvalidOIDCRedirectURL(t *testing.T) {
	tests := []string{
		"http://example.com/api/auth/sso/callback",
		"https://example.com/not-the-callback",
		"https://example.com/api/auth/sso/callback?next=/hosts",
	}
	for _, redirectURL := range tests {
		t.Run(redirectURL, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv("WOODSTAR_OIDC_REDIRECT_URL", redirectURL)
			err := resolveConfig(&Config{})
			if !errors.Is(err, ErrInvalidOIDCRedirectURL) {
				t.Fatalf("resolveConfig error = %v, want ErrInvalidOIDCRedirectURL", err)
			}
		})
	}
}

func TestConfigAllowsOIDCIssuerPath(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_OIDC_ISSUER_URL", " https://login.example.com/tenant/v2.0 ")
	t.Setenv("WOODSTAR_OIDC_CLIENT_ID", "client")
	t.Setenv("WOODSTAR_OIDC_CLIENT_SECRET", "secret")

	cfg := Config{}
	if err := resolveConfig(&cfg); err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.OIDCIssuerURL != "https://login.example.com/tenant/v2.0" {
		t.Fatalf("OIDCIssuerURL = %q", cfg.OIDCIssuerURL)
	}
}

func TestConfigRejectsPartialTLSConfig(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_URL", "https://localhost:8080")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", strings.Repeat("a", storageCapabilityKeyHexLength))
	t.Setenv("WOODSTAR_TLS_CERT_FILE", "/etc/woodstar/tls.crt")
	t.Setenv("WOODSTAR_TLS_KEY_FILE", "")

	err := resolveConfig(&Config{})
	if err == nil {
		t.Fatal("resolveConfig returned nil error, want partial TLS configuration rejection")
	}
}

func TestConfigIgnoresFileCapabilityKeyForS3(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("WOODSTAR_STORAGE_KIND", "s3")
	t.Setenv("WOODSTAR_STORAGE_CAPABILITY_KEY", "not-a-file-capability-key")
	t.Setenv("WOODSTAR_STORAGE_S3_BUCKET", "woodstar")
	t.Setenv("WOODSTAR_STORAGE_S3_REGION", "ap-southeast-2")
	t.Setenv("WOODSTAR_STORAGE_S3_ACCESS_KEY", "access")
	t.Setenv("WOODSTAR_STORAGE_S3_SECRET_KEY", "secret")

	if err := resolveConfig(&Config{}); err != nil {
		t.Fatalf("resolveConfig with unused file capability key: %v", err)
	}
}

func TestConfigRejectsInvalidRuntimeSettings(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "missing database", key: "WOODSTAR_DATABASE_URL", value: ""},
		{name: "log level alias", key: "WOODSTAR_LOG_LEVEL", value: "warning"},
		{name: "partial OIDC", key: "WOODSTAR_OIDC_ISSUER_URL", value: "https://id.example.com"},
		{name: "partial Entra", key: "WOODSTAR_ENTRA_TENANT_ID", value: "tenant"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(test.key, test.value)
			if err := resolveConfig(&Config{}); err == nil {
				t.Fatal("resolveConfig returned nil error")
			}
		})
	}
}
