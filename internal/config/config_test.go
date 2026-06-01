package config

import (
	"errors"
	"strings"
	"testing"
)

func TestApplyEnvironmentRequiresSessionSecret(t *testing.T) {
	t.Setenv("WOODSTAR_HOST", "")
	t.Setenv("WOODSTAR_PORT", "")
	t.Setenv("WOODSTAR_PUBLIC_URL", "http://localhost:8080")
	t.Setenv("WOODSTAR_SESSION_SECRET", "")
	t.Setenv("WOODSTAR_DATABASE_URL", "")
	t.Setenv("WOODSTAR_LOG_LEVEL", "")

	cfg := Config{}

	err := ApplyEnvironment(&cfg)
	if err == nil {
		t.Fatal("ApplyEnvironment returned nil error, want required session secret error")
	}
}

func TestApplyEnvironmentRequiresPublicURL(t *testing.T) {
	t.Setenv("WOODSTAR_PUBLIC_URL", "")
	t.Setenv("WOODSTAR_SESSION_SECRET", strings.Repeat("s", minSessionSecretLength))

	err := ApplyEnvironment(&Config{})
	if err == nil {
		t.Fatal("ApplyEnvironment returned nil error, want required public URL error")
	}
}

func TestApplyEnvironmentDefaults(t *testing.T) {
	t.Setenv("WOODSTAR_HOST", "")
	t.Setenv("WOODSTAR_PORT", "")
	t.Setenv("WOODSTAR_PUBLIC_URL", "http://localhost:8080")
	t.Setenv("WOODSTAR_SESSION_SECRET", strings.Repeat("s", minSessionSecretLength))
	t.Setenv("WOODSTAR_DATABASE_URL", "")
	t.Setenv("WOODSTAR_LOG_LEVEL", "")

	cfg := Config{}

	if err := ApplyEnvironment(&cfg); err != nil {
		t.Fatalf("ApplyEnvironment returned error: %v", err)
	}

	if cfg.Host != "0.0.0.0" {
		t.Fatalf("Host = %q, want 0.0.0.0", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Fatalf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.IsHTTPS() {
		t.Fatalf("IsHTTPS = true, want false for http URL")
	}
	if len(cfg.SessionSecret) < minSessionSecretLength {
		t.Fatalf("SessionSecret length = %d, want at least %d", len(cfg.SessionSecret), minSessionSecretLength)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.ShutdownTimeoutSeconds != 15 {
		t.Fatalf("ShutdownTimeoutSeconds = %d, want 15", cfg.ShutdownTimeoutSeconds)
	}
}

func TestApplyEnvironmentReadsAndNormalizesConfiguredValues(t *testing.T) {
	t.Setenv("WOODSTAR_HOST", "127.0.0.1")
	t.Setenv("WOODSTAR_PORT", "9090")
	t.Setenv("WOODSTAR_PUBLIC_URL", "https://example.com/")
	t.Setenv("WOODSTAR_SESSION_SECRET", strings.Repeat("s", minSessionSecretLength))
	t.Setenv("WOODSTAR_DATABASE_URL", "postgres://example")
	t.Setenv("WOODSTAR_LOG_LEVEL", "debug")

	cfg := Config{}

	err := ApplyEnvironment(&cfg)
	if err != nil {
		t.Fatalf("ApplyEnvironment returned error: %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("Host = %q", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port = %d", cfg.Port)
	}
	if cfg.PublicURL != "https://example.com" {
		t.Fatalf("PublicURL = %q, want https://example.com", cfg.PublicURL)
	}
	if !cfg.IsHTTPS() {
		t.Fatalf("IsHTTPS = false, want true for https URL")
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestApplyEnvironmentReadsMunkiS3Config(t *testing.T) {
	t.Setenv("WOODSTAR_PUBLIC_URL", "https://example.com/")
	t.Setenv("WOODSTAR_SESSION_SECRET", strings.Repeat("s", minSessionSecretLength))
	t.Setenv("WOODSTAR_MUNKI_S3_BUCKET", "woodstar-munki")
	t.Setenv("WOODSTAR_MUNKI_S3_REGION", "ap-southeast-2")
	t.Setenv("WOODSTAR_MUNKI_S3_ENDPOINT", " https://storage.example ")
	t.Setenv("WOODSTAR_MUNKI_S3_PUBLIC_ENDPOINT", " https://downloads.example ")
	t.Setenv("WOODSTAR_MUNKI_S3_ACCESS_KEY", "access")
	t.Setenv("WOODSTAR_MUNKI_S3_SECRET_KEY", "secret")
	t.Setenv("WOODSTAR_MUNKI_S3_PATH_STYLE", "true")
	t.Setenv("WOODSTAR_MUNKI_S3_PRESIGN_TTL", "10m")

	cfg := Config{}
	if err := ApplyEnvironment(&cfg); err != nil {
		t.Fatalf("ApplyEnvironment returned error: %v", err)
	}

	if !cfg.MunkiS3Enabled() {
		t.Fatal("MunkiS3Enabled = false, want true")
	}
	if cfg.MunkiS3Endpoint != "https://storage.example" {
		t.Fatalf("MunkiS3Endpoint = %q", cfg.MunkiS3Endpoint)
	}
	if cfg.MunkiS3PublicEndpoint != "https://downloads.example" {
		t.Fatalf("MunkiS3PublicEndpoint = %q", cfg.MunkiS3PublicEndpoint)
	}
	if cfg.MunkiS3PresignTTL.String() != "10m0s" {
		t.Fatalf("MunkiS3PresignTTL = %s", cfg.MunkiS3PresignTTL)
	}
}

func TestApplyEnvironmentRejectsPartialMunkiS3Config(t *testing.T) {
	t.Setenv("WOODSTAR_PUBLIC_URL", "https://example.com/")
	t.Setenv("WOODSTAR_SESSION_SECRET", strings.Repeat("s", minSessionSecretLength))
	t.Setenv("WOODSTAR_MUNKI_S3_BUCKET", "woodstar-munki")

	err := ApplyEnvironment(&Config{})
	if err == nil {
		t.Fatal("ApplyEnvironment returned nil error, want partial Munki S3 rejection")
	}
}

func TestApplyEnvironmentRejectsURLWithPath(t *testing.T) {
	t.Setenv("WOODSTAR_PUBLIC_URL", "https://example.com/woodstar")
	t.Setenv("WOODSTAR_SESSION_SECRET", strings.Repeat("s", minSessionSecretLength))

	err := ApplyEnvironment(&Config{})
	if err == nil {
		t.Fatal("ApplyEnvironment returned nil error, want path rejection")
	}
	if !errors.Is(err, ErrInvalidPublicURL) {
		t.Fatalf("err = %v, want ErrInvalidPublicURL", err)
	}
}

func TestApplyEnvironmentRejectsWeakSessionSecret(t *testing.T) {
	t.Setenv("WOODSTAR_PUBLIC_URL", "http://localhost:8080")
	t.Setenv("WOODSTAR_SESSION_SECRET", "too-short")

	err := ApplyEnvironment(&Config{})
	if err == nil {
		t.Fatal("ApplyEnvironment returned nil error, want weak session secret error")
	}
}
